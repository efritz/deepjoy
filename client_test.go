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

func (s *ClientSuite) TestConfigureReadReplica(t sweet.T) {
	client := NewClient(
		"master",
		WithLogger(testLogger),
		WithReadReplicaAddrs("replica"),
		WithDialerFactory(func(addrs []string) DialFunc {
			return func() (Conn, error) {
				c := NewMockConn()
				c.DoFunc = func(command string, args ...interface{}) (interface{}, error) {
					return addrs[0], nil
				}

				return c, nil
			}
		}),
	)

	Expect(client.Do("ping")).To(Equal("master"))
	Expect(client.ReadReplica().Do("ping")).To(Equal("replica"))
}

func (s *ClientSuite) TestReadReplica(t sweet.T) {
	var (
		pool1   = NewMockPool()
		pool2   = NewMockPool()
		conn1   = NewMockConn()
		conn2   = NewMockConn()
		called1 = 0
		called2 = 0
		client1 = makeClient(pool1, nil)
		client2 = makeClient(pool2, nil)
	)

	client1.readReplicaClient = client2

	pool1.BorrowFunc = func() (Conn, bool) { return conn1, true }
	pool2.BorrowFunc = func() (Conn, bool) { return conn2, true }

	conn1.DoFunc = func(command string, args ...interface{}) (interface{}, error) { called1++; return "", nil }
	conn2.DoFunc = func(command string, args ...interface{}) (interface{}, error) { called2++; return "", nil }

	client1.Do("foo")
	Expect(called1).To(Equal(1))
	Expect(called2).To(Equal(0))

	replica := client1.ReadReplica()
	replica.Do("foo")
	Expect(replica).To(Equal(client2))
	Expect(called1).To(Equal(1))
	Expect(called2).To(Equal(1))
}

func (s *ClientSuite) TestCloseReadReplica(t sweet.T) {
	var (
		pool1   = NewMockPool()
		pool2   = NewMockPool()
		closed1 = false
		closed2 = false
		client1 = makeClient(pool1, nil)
		client2 = makeClient(pool2, nil)
	)

	client1.readReplicaClient = client2

	pool1.CloseFunc = func() { closed1 = true }
	pool2.CloseFunc = func() { closed2 = true }

	client1.Close()
	Expect(closed1).To(BeTrue())
	Expect(closed2).To(BeTrue())
}

func (s *ClientSuite) TestNilReadReplica(t sweet.T) {
	c := makeClient(nil, nil)
	Expect(c.ReadReplica()).To(Equal(c))
}

func (s *ClientSuite) TestClose(t sweet.T) {
	var (
		pool   = NewMockPool()
		called = false
		c      = makeClient(pool, nil)
	)

	pool.CloseFunc = func() {
		called = true
	}

	c.Close()
	Expect(called).To(BeTrue())
}

func (s *ClientSuite) TestDo(t sweet.T) {
	var (
		pool     = NewMockPool()
		conn     = NewMockConn()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.BorrowFunc = func() (Conn, bool) {
		return conn, true
	}

	pool.ReleaseFunc = func(conn Conn) {
		released <- conn
	}

	conn.DoFunc = func(command string, args ...interface{}) (interface{}, error) {
		return []string{"BAR", "BAZ", "QUUX"}, nil
	}

	result, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(BeNil())
	Expect(result).To(Equal([]string{"BAR", "BAZ", "QUUX"}))
	Expect(released).To(Receive(Equal(conn)))
}

func (s *ClientSuite) TestDoNoConnection(t sweet.T) {
	var (
		pool     = NewMockPool()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.BorrowFunc = func() (Conn, bool) {
		return nil, false
	}

	pool.ReleaseFunc = func(conn Conn) {
		released <- conn
	}

	_, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(Equal(ErrNoConnection))

	// Nothing to release
	Consistently(released).ShouldNot(Receive())
}

func (s *ClientSuite) TestDoError(t sweet.T) {
	var (
		pool     = NewMockPool()
		conn     = NewMockConn()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.BorrowFunc = func() (Conn, bool) {
		return conn, true
	}

	pool.ReleaseFunc = func(conn Conn) {
		released <- conn
	}

	conn.DoFunc = func(command string, args ...interface{}) (interface{}, error) {
		return nil, errors.New("utoh")
	}

	_, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(MatchError("utoh"))
	Expect(released).To(Receive(BeNil()))
}

func (s *ClientSuite) TestDoRetryableError(t sweet.T) {
	var (
		pool        = NewMockPool()
		conn1       = NewMockConn()
		conn2       = NewMockConn()
		clock       = glock.NewMockClock()
		borrowCount = 0
		released    = make(chan Conn, 2)
		c           = makeClient(pool, clock)
	)

	defer close(released)

	pool.BorrowFunc = func() (Conn, bool) {
		c := []Conn{conn1, conn2}[borrowCount]
		borrowCount++
		return c, true
	}

	pool.ReleaseFunc = func(conn Conn) {
		released <- conn
	}

	conn1.DoFunc = func(command string, args ...interface{}) (interface{}, error) {
		return nil, connErr{io.EOF}
	}

	conn2.DoFunc = func(command string, args ...interface{}) (interface{}, error) {
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
		pool     = NewMockPool()
		conn     = NewMockConn()
		released = make(chan Conn, 1)
		commands = make(chan commandPair, 5)
		c        = makeClient(pool, nil)
	)

	defer close(released)
	defer close(commands)

	pool.BorrowFunc = func() (Conn, bool) {
		return conn, true
	}

	pool.ReleaseFunc = func(conn Conn) {
		released <- conn
	}

	conn.DoFunc = func(command string, args ...interface{}) (interface{}, error) {
		commands <- commandPair{command, args}
		return []int{1, 2, 3, 4}, nil
	}

	conn.SendFunc = func(command string, args ...interface{}) error {
		commands <- commandPair{command, args}
		return nil
	}

	pipeline := c.Pipeline()
	pipeline.Add("foo", 1, 2, 3)
	pipeline.Add("bar", 2, 3, 4)
	pipeline.Add("baz", 3, 4, 5)

	result, err := pipeline.Run()
	Expect(err).To(BeNil())
	Expect(result).To(Equal([]int{1, 2, 3, 4}))

	Eventually(released).Should(Receive(Equal(conn)))
	Eventually(commands).Should(Receive(Equal(commandPair{"MULTI", nil})))
	Eventually(commands).Should(Receive(Equal(commandPair{"foo", []interface{}{1, 2, 3}})))
	Eventually(commands).Should(Receive(Equal(commandPair{"bar", []interface{}{2, 3, 4}})))
	Eventually(commands).Should(Receive(Equal(commandPair{"baz", []interface{}{3, 4, 5}})))
	Eventually(commands).Should(Receive(Equal(commandPair{"EXEC", nil})))
	Consistently(commands).ShouldNot(Receive())
}

func (s *ClientSuite) TestTransactionNoConnection(t sweet.T) {
	var (
		pool     = NewMockPool()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.BorrowFunc = func() (Conn, bool) {
		return nil, false
	}

	pool.ReleaseFunc = func(conn Conn) {
		released <- conn
	}

	_, err := c.Pipeline().Run()
	Expect(err).To(Equal(ErrNoConnection))

	// Nothing to release
	Consistently(released).ShouldNot(Receive())
}

func (s *ClientSuite) TestTransactionError(t sweet.T) {
	var (
		pool     = NewMockPool()
		conn     = NewMockConn()
		released = make(chan Conn, 1)
		c        = makeClient(pool, nil)
	)

	defer close(released)

	pool.BorrowFunc = func() (Conn, bool) {
		return conn, true
	}

	pool.ReleaseFunc = func(conn Conn) {
		released <- conn
	}

	conn.SendFunc = func(command string, args ...interface{}) error {
		if command == "bar" {
			return errors.New("utoh")
		}

		return nil
	}

	pipeline := c.Pipeline()
	pipeline.Add("foo", 1, 2, 3)
	pipeline.Add("bar", 2, 3, 4)
	pipeline.Add("baz", 3, 4, 5)
	_, err := pipeline.Run()

	Expect(err).To(MatchError("utoh"))
	Eventually(released).Should(Receive(BeNil()))
}

func (s *ClientSuite) TestTransactionRetryableError(t sweet.T) {
	var (
		pool        = NewMockPool()
		conn1       = NewMockConn()
		clock       = glock.NewMockClock()
		borrowCount = 0
		conn2       = NewMockConn()
		released    = make(chan Conn, 2)
		c           = makeClient(pool, clock)
	)

	defer close(released)

	pool.BorrowFunc = func() (Conn, bool) {
		c := []Conn{conn1, conn2}[borrowCount]
		borrowCount++
		return c, true
	}

	pool.ReleaseFunc = func(conn Conn) {
		released <- conn
	}

	conn2.DoFunc = func(command string, args ...interface{}) (interface{}, error) {
		return []int{1, 2, 3, 4}, nil
	}

	conn1.SendFunc = func(command string, args ...interface{}) error {
		if borrowCount == 1 && command == "MULTI" {
			return connErr{io.ErrUnexpectedEOF}
		}

		return nil
	}

	go func() {
		// Unlock the after call in client
		clock.BlockingAdvance(time.Second)
	}()

	pipeline := c.Pipeline()
	pipeline.Add("foo", 1, 2, 3)
	pipeline.Add("bar", 2, 3, 4)
	pipeline.Add("baz", 3, 4, 5)
	result, err := pipeline.Run()

	Expect(err).To(BeNil())
	Expect(result).To(Equal([]int{1, 2, 3, 4}))
	Eventually(released).Should(Receive(BeNil()))
	Eventually(released).Should(Receive(Equal(conn2)))
}

func (s *ClientSuite) TestTransactionRetryableErrorAfterMulti(t sweet.T) {
	var (
		pool        = NewMockPool()
		conn1       = NewMockConn()
		clock       = glock.NewMockClock()
		borrowCount = 0
		conn2       = NewMockConn()
		released    = make(chan Conn, 2)
		c           = makeClient(pool, clock)
	)

	defer close(released)

	pool.BorrowFunc = func() (Conn, bool) {
		c := []Conn{conn1, conn2}[borrowCount]
		borrowCount++
		return c, true
	}

	pool.ReleaseFunc = func(conn Conn) {
		released <- conn
	}

	conn2.DoFunc = func(command string, args ...interface{}) (interface{}, error) {
		return []int{1, 2, 3, 4}, nil
	}

	conn1.SendFunc = func(command string, args ...interface{}) error {
		if borrowCount == 1 && command == "bar" {
			return connErr{io.ErrUnexpectedEOF}
		}

		return nil
	}

	go func() {
		// Unlock the after call in client
		clock.BlockingAdvance(time.Second)
	}()

	pipeline := c.Pipeline()
	pipeline.Add("foo", 1, 2, 3)
	pipeline.Add("bar", 2, 3, 4)
	pipeline.Add("baz", 3, 4, 5)

	result, err := pipeline.Run()
	Expect(err).To(BeNil())
	Expect(result).To(Equal([]int{1, 2, 3, 4}))

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
