package deepjoy

import (
	"errors"
	"time"

	"github.com/bradhe/stopwatch"
	"github.com/efritz/backoff"
	"github.com/efritz/glock"
	"github.com/efritz/overcurrent"
)

type (
	// Client is a goroutine-safe, minimal, and pooled Redis client.
	Client interface {
		// Close will close all open connections to the remote Redis server.
		Close()

		// ReadReplica returns a host that points to the read replica. If no
		// read replica is configured, this returns the current client. The
		// client returned from this method does NOT need to be independently
		// closed (closing the source client will also close replica clients).
		ReadReplica() Client

		// Do runs the command on the remote Redis server and returns its raw
		// response.
		Do(command string, args ...interface{}) (interface{}, error)

		// Transaction runs several commands in a single connection. MULTI/EXEC
		// commands are added implicitly by the client.
		Transaction(commands ...Command) (interface{}, error)
	}

	// Command is a struct that bundles the command and the command arguments
	// together to be used in a transaction.
	Command struct {
		Command string
		Args    []interface{}
	}

	client struct {
		readReplicaClient Client
		pool              Pool
		borrowTimeout     *time.Duration
		backoff           backoff.Backoff
		clock             glock.Clock
		logger            Logger
	}

	clientConfig struct {
		dialerFactory  DialerFactory
		readAddr       string
		password       string
		database       int
		connectTimeout time.Duration
		readTimeout    time.Duration
		writeTimeout   time.Duration
		poolCapacity   int
		backoff        backoff.Backoff
		breakerFunc    BreakerFunc
		clock          glock.Clock
		borrowTimeout  *time.Duration
		logger         Logger
	}

	retryableFunc func() (interface{}, error)

	// ConfigFunc is a function used to initialize a new client.
	ConfigFunc func(*clientConfig)
)

var (
	// ErrNoConnection is returned when the borrow timeout elapses.
	ErrNoConnection = errors.New("no connection available in pool")
	defaultBackoff  = backoff.NewLinearBackoff(time.Millisecond, time.Millisecond*250, time.Second*5)
)

// NewClient creates a new Client.
func NewClient(addr string, configs ...ConfigFunc) Client {
	config := &clientConfig{
		connectTimeout: time.Second * 5,
		writeTimeout:   time.Second * 5,
		readTimeout:    time.Second * 5,
		poolCapacity:   10,
		breakerFunc:    noopBreakerFunc,
		backoff:        defaultBackoff,
		clock:          glock.NewRealClock(),
		logger:         &defaultLogger{},
	}

	for _, f := range configs {
		f(config)
	}

	if config.dialerFactory == nil {
		config.dialerFactory = makeDefaultDialerFactory(config)
	}

	return newClient(addr, config.readAddr, config)
}

func newClient(addr, replicaAddr string, config *clientConfig) Client {
	if addr == "" {
		return nil
	}

	pool := NewPool(
		config.dialerFactory(addr),
		config.poolCapacity,
		config.logger,
		config.breakerFunc,
		config.clock,
	)

	return &client{
		pool:              pool,
		readReplicaClient: newClient(replicaAddr, "", config),
		backoff:           config.backoff,
		clock:             config.clock,
		logger:            config.logger,
	}
}

// WithDialerFactory sets the dialer factory to use to create the
// connection pools. Each factory creates connections to a single
// unique address.
func WithDialerFactory(dialerFactory DialerFactory) ConfigFunc {
	return func(c *clientConfig) { c.dialerFactory = dialerFactory }
}

// WithReadReplicaAddr sets the address of the client returned by
// the client's ReadReplica() method.
func WithReadReplicaAddr(addr string) ConfigFunc {
	return func(c *clientConfig) { c.readAddr = addr }
}

// WithPassword sets the password (default is "").
func WithPassword(password string) ConfigFunc {
	return func(c *clientConfig) { c.password = password }
}

// WithDatabase sets the database index (default is 0).
func WithDatabase(database int) ConfigFunc {
	return func(c *clientConfig) { c.database = database }
}

// WithConnectTimeout sets the connect timeout for new connections
// (default is 5 seconds).
func WithConnectTimeout(timeout time.Duration) ConfigFunc {
	return func(c *clientConfig) { c.connectTimeout = timeout }
}

// WithReadTimeout sets the read timeout for all connections in the
// pool (default is 5 seconds).
func WithReadTimeout(timeout time.Duration) ConfigFunc {
	return func(c *clientConfig) { c.readTimeout = timeout }
}

// WithWriteTimeout sets the write timeout for all connections in the
// pool (default is 5 seconds).
func WithWriteTimeout(timeout time.Duration) ConfigFunc {
	return func(c *clientConfig) { c.writeTimeout = timeout }
}

// WithPoolCapacity sets the maximum number of concurrent connections
// that can be in use at once (default is 10).
func WithPoolCapacity(capacity int) ConfigFunc {
	return func(c *clientConfig) { c.poolCapacity = capacity }
}

// WithRetryBackoff sets the circuit backoff prototype to use when
// retrying a redis command after a non-protocol network error.
func WithRetryBackoff(backoff backoff.Backoff) ConfigFunc {
	return func(c *clientConfig) { c.backoff = backoff }
}

// WithBreaker sets the circuit breaker instance to use around new
// connections. The default uses a no-op circuit breaker.
func WithBreaker(breaker overcurrent.CircuitBreaker) ConfigFunc {
	return func(c *clientConfig) { c.breakerFunc = breaker.Call }
}

// WithBreakerRegistry sets the overcurrent registry to use and the
// name of the circuit breaker config tu use around new connections.
// The default uses a no-op circuit breaker.
func WithBreakerRegistry(registry overcurrent.Registry, name string) ConfigFunc {
	return func(c *clientConfig) {
		c.breakerFunc = func(f overcurrent.BreakerFunc) error {
			return registry.Call(name, f, nil)
		}
	}
}

// WithBorrowTimeout sets the maximum time
func WithBorrowTimeout(timeout time.Duration) ConfigFunc {
	return func(c *clientConfig) { c.borrowTimeout = &timeout }
}

// WithLogger sets the logger instance (the default will use Go's
// builtin logging library).
func WithLogger(logger Logger) ConfigFunc {
	return func(c *clientConfig) { c.logger = logger }
}

// NewCommand creates a Command instance.
func NewCommand(command string, args ...interface{}) Command {
	return Command{
		Command: command,
		Args:    args,
	}
}

//
// Client Implementation

func (c *client) ReadReplica() Client {
	if c.readReplicaClient != nil {
		return c.readReplicaClient
	}

	return c
}

func (c *client) Close() {
	if c.readReplicaClient != nil {
		c.readReplicaClient.Close()
	}

	c.pool.Close()
}

func (c *client) Do(command string, args ...interface{}) (interface{}, error) {
	return c.withRetry(func() (interface{}, error) { return c.do(command, args) })
}

func (c *client) Transaction(commands ...Command) (interface{}, error) {
	return c.withRetry(func() (interface{}, error) { return c.transaction(commands...) })
}

//
// Client Helper Functions

func (c *client) withRetry(f retryableFunc) (interface{}, error) {
	// Get a copy of the backoff
	backoff := c.backoff.Clone()

	for {
		result, err := f()

		// Stop retry loop if we either succeeded or encountered a non-recoverable
		// error (non-network error). We don't want to retry protocol or redis logic
		// errors, as the command will likely behave the same way a second time.

		if err == nil {
			return result, nil
		}

		if _, ok := err.(connErr); !ok {
			return result, err
		}

		// Log error here so it's not silently dropped
		c.logger.Printf("Received error from command, retrying (%s)", err.Error())

		// Backoff, don't thrash the pool
		<-c.clock.After(backoff.NextInterval())
	}
}

// Invoke a command and release the connection back to the pool.
func (c *client) do(command string, args []interface{}) (interface{}, error) {
	conn, ok := c.timedBorrow()
	if !ok {
		return nil, ErrNoConnection
	}

	result, err := conn.Do(command, args...)
	c.release(conn, err)
	return result, err
}

// Invoke a series of commands and release the connection back to the pool.
func (c *client) transaction(commands ...Command) (interface{}, error) {
	conn, ok := c.timedBorrow()
	if !ok {
		return nil, ErrNoConnection
	}

	if err := conn.Send("MULTI"); err != nil {
		c.release(conn, err)
		return nil, err
	}

	for _, command := range commands {
		if err := conn.Send(command.Command, command.Args...); err != nil {
			c.release(conn, err)
			return nil, err
		}
	}

	result, err := conn.Do("EXEC")
	c.release(conn, err)
	return result, err
}

// Borrows and logs the time it took to return from blocking on the
// pool's borrow method.
func (c *client) timedBorrow() (Conn, bool) {
	start := stopwatch.Start()
	conn, ok := c.borrow()
	elapsed := stopwatch.Stop(start).Milliseconds()

	if ok {
		c.logger.Printf("Received connection after %vms", elapsed)
	} else {
		c.logger.Printf("Could not borrow connection after %vms", elapsed)
	}

	return conn, ok
}

// Borrows from the pool using the correct method (depending on if
// a borrow timeout was configured on this client).
func (c *client) borrow() (Conn, bool) {
	if c.borrowTimeout == nil {
		return c.pool.Borrow()
	}

	return c.pool.BorrowTimeout(*c.borrowTimeout)
}

// Close the connection on error and release it back to the pool.
// Bad connections never go back to the pool, so in the case that
// there was an error we return nil (if we do not do this on some
// code path then the capacity of the pool permanently decreases).
func (c *client) release(conn Conn, err error) {
	if err != nil {
		if err := conn.Close(); err != nil {
			c.logger.Printf("Could not close connection (%s)", err.Error())
		}

		conn = nil
	}

	c.pool.Release(conn)
}
