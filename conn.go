package deepjoy

import (
	"math/rand"

	"github.com/gomodule/redigo/redis"

	"github.com/efritz/deepjoy/iface"
)

type (
	// Conn abstracts a single, feature-minimal connection to Redis.
	Conn = iface.Conn

	// DialFunc creates a connection to Redis or returns an error.
	DialFunc func() (Conn, error)

	// DialerFactory creates a DialFunc for the given address.
	DialerFactory func(addrs []string) DialFunc

	redigoShim struct {
		conn redis.Conn
	}

	connErr struct{ error }
)

func makeDefaultDialerFactory(config *clientConfig) DialerFactory {
	return func(addrs []string) DialFunc {
		return func() (Conn, error) {
			addr := chooseRandom(addrs)

			config.logger.Printf("Attempting to dial redis at %s", addr)

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
}

func chooseRandom(addrs []string) string {
	if len(addrs) == 0 {
		return ""
	}

	return addrs[rand.Intn(len(addrs))]
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
