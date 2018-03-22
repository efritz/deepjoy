package deepjoy

import "github.com/garyburd/redigo/redis"

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

	redigoShim struct {
		conn redis.Conn
	}

	connErr struct{ error }

	// DialFunc creates a connection to Redis or returns an error.
	DialFunc func() (Conn, error)
)

func makeDialer(addr string, config *clientConfig) DialFunc {
	return func() (Conn, error) {
		conn, err := redis.Dial(
			"tcp",
			addr,
			redis.DialPassword(config.password),
			redis.DialDatabase(config.database),
			redis.DialConnectTimeout(config.connectTimeout),
			redis.DialReadTimeout(config.readTimeout),
			redis.DialWriteTimeout(config.writeTimeout),
		)

		if err != nil {
			return nil, err
		}

		return &redigoShim{conn}, nil
	}
}

func (s *redigoShim) Close() error {
	return s.conn.Close()
}

func (s *redigoShim) Do(command string, args ...interface{}) (interface{}, error) {
	result, err := s.conn.Do(command, args...)
	return result, s.wrapError(err)
}

func (s *redigoShim) Send(command string, args ...interface{}) error {
	return s.wrapError(s.conn.Send(command, args...))
}

func (s *redigoShim) wrapError(err error) error {
	// If there's an error on the connection, wrap it and return that
	// so we can flag the retry loop in the client to retry instead of
	// returning the error on this attempt.

	if s.conn.Err() != nil {
		return connErr{s.conn.Err()}
	}

	return err
}
