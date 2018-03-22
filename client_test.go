package deepjoy

import (
	"errors"
	"io"
	"time"

	"github.com/aphistic/sweet"
	"github.com/efritz/glock"
	. "github.com/onsi/gomega"
)

type ClientSuite struct{}

func (s *ClientSuite) TestClose(t sweet.T) {
	var (
		pool   = newMockPool()
		called = false
		c      = makeClient(pool, nil)
	)

	pool.close = func() {
		called = true
	}

	c.Close()
	Expect(called).To(BeTrue())
}

func (s *ClientSuite) TestDo(t sweet.T) {
	var (
		pool     = newMockPool()
		conn     = newMockConn()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.borrow = func() (Conn, bool) {
		return conn, true
	}

	pool.release = func(conn Conn) {
		released <- conn
	}

	conn.do = func(command string, args ...interface{}) (interface{}, error) {
		return []string{"BAR", "BAZ", "QUUX"}, nil
	}

	result, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(BeNil())
	Expect(result).To(Equal([]string{"BAR", "BAZ", "QUUX"}))
	Expect(released).To(Receive(Equal(conn)))
}

func (s *ClientSuite) TestDoNoConnection(t sweet.T) {
	var (
		pool     = newMockPool()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.borrow = func() (Conn, bool) {
		return nil, false
	}

	pool.release = func(conn Conn) {
		released <- conn
	}

	_, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(Equal(ErrNoConnection))

	// Nothing to release
	Consistently(released).ShouldNot(Receive())
}

func (s *ClientSuite) TestDoError(t sweet.T) {
	var (
		pool     = newMockPool()
		conn     = newMockConn()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.borrow = func() (Conn, bool) {
		return conn, true
	}

	pool.release = func(conn Conn) {
		released <- conn
	}

	conn.do = func(command string, args ...interface{}) (interface{}, error) {
		return nil, errors.New("utoh")
	}

	_, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(MatchError("utoh"))
	Expect(released).To(Receive(BeNil()))
}

func (s *ClientSuite) TestDoRetryableError(t sweet.T) {
	var (
		pool        = newMockPool()
		conn1       = newMockConn()
		conn2       = newMockConn()
		clock       = glock.NewMockClock()
		borrowCount = 0
		released    = make(chan Conn, 2)
		c           = makeClient(pool, clock)
	)

	defer close(released)

	pool.borrow = func() (Conn, bool) {
		c := []Conn{conn1, conn2}[borrowCount]
		borrowCount++
		return c, true
	}

	pool.release = func(conn Conn) {
		released <- conn
	}

	conn1.do = func(command string, args ...interface{}) (interface{}, error) {
		return nil, connErr{io.EOF}
	}

	conn2.do = func(command string, args ...interface{}) (interface{}, error) {
		return []string{"BAR", "BAZ", "QUUX"}, nil
	}

	go func() {
		// Unlock the after call in client
		clock.BlockingAdvance(time.Second)
	}()

	result, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(BeNil())
	Expect(result).To(Equal([]string{"BAR", "BAZ", "QUUX"}))
	Expect(released).To(Receive(BeNil()))
	Expect(released).To(Receive(Equal(conn2)))
}

func (s *ClientSuite) TestTransaction(t sweet.T) {
	var (
		pool     = newMockPool()
		conn     = newMockConn()
		released = make(chan Conn, 1)
		commands = make(chan Command, 5)
		c        = makeClient(pool, nil)
	)

	defer close(released)
	defer close(commands)

	pool.borrow = func() (Conn, bool) {
		return conn, true
	}

	pool.release = func(conn Conn) {
		released <- conn
	}

	conn.do = func(command string, args ...interface{}) (interface{}, error) {
		commands <- NewCommand(command, args...)
		return []int{1, 2, 3, 4}, nil
	}

	conn.send = func(command string, args ...interface{}) error {
		commands <- NewCommand(command, args...)
		return nil
	}

	result, err := c.Transaction(
		NewCommand("foo", 1, 2, 3),
		NewCommand("bar", 2, 3, 4),
		NewCommand("baz", 3, 4, 5),
	)

	Expect(result).To(Equal([]int{1, 2, 3, 4}))
	Expect(err).To(BeNil())
	Eventually(released).Should(Receive(Equal(conn)))

	Eventually(commands).Should(Receive(Equal(NewCommand("MULTI"))))
	Eventually(commands).Should(Receive(Equal(NewCommand("foo", 1, 2, 3))))
	Eventually(commands).Should(Receive(Equal(NewCommand("bar", 2, 3, 4))))
	Eventually(commands).Should(Receive(Equal(NewCommand("baz", 3, 4, 5))))
	Eventually(commands).Should(Receive(Equal(NewCommand("EXEC"))))
	Consistently(commands).ShouldNot(Receive())
}

func (s *ClientSuite) TestTransactionNoConnection(t sweet.T) {
	var (
		pool     = newMockPool()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.borrow = func() (Conn, bool) {
		return nil, false
	}

	pool.release = func(conn Conn) {
		released <- conn
	}

	_, err := c.Transaction()
	Expect(err).To(Equal(ErrNoConnection))

	// Nothing to release
	Consistently(released).ShouldNot(Receive())
}

func (s *ClientSuite) TestTransactionError(t sweet.T) {
	var (
		pool     = newMockPool()
		conn     = newMockConn()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.borrow = func() (Conn, bool) {
		return conn, true
	}

	pool.release = func(conn Conn) {
		released <- conn
	}

	conn.send = func(command string, args ...interface{}) error {
		if command == "bar" {
			return errors.New("utoh")
		}

		return nil
	}

	_, err := c.Transaction(
		NewCommand("foo", 1, 2, 3),
		NewCommand("bar", 2, 3, 4),
		NewCommand("baz", 3, 4, 5),
	)

	Expect(err).To(MatchError("utoh"))
	Eventually(released).Should(Receive(BeNil()))
}

func (s *ClientSuite) TestTransactionRetryableError(t sweet.T) {
	var (
		pool        = newMockPool()
		conn1       = newMockConn()
		clock       = glock.NewMockClock()
		borrowCount = 0
		conn2       = newMockConn()
		released    = make(chan Conn, 2)
		c           = makeClient(pool, clock)
	)

	defer close(released)

	pool.borrow = func() (Conn, bool) {
		c := []Conn{conn1, conn2}[borrowCount]
		borrowCount++
		return c, true
	}

	pool.release = func(conn Conn) {
		released <- conn
	}

	conn2.do = func(command string, args ...interface{}) (interface{}, error) {
		return []int{1, 2, 3, 4}, nil
	}

	conn1.send = func(command string, args ...interface{}) error {
		if borrowCount == 1 && command == "MULTI" {
			return connErr{io.ErrUnexpectedEOF}
		}

		return nil
	}

	go func() {
		// Unlock the after call in client
		clock.BlockingAdvance(time.Second)
	}()

	result, err := c.Transaction(
		NewCommand("foo", 1, 2, 3),
		NewCommand("bar", 2, 3, 4),
		NewCommand("baz", 3, 4, 5),
	)

	Expect(result).To(Equal([]int{1, 2, 3, 4}))
	Expect(err).To(BeNil())
	Eventually(released).Should(Receive(BeNil()))
	Eventually(released).Should(Receive(Equal(conn2)))
}

func (s *ClientSuite) TestTransactionRetryableErrorAfterMulti(t sweet.T) {
	var (
		pool        = newMockPool()
		conn1       = newMockConn()
		clock       = glock.NewMockClock()
		borrowCount = 0
		conn2       = newMockConn()
		released    = make(chan Conn, 2)
		c           = makeClient(pool, clock)
	)

	defer close(released)

	pool.borrow = func() (Conn, bool) {
		c := []Conn{conn1, conn2}[borrowCount]
		borrowCount++
		return c, true
	}

	pool.release = func(conn Conn) {
		released <- conn
	}

	conn2.do = func(command string, args ...interface{}) (interface{}, error) {
		return []int{1, 2, 3, 4}, nil
	}

	conn1.send = func(command string, args ...interface{}) error {
		if borrowCount == 1 && command == "bar" {
			return connErr{io.ErrUnexpectedEOF}
		}

		return nil
	}

	go func() {
		// Unlock the after call in client
		clock.BlockingAdvance(time.Second)
	}()

	result, err := c.Transaction(
		NewCommand("foo", 1, 2, 3),
		NewCommand("bar", 2, 3, 4),
		NewCommand("baz", 3, 4, 5),
	)

	Expect(result).To(Equal([]int{1, 2, 3, 4}))
	Expect(err).To(BeNil())
	Eventually(released).Should(Receive(BeNil()))
	Eventually(released).Should(Receive(Equal(conn2)))
}

//
// Helpers

func makeClient(pool Pool, clock glock.Clock) *client {
	return &client{
		pool:    pool,
		backoff: defaultBackoff,
		clock:   clock,
		logger:  testLogger,
	}
}

//
// Mock Pool

type mockPool struct {
	close         func()
	borrow        func() (Conn, bool)
	borrowTimeout func(timeout time.Duration) (Conn, bool)
	release       func(conn Conn)
}

func newMockPool() *mockPool {
	return &mockPool{
		close:         func() {},
		borrow:        func() (Conn, bool) { return nil, false },
		borrowTimeout: func(timeout time.Duration) (Conn, bool) { return nil, false },
		release:       func(conn Conn) {},
	}
}

func (p *mockPool) Close()                                           { p.close() }
func (p *mockPool) Borrow() (Conn, bool)                             { return p.borrow() }
func (p *mockPool) BorrowTimeout(timeout time.Duration) (Conn, bool) { return p.borrowTimeout(timeout) }
func (p *mockPool) Release(conn Conn)                                { p.release(conn) }
