package deepjoy

import (
	"errors"
	"io"
	"time"

	"github.com/bradhe/stopwatch"
	"github.com/efritz/glock"
	"github.com/efritz/overcurrent"
	"github.com/garyburd/redigo/redis"
)

type (
	// Client is a goroutine-safe, minimal, and pooled Redis client.
	Client interface {
		// Close will close all open connections to the remote Redis server.
		Close()

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
		pool          Pool
		borrowTimeout *time.Duration
		logger        Logger
	}

	clientConfig struct {
		password       string
		database       int
		connectTimeout time.Duration
		readTimeout    time.Duration
		writeTimeout   time.Duration
		poolCapacity   int
		breakerFunc    BreakerFunc
		clock          glock.Clock
		borrowTimeout  *time.Duration
		logger         Logger
	}

	// ConfigFunc is a function used to initialize a new client.
	ConfigFunc func(*clientConfig)
)

// ErrNoConnection is returned when the borrow timeout elapses.
var ErrNoConnection = errors.New("no connection available in pool")

// NewClient creates a new Client.
func NewClient(addr string, configs ...ConfigFunc) Client {
	config := &clientConfig{
		password:       "",
		database:       0,
		connectTimeout: time.Second * 5,
		writeTimeout:   time.Second * 5,
		readTimeout:    time.Second * 5,
		poolCapacity:   10,
		breakerFunc:    noopBreakerFunc,
		clock:          glock.NewRealClock(),
		borrowTimeout:  nil,
		logger:         &defaultLogger{},
	}

	for _, f := range configs {
		f(config)
	}

	dialer := func() (Conn, error) {
		return redis.Dial(
			"tcp",
			addr,
			redis.DialPassword(config.password),
			redis.DialDatabase(config.database),
			redis.DialConnectTimeout(config.connectTimeout),
			redis.DialReadTimeout(config.readTimeout),
			redis.DialWriteTimeout(config.writeTimeout),
		)
	}

	return &client{
		pool: NewPool(
			dialer,
			config.poolCapacity,
			config.logger,
			config.breakerFunc,
			config.clock,
		),
		logger: config.logger,
	}
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

func withClock(clock glock.Clock) ConfigFunc {
	return func(c *clientConfig) { c.clock = clock }
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

func (c *client) Close() {
	c.pool.Close()
}

func (c *client) Do(command string, args ...interface{}) (interface{}, error) {
	conn, ok := c.timedBorrow()
	if !ok {
		return nil, ErrNoConnection
	}

	result, err := c.doWithConn(conn, command, args)

	if err != nil && shouldRetry(err) {
		// The TCP connection to the remote Redis server may have been
		// reaped by a proxy (depending on your network topology). If
		// we have an IO error, we can try again.
		c.logger.Printf("Connection from pool was stale, retrying")
		return c.Do(command, args...)
	}

	return result, err
}

func (c *client) Transaction(commands ...Command) (interface{}, error) {
	conn, ok := c.timedBorrow()
	if !ok {
		return nil, ErrNoConnection
	}

	if err := conn.Send("MULTI"); err != nil {
		// Ensure connection is released after MULTI error
		c.release(conn, err)

		if shouldRetry(err) {
			c.logger.Printf("Connection from pool was stale, retrying")
			return c.Transaction(commands...)
		}

		return nil, err
	}

	// After this point if we get an error we immediately return. We can't
	// safely retry anything after we've sent the MULTI as pipelined commands
	// aren't really atomic.

	for _, command := range commands {
		if err := conn.Send(command.Command, command.Args...); err != nil {
			c.release(conn, err)
			return nil, err
		}
	}

	return c.doWithConn(conn, "EXEC", nil)
}

//
// Client Helper Functions

// Invoke a command and release the connection back to the pool.
func (c *client) doWithConn(conn Conn, command string, args []interface{}) (interface{}, error) {
	result, err := conn.Do(command, args...)
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
		conn.Close()
		conn = nil
	}

	c.pool.Release(conn)
}

// Given an error, determine if we should try to re-invoke the
// command on another (possibly fresh) connection.
func shouldRetry(err error) bool {
	return err == io.EOF || err == io.ErrUnexpectedEOF
}
