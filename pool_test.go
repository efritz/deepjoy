package deepjoy

import (
	"context"
	"testing"
	"time"

	"github.com/efritz/overcurrent"

	"github.com/efritz/glock"

	. "github.com/onsi/gomega"
)

type PoolSuite struct{}

func (s *PoolSuite) TestNewPoolAtCapacity(t *testing.T) {
	var (
		clock = glock.NewMockClock()
		sync  = make(chan struct{})
		pool  = NewPool(
			testDial,
			20,
			testLogger,
			overcurrent.NewNoopBreaker(),
			clock,
		)
	)

	for i := 0; i < 20; i++ {
		_, ok := pool.Borrow()
		Expect(ok).To(BeTrue())
	}

	go func() {
		_, ok := pool.BorrowTimeout(time.Second * 10)
		Expect(ok).To(BeFalse())
		close(sync)
	}()

	clock.BlockingAdvance(time.Second * 10)
	<-sync
}

func (s *PoolSuite) TestPoolDialOnNilConnection(t *testing.T) {
	var (
		conn = newMockConn()
		dial = func() (Conn, error) { return conn, nil }
		pool = NewPool(
			dial,
			20,
			testLogger,
			overcurrent.NewNoopBreaker(),
			nil,
		)
	)

	c, ok := pool.Borrow()
	Expect(c).To(BeIdenticalTo(conn))
	Expect(ok).To(BeTrue())
}

func (s *PoolSuite) TestPoolDialOnNilConnectionAfterRelease(t *testing.T) {
	var (
		dials = 0
		conn  = newMockConn()
		dial  = func() (Conn, error) { dials++; return conn, nil }
		pool  = NewPool(
			dial,
			20,
			testLogger,
			overcurrent.NewNoopBreaker(),
			nil,
		)
	)

	for i := 0; i < 20; i++ {
		pool.Borrow()
	}

	Expect(dials).To(Equal(20))

	for i := 0; i < 10; i++ {
		pool.Release(nil)
	}

	for i := 0; i < 10; i++ {
		pool.Release(conn)
	}

	for i := 0; i < 20; i++ {
		pool.Borrow()
	}

	// re-dial the 10 released nils
	Expect(dials).To(Equal(30))
}

func (s *PoolSuite) TestClose(t *testing.T) {
	var (
		closeCount = 0
		conn       = newMockConn()
		pool       = NewPool(
			testDial,
			20,
			testLogger,
			overcurrent.NewNoopBreaker(),
			nil,
		)
	)

	conn.close = func() error {
		closeCount++
		return nil
	}

	for i := 0; i < 15; i++ {
		pool.Borrow()
	}

	for i := 0; i < 5; i++ {
		pool.Release(nil)
	}

	for i := 0; i < 10; i++ {
		pool.Release(conn)
	}

	// Release the 10 live connections in pool
	pool.Close()
	Expect(closeCount).To(Equal(10))
}

func (s *PoolSuite) TestCloseBlocks(t *testing.T) {
	var (
		sync  = make(chan struct{})
		block = make(chan struct{})
		conn  = newMockConn()
		pool  = NewPool(
			testDial,
			20,
			testLogger,
			overcurrent.NewNoopBreaker(),
			nil,
		)
	)

	conn.close = func() error {
		<-block
		return nil
	}

	for i := 0; i < 5; i++ {
		pool.Borrow()
	}

	for i := 0; i < 5; i++ {
		pool.Release(conn)
	}

	go func() {
		pool.Close()
		close(sync)
	}()

	Consistently(sync).ShouldNot(Receive())
	close(block)
	Eventually(sync).Should(BeClosed())
}

func (s *PoolSuite) TestBorrowFavorsNonNil(t *testing.T) {
	var (
		dials = 0
		conn  = newMockConn()
		pool  = NewPool(
			func() (Conn, error) { dials++; return conn, nil },
			20,
			testLogger,
			overcurrent.NewNoopBreaker(),
			nil,
		)
	)

	// Dial one
	c1, _ := pool.Borrow()
	Expect(dials).To(Equal(1))

	// Still borrowed, dial another
	c2, _ := pool.Borrow()
	Expect(dials).To(Equal(2))

	// Return both, will get these back immediately
	pool.Release(c1)
	pool.Release(c2)
	pool.Borrow()
	pool.Borrow()
	Expect(dials).To(Equal(2))

	// Two borrowed, dial a third
	pool.Borrow()
	Expect(dials).To(Equal(3))
}

func (s *PoolSuite) TestPoolCapacity(t *testing.T) {
	var (
		sync = make(chan struct{})
		pool = NewPool(
			testDial,
			20,
			testLogger,
			overcurrent.NewNoopBreaker(),
			nil,
		)
	)

	for i := 0; i < 20; i++ {
		pool.Borrow()
	}

	go func() {
		pool.Borrow()
		close(sync)
	}()

	Consistently(sync).ShouldNot(BeClosed())
	pool.Release(nil)
	Eventually(sync).Should(BeClosed())
}

func (s *PoolSuite) TestBorrowTimeout(t *testing.T) {
	var (
		result = make(chan bool)
		clock  = glock.NewMockClock()
		pool   = NewPool(
			testDial,
			20,
			testLogger,
			overcurrent.NewNoopBreaker(),
			clock,
		)
	)

	for i := 0; i < 20; i++ {
		pool.Borrow()
	}

	go func() {
		defer close(result)
		_, ok := pool.BorrowTimeout(time.Second * 30)
		result <- ok
	}()

	Consistently(result).ShouldNot(BeClosed())
	clock.BlockingAdvance(time.Second * 30)
	Eventually(result).Should(Receive(Equal(false)))
}

func (s *PoolSuite) TestCircuitBreaker(t *testing.T) {
	var (
		pool = NewPool(
			testDial,
			20,
			testLogger,
			newMockCircuitBreaker(5),
			nil,
		)
	)

	for i := 0; i < 5; i++ {
		_, ok := pool.Borrow()
		Expect(ok).To(BeTrue())
	}

	for i := 0; i < 100; i++ {
		_, ok := pool.Borrow()
		Expect(ok).To(BeFalse())
	}
}

//
// Mock Connection

func testDial() (Conn, error) {
	return &mockConn{}, nil
}

type mockConn struct {
	close func() error
	do    func(command string, args ...interface{}) (interface{}, error)
	send  func(command string, args ...interface{}) error
}

func newMockConn() *mockConn {
	return &mockConn{
		close: func() error { return nil },
		do:    func(command string, args ...interface{}) (interface{}, error) { return nil, nil },
		send:  func(command string, args ...interface{}) error { return nil },
	}
}

func (c *mockConn) Close() error {
	return c.close()
}

func (c *mockConn) Do(command string, args ...interface{}) (interface{}, error) {
	return c.do(command, args...)
}

func (c *mockConn) Send(command string, args ...interface{}) error {
	return c.send(command, args...)
}

//
// Circuit Breaker

type mockCircuitBreaker struct {
	tripAfter int
}

func newMockCircuitBreaker(tripAfter int) overcurrent.CircuitBreaker {
	return &mockCircuitBreaker{tripAfter: tripAfter}
}

func (b *mockCircuitBreaker) Trip()                     {}
func (b *mockCircuitBreaker) Reset()                    {}
func (b *mockCircuitBreaker) ShouldTry() bool           { return true }
func (b *mockCircuitBreaker) MarkResult(err error) bool { return true }
func (b *mockCircuitBreaker) Call(f overcurrent.BreakerFunc) error {
	if b.tripAfter <= 0 {
		return overcurrent.ErrCircuitOpen
	}

	b.tripAfter--
	return f(context.Background())
}

func (b *mockCircuitBreaker) CallAsync(f overcurrent.BreakerFunc) <-chan error {
	return nil
}
