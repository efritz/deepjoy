package deepjoy

import (
	"time"

	"github.com/efritz/backoff"
	"github.com/efritz/overcurrent"
)

// ConfigFunc is a function used to initialize a new client.
type ConfigFunc func(*clientConfig)

// WithDialerFactory sets the dialer factory to use to create the
// connection pools. Each factory creates connections to a single
// unique address.
func WithDialerFactory(dialerFactory DialerFactory) ConfigFunc {
	return func(c *clientConfig) { c.dialerFactory = dialerFactory }
}

// WithReadReplicaAddrs sets the addresses of the client returned
// by client's the ReadReplica() method.
func WithReadReplicaAddrs(addrs ...string) ConfigFunc {
	return func(c *clientConfig) { c.readAddrs = addrs }
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
