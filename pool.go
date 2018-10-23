package deepjoy

import (
	"context"
	"sync"
	"time"

	"github.com/efritz/glock"
	"github.com/efritz/overcurrent"

	"github.com/efritz/deepjoy/iface"
)

type (
	// Pool abstracts a fixed-size Redis connection pool.
	Pool = iface.Pool

	pool struct {
		dialer         DialFunc
		capacity       int
		logger         Logger
		breakerFunc    BreakerFunc
		clock          glock.Clock
		connections    chan Conn
		nilConnections chan Conn
		mutex          sync.RWMutex
	}

	// BreakerFunc bridges the interface between the Call function of
	// an overcurrent breaker and an overcurrent registry.
	BreakerFunc func(overcurrent.BreakerFunc) error
)

func noopBreakerFunc(f overcurrent.BreakerFunc) error {
	return f(context.Background())
}

// NewPool creates a pool with initially nil-connections.
func NewPool(
	dialer DialFunc,
	capacity int,
	logger Logger,
	breakerFunc BreakerFunc,
	clock glock.Clock,
) Pool {
	p := &pool{
		dialer:         dialer,
		capacity:       capacity,
		logger:         logger,
		breakerFunc:    breakerFunc,
		clock:          clock,
		connections:    make(chan Conn, capacity),
		nilConnections: make(chan Conn, capacity),
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
			if err := conn.Close(); err != nil {
				p.logger.Printf("Could not close connection (%s)", err.Error())
			}
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
	err := p.breakerFunc(func(ctx context.Context) error {
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
