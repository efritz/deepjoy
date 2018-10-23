package iface

import "time"

// Pool abstracts a fixed-size Redis connection pool.
type Pool interface {
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
