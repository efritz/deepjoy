package deepjoy

import (
	"context"
	"time"

	"github.com/aphistic/sweet"
	"github.com/efritz/glock"
	. "github.com/efritz/go-mockgen/matchers"
	"github.com/efritz/overcurrent"
	. "github.com/onsi/gomega"

	"github.com/efritz/deepjoy/mocks"
)

type PoolSuite struct{}

func (s *PoolSuite) TestNewPoolAtCapacity(t sweet.T) {
	var (
		clock = glock.NewMockClock()
		sync  = make(chan struct{})
		pool  = NewPool(
			testDial,
			20,
			NilLogger,
			noopBreakerFunc,
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

func (s *PoolSuite) TestPoolDialOnNilConnection(t sweet.T) {
	var (
		conn = mocks.NewMockConn()
		dial = func() (Conn, error) { return conn, nil }
		pool = NewPool(
			dial,
			20,
			NilLogger,
			noopBreakerFunc,
			nil,
		)
	)

	c, ok := pool.Borrow()
	Expect(c).To(BeIdenticalTo(conn))
	Expect(ok).To(BeTrue())
}

func (s *PoolSuite) TestPoolDialOnNilConnectionAfterRelease(t sweet.T) {
	var (
		dials = 0
		conn  = mocks.NewMockConn()
		dial  = func() (Conn, error) { dials++; return conn, nil }
		pool  = NewPool(
			dial,
			20,
			NilLogger,
			noopBreakerFunc,
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

func (s *PoolSuite) TestClose(t sweet.T) {
	var (
		conn = mocks.NewMockConn()
		pool = NewPool(
			testDial,
			20,
			NilLogger,
			noopBreakerFunc,
			nil,
		)
	)

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
	Expect(conn.CloseFunc).To(BeCalledN(10))
}

func (s *PoolSuite) TestCloseBlocks(t sweet.T) {
	var (
		sync  = make(chan struct{})
		block = make(chan struct{})
		conn  = mocks.NewMockConn()
		pool  = NewPool(
			testDial,
			20,
			NilLogger,
			noopBreakerFunc,
			nil,
		)
	)

	conn.CloseFunc.SetDefaultHook(func() error {
		<-block
		return nil
	})

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

func (s *PoolSuite) TestBorrowFavorsNonNil(t sweet.T) {
	var (
		dials = 0
		conn  = mocks.NewMockConn()
		pool  = NewPool(
			func() (Conn, error) { dials++; return conn, nil },
			20,
			NilLogger,
			noopBreakerFunc,
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

func (s *PoolSuite) TestPoolCapacity(t sweet.T) {
	var (
		sync = make(chan struct{})
		pool = NewPool(
			testDial,
			20,
			NilLogger,
			noopBreakerFunc,
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

func (s *PoolSuite) TestBorrowTimeout(t sweet.T) {
	var (
		result = make(chan bool)
		clock  = glock.NewMockClock()
		pool   = NewPool(
			testDial,
			20,
			NilLogger,
			noopBreakerFunc,
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

func (s *PoolSuite) TestCircuitBreaker(t sweet.T) {
	var (
		count       = 5
		breakerFunc = func(f overcurrent.BreakerFunc) error {
			if count <= 0 {
				return overcurrent.ErrCircuitOpen
			}

			count--
			return f(context.Background())
		}

		pool = NewPool(
			testDial,
			20,
			NilLogger,
			breakerFunc,
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

func testDial() (Conn, error) {
	return mocks.NewMockConn(), nil
}
