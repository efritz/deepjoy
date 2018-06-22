package deepjoy

import (
	"errors"
	"time"

	"github.com/bradhe/stopwatch"
	"github.com/efritz/backoff"
	"github.com/efritz/glock"
)

type (
	// Client is a goroutine-safe, minimal, and pooled Redis client.
	Client interface {
		// Close will close all open connections to the remote Redis server.
		Close()

		// ReadReplica returns a host that points to the set of configured
		// read replicas. If no read replicas are configured, this returns
		// the current client. The client returned from this method does NOT
		// need to be independently closed (closing the source client will
		// also close replica clients).
		ReadReplica() Client

		// Do runs the command on the remote Redis server and returns its raw
		// response.
		Do(command string, args ...interface{}) (interface{}, error)

		// Pipeline returns a builder object to which commands can be attached.
		// All commands in the pipeline are sent to the remote server in a
		// single request and all results will be returned in a single response.
		// The MULTI/EXEC commands are added implicitly by the client. A pipeline
		// does NOT guarantee atomicity (it is not a transaction in the ACID
		// sense). If you require multiple commands to be run atomically, bundle
		// them in a Lua script and run it on the remote server with the EVAL
		// command.
		Pipeline() Pipeline
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
		readAddrs      []string
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
)

var (
	// ErrNoConnection is returned when the borrow timeout elapses.
	ErrNoConnection = errors.New("no connection available in pool")

	defaultBackoff = backoff.NewLinearBackoff(time.Millisecond, time.Millisecond*250, time.Second*5)
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
		logger:         &nilLogger{},
	}

	for _, f := range configs {
		f(config)
	}

	if config.dialerFactory == nil {
		config.dialerFactory = makeDefaultDialerFactory(config)
	}

	return newClient([]string{addr}, config.readAddrs, config)
}

func newClient(addrs, replicaAddrs []string, config *clientConfig) Client {
	if addrs == nil {
		return nil
	}

	pool := NewPool(
		config.dialerFactory(addrs),
		config.poolCapacity,
		config.logger,
		config.breakerFunc,
		config.clock,
	)

	return &client{
		pool:              pool,
		readReplicaClient: newClient(replicaAddrs, nil, config),
		backoff:           config.backoff,
		clock:             config.clock,
		logger:            config.logger,
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

func (c *client) Pipeline() Pipeline {
	return newPipeline(c)
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

// Invoke a series of commands wrapped in MULTI and EXEC commands
// and release the connection back to the pool. Will retry on error.
func (c *client) pipeline(commands []commandPair) (interface{}, error) {
	return c.withRetry(func() (interface{}, error) { return c.doPipeline(commands) })
}

// Invoke a series of commands wrapped in MULTI and EXEC commands
// and release the connection back to the pool.
func (c *client) doPipeline(commands []commandPair) (interface{}, error) {
	conn, ok := c.timedBorrow()
	if !ok {
		return nil, ErrNoConnection
	}

	if err := conn.Send("MULTI"); err != nil {
		c.release(conn, err)
		return nil, err
	}

	for _, command := range commands {
		if err := conn.Send(command.command, command.args...); err != nil {
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
	watch := stopwatch.Start()
	conn, ok := c.borrow()
	watch.Stop()
	elapsed := watch.Milliseconds()

	if ok {
		c.logger.Printf("Received connection after %v", elapsed)
	} else {
		c.logger.Printf("Could not borrow connection after %v", elapsed)
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
