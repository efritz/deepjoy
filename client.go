package deepjoy

import (
	"errors"
	"io"
	"time"

	"github.com/efritz/glock"

	"github.com/bradhe/stopwatch"
	"github.com/efritz/backoff"
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
		breaker        overcurrent.CircuitBreaker
		clock          glock.Clock
		borrowTimeout  *time.Duration
		logger         Logger
	}

	ClientConfig func(*clientConfig)
)

var ErrNoConnection = errors.New("no connection available in pool")

// NewClient creates a new Client.
func NewClient(addr string, configs ...ClientConfig) Client {
	defaultBreaker := overcurrent.NewCircuitBreaker(
		overcurrent.WithHalfClosedRetryProbability(1.0),
		overcurrent.WithResetBackoff(backoff.NewConstantBackoff(time.Second*5)),
		overcurrent.WithFailureInterpreter(overcurrent.NewAnyErrorFailureInterpreter()),
		overcurrent.WithTripCondition(overcurrent.NewConsecutiveFailureTripCondition(1)),
	)

	config := &clientConfig{
		password:       "",
		database:       0,
		connectTimeout: time.Second * 5,
		writeTimeout:   time.Second * 5,
		readTimeout:    time.Second * 5,
		poolCapacity:   10,
		breaker:        defaultBreaker,
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
			config.breaker,
			config.clock,
		),
		logger: config.logger,
	}
}

func WithPassword(password string) ClientConfig {
	return func(c *clientConfig) { c.password = password }
}

func WithDatabase(database int) ClientConfig {
	return func(c *clientConfig) { c.database = database }
}

func WithConnectTimeout(timeout time.Duration) ClientConfig {
	return func(c *clientConfig) { c.connectTimeout = timeout }
}

func WithReadTimeout(timeout time.Duration) ClientConfig {
	return func(c *clientConfig) { c.readTimeout = timeout }
}

func WithWriteTimeout(timeout time.Duration) ClientConfig {
	return func(c *clientConfig) { c.writeTimeout = timeout }
}

func WithPoolCapacity(capacity int) ClientConfig {
	return func(c *clientConfig) { c.poolCapacity = capacity }
}

func WithBreaker(breaker overcurrent.CircuitBreaker) ClientConfig {
	return func(c *clientConfig) { c.breaker = breaker }
}

func WithBorrowTimeout(timeout time.Duration) ClientConfig {
	return func(c *clientConfig) { c.borrowTimeout = &timeout }
}

func WithClock(clock glock.Clock) ClientConfig {
	return func(c *clientConfig) { c.clock = clock }
}

func WithLogger(logger Logger) ClientConfig {
	return func(c *clientConfig) { c.logger = logger }
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
