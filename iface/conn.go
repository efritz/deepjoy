package iface

// Conn abstracts a single, feature-minimal connection to Redis.
type Conn interface {
	// Close the connection to the remote Redis server.
	Close() error

	// Do performs a command on the remote Redis server and returns
	// its result.
	Do(command string, args ...interface{}) (interface{}, error)

	// Send will publish command as part of a MULTI/EXEC sequence
	// to the remote Redis server.
	Send(command string, args ...interface{}) error
}
