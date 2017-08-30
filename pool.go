package deepjoy

import (
	"context"
	"sync"
	"time"

	"github.com/efritz/glock"
	"github.com/efritz/overcurrent"
)

type (
	// Conn abstracts a single, feature-minimal connection to Redis.
	Conn interface {
		// Close the connection to the remote Redis server.
		Close() error

		// Do performs a command on the remote Redis server and returns
		// its result.
		Do(command string, args ...interface{}) (interface{}, error)

		// Send will publish command as part of a MULTI/EXEC sequence
		// to the remote Redis server.
		Send(command string, args ...interface{}) error
	}

	// Pool abstracts a fixed-size Redis connection pool.
	Pool interface {
		// Close will drain all available connections from the pool.
		// Every live connection is closed. This method blocks.
		Close()

		// Borrow will block until a connection value is available in
		// the pool. If the connection is nil, then a new connection
		// is dialed in its place.
		Borrow() (Conn, bool)

		// BorrowTimeout is like borrow, but will return the pair
		// (nil, false) if no value is returned to the pool before the
		// given timeout elapses.
		BorrowTimeout(timeout time.Duration) (Conn, bool)

		// Release returns a connection to the pool. This method must
		// be called exactly once for each call to a Borrow method. A
		// connection which encountered an error should be returned to
		// the pool as a nil value.
		Release(conn Conn)
	}

	DialFunc func() (Conn, error)

	pool struct {
		dialer         DialFunc
		capacity       int
		logger         Logger
		breaker        overcurrent.CircuitBreaker
		clock          glock.Clock
		connections    chan Conn
		nilConnections chan Conn
		mutex          *sync.RWMutex
	}
)

func NewPool(
	dialer DialFunc,
	capacity int,
	logger Logger,
	breaker overcurrent.CircuitBreaker,
	clock glock.Clock,
) Pool {
	p := &pool{
		dialer:         dialer,
		capacity:       capacity,
		logger:         logger,
		breaker:        breaker,
		clock:          clock,
		connections:    make(chan Conn, capacity),
		nilConnections: make(chan Conn, capacity),
		mutex:          &sync.RWMutex{},
	}

	// Set the capacity of the pool. Each time a nil value is borrowed, a new
	// connection is established and used in its place.

	for i := 0; i < p.capacity; i++ {
		p.nilConnections <- nil
	}

	return p
}

func (p *pool) Close() {
	for i := 0; i < p.capacity; i++ {
		if conn, _ := p.get(nil); conn != nil {
			conn.Close()
		}
	}

	close(p.connections)
	close(p.nilConnections)
}

func (p *pool) Borrow() (Conn, bool) {
	if conn, _ := p.get(nil); conn != nil {
		return conn, true
	}

	return p.dial()
}

func (p *pool) BorrowTimeout(timeout time.Duration) (Conn, bool) {
	if conn, ok := p.get(&timeout); conn != nil || !ok {
		return conn, ok
	}

	return p.dial()
}

func (p *pool) Release(conn Conn) {
	if conn == nil {
		p.nilConnections <- conn
	} else {
		p.connections <- conn
	}
}

//
// Pool Helper Functions

// Get a value from the pool. If timeout is nil, no timeout is applied.
// This method attempts to read from the non-nil connection channel first
// in order to minimize the number of open connections when the pool is
// not under heavy concurrent load.
func (p *pool) get(timeout *time.Duration) (Conn, bool) {
	select {
	case conn := <-p.connections:
		return conn, true
	default:
	}

	select {
	case conn := <-p.connections:
		return conn, true

	case conn := <-p.nilConnections:
		return conn, true

	case <-makeTimeoutChan(timeout, p.clock):
		return nil, false
	}
}

// Dial a new Redis connection. The call ot the dialer function is wrapped
// in a circuit breaker so that if the remote end is down we are not going
// to hammer it.
func (p *pool) dial() (Conn, bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var conn Conn
	err := p.breaker.Call(func(ctx context.Context) error {
		temp, err := p.dialer()
		conn = temp
		return err
	})

	if err != nil {
		// We were dialing a nil connection, put this back in the pool
		// so that we're not draining our pool on connection errors.
		p.nilConnections <- nil

		p.logger.Printf("Could not connect to Redis (%s)", err.Error())
		return nil, false
	}

	p.logger.Printf("Established a new connection with Redis")
	return conn, true
}

var blockingChan = make(chan time.Time)

// Wraps time.After around a possibly nil-timeout. When timeout is nil this
// method will return a channel which is always open but never written to.
func makeTimeoutChan(timeout *time.Duration, clock glock.Clock) <-chan time.Time {
	if timeout == nil {
		return blockingChan
	}

	return clock.After(*timeout)
}
