package deepjoy

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aphistic/sweet"
	"github.com/efritz/glock"
	. "github.com/efritz/go-mockgen/matchers"
	. "github.com/onsi/gomega"

	"github.com/efritz/deepjoy/mocks"
)

type ClientSuite struct{}

func (s *ClientSuite) TestConfigureReadReplica(t sweet.T) {
	client := NewClient(
		"master",
		WithLogger(NilLogger),
		WithReadReplicaAddrs("replica"),
		WithDialerFactory(func(addrs []string) DialFunc {
			return func() (Conn, error) {
				c := mocks.NewMockConn()
				c.DoFunc.SetDefaultReturn(addrs[0], nil)
				return c, nil
			}
		}),
	)

	Expect(client.Do("ping")).To(Equal("master"))
	Expect(client.ReadReplica().Do("ping")).To(Equal("replica"))
}

func (s *ClientSuite) TestReadReplica(t sweet.T) {
	var (
		pool1   = mocks.NewMockPool()
		pool2   = mocks.NewMockPool()
		conn1   = mocks.NewMockConn()
		conn2   = mocks.NewMockConn()
		client1 = makeClient(pool1, nil)
		client2 = makeClient(pool2, nil)
	)

	client1.readReplicaClient = client2
	pool1.BorrowFunc.SetDefaultReturn(conn1, true)
	pool2.BorrowFunc.SetDefaultReturn(conn2, true)

	client1.Do("foo")
	Expect(conn1.DoFunc).To(BeCalledOnce())
	Expect(conn2.DoFunc).NotTo(BeCalled())

	replica := client1.ReadReplica()
	replica.Do("foo")
	Expect(replica).To(Equal(client2))
	Expect(conn1.DoFunc).To(BeCalledOnce())
	Expect(conn2.DoFunc).To(BeCalledOnce())
}

func (s *ClientSuite) TestCloseReadReplica(t sweet.T) {
	var (
		pool1   = mocks.NewMockPool()
		pool2   = mocks.NewMockPool()
		client1 = makeClient(pool1, nil)
		client2 = makeClient(pool2, nil)
	)

	client1.readReplicaClient = client2
	client1.Close()
	Expect(pool1.CloseFunc).To(BeCalled())
	Expect(pool2.CloseFunc).To(BeCalled())
}

func (s *ClientSuite) TestNilReadReplica(t sweet.T) {
	c := makeClient(nil, nil)
	Expect(c.ReadReplica()).To(Equal(c))
}

func (s *ClientSuite) TestClose(t sweet.T) {
	pool := mocks.NewMockPool()
	makeClient(pool, nil).Close()
	Expect(pool.CloseFunc).To(BeCalled())
}

func (s *ClientSuite) TestDo(t sweet.T) {
	var (
		pool = mocks.NewMockPool()
		conn = mocks.NewMockConn()
		c    = makeClient(pool, nil)
	)

	pool.BorrowFunc.SetDefaultReturn(conn, true)
	conn.DoFunc.SetDefaultReturn([]string{"BAR", "BAZ", "QUUX"}, nil)

	result, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(BeNil())
	Expect(result).To(Equal([]string{"BAR", "BAZ", "QUUX"}))
	Expect(pool.ReleaseFunc).To(BeCalledWith(conn))
}

func (s *ClientSuite) TestDoNoConnection(t sweet.T) {
	var (
		pool = mocks.NewMockPool()
		c    = makeClient(pool, nil)
	)

	_, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(Equal(ErrNoConnection))

	// Nothing to release
	Expect(pool.ReleaseFunc).NotTo(BeCalled())
}

func (s *ClientSuite) TestDoError(t sweet.T) {
	var (
		pool = mocks.NewMockPool()
		conn = mocks.NewMockConn()
		c    = makeClient(pool, nil)
	)

	pool.BorrowFunc.SetDefaultReturn(conn, true)
	conn.DoFunc.SetDefaultReturn(nil, errors.New("utoh"))

	_, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(MatchError("utoh"))
	Expect(pool.ReleaseFunc).To(BeCalledWith(BeNil()))
}

func (s *ClientSuite) TestDoRetryableError(t sweet.T) {
	var (
		pool  = mocks.NewMockPool()
		conn1 = mocks.NewMockConn()
		conn2 = mocks.NewMockConn()
		clock = glock.NewMockClock()
		c     = makeClient(pool, clock)
	)

	pool.BorrowFunc.PushReturn(conn1, true)
	pool.BorrowFunc.PushReturn(conn2, true)
	conn1.DoFunc.SetDefaultReturn(nil, connErr{io.EOF})
	conn2.DoFunc.SetDefaultReturn([]string{"BAR", "BAZ", "QUUX"}, nil)

	go func() {
		// Unlock the after call in client
		clock.BlockingAdvance(time.Second)
	}()

	result, err := c.Do("upper", "bar", "baz", "quux")
	Expect(err).To(BeNil())
	Expect(result).To(Equal([]string{"BAR", "BAZ", "QUUX"}))
	Expect(pool.ReleaseFunc).To(BeCalledWith(BeNil()))
	Expect(pool.ReleaseFunc).To(BeCalledWith(conn2))
}

func (s *ClientSuite) TestPipeline(t sweet.T) {
	var (
		pool = mocks.NewMockPool()
		conn = mocks.NewMockConn()
		c    = makeClient(pool, nil)
	)

	pool.BorrowFunc.SetDefaultReturn(conn, true)
	conn.DoFunc.SetDefaultReturn([]int{1, 2, 3, 4}, nil)

	pipeline := c.Pipeline()
	pipeline.Add("foo", 1, 2, 3)
	pipeline.Add("bar", 2, 3, 4)
	pipeline.Add("baz", 3, 4, 5)

	result, err := pipeline.Run()
	Expect(err).To(BeNil())
	Expect(result).To(Equal([]int{1, 2, 3, 4}))

	Expect(pool.ReleaseFunc).To(BeCalledWith(conn))
	Expect(conn.SendFunc).To(BeCalledWith("MULTI"))
	Expect(conn.SendFunc).To(BeCalledWith("foo", 1, 2, 3))
	Expect(conn.SendFunc).To(BeCalledWith("bar", 2, 3, 4))
	Expect(conn.SendFunc).To(BeCalledWith("baz", 3, 4, 5))
	Expect(conn.DoFunc).To(BeCalledWith("EXEC"))
}

func (s *ClientSuite) TestPipelineNoConnection(t sweet.T) {
	var (
		pool = mocks.NewMockPool()
		c    = makeClient(pool, nil)
	)

	_, err := c.Pipeline().Run()
	Expect(err).To(Equal(ErrNoConnection))

	// Nothing to release
	Expect(pool.ReleaseFunc).NotTo(BeCalled())
}

func (s *ClientSuite) TestPipelineError(t sweet.T) {
	var (
		pool = mocks.NewMockPool()
		conn = mocks.NewMockConn()
		c    = makeClient(pool, nil)
	)

	pool.BorrowFunc.SetDefaultReturn(conn, true)
	conn.SendFunc.PushReturn(nil)
	conn.SendFunc.PushReturn(fmt.Errorf("utoh"))

	pipeline := c.Pipeline()
	pipeline.Add("foo", 1, 2, 3)
	pipeline.Add("bar", 2, 3, 4)
	pipeline.Add("baz", 3, 4, 5)
	_, err := pipeline.Run()

	Expect(err).To(MatchError("utoh"))
	Expect(pool.ReleaseFunc).To(BeCalledWith(BeNil()))
}

func (s *ClientSuite) TestPipelineRetryableError(t sweet.T) {
	var (
		pool  = mocks.NewMockPool()
		conn1 = mocks.NewMockConn()
		clock = glock.NewMockClock()
		conn2 = mocks.NewMockConn()
		c     = makeClient(pool, clock)
	)

	pool.BorrowFunc.PushReturn(conn1, true)
	pool.BorrowFunc.PushReturn(conn2, true)
	conn2.DoFunc.SetDefaultReturn([]int{1, 2, 3, 4}, nil)
	conn1.SendFunc.PushReturn(connErr{io.ErrUnexpectedEOF})

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
	Expect(pool.ReleaseFunc).To(BeCalledWith(BeNil()))
	Expect(pool.ReleaseFunc).To(BeCalledWith(conn2))
}

func (s *ClientSuite) TestPipelineRetryableErrorAfterMulti(t sweet.T) {
	var (
		pool  = mocks.NewMockPool()
		conn1 = mocks.NewMockConn()
		clock = glock.NewMockClock()
		conn2 = mocks.NewMockConn()
		c     = makeClient(pool, clock)
	)

	pool.BorrowFunc.PushReturn(conn1, true)
	pool.BorrowFunc.PushReturn(conn2, true)
	conn2.DoFunc.SetDefaultReturn([]int{1, 2, 3, 4}, nil)
	conn1.SendFunc.PushReturn(nil)
	conn1.SendFunc.PushReturn(connErr{io.ErrUnexpectedEOF})

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

	Expect(pool.ReleaseFunc).To(BeCalledWith(BeNil()))
	Expect(pool.ReleaseFunc).To(BeCalledWith(conn2))
}

//
// Helpers

func makeClient(pool Pool, clock glock.Clock) *client {
	return &client{
		pool:    pool,
		backoff: defaultBackoff,
		clock:   clock,
		logger:  NilLogger,
	}
}
