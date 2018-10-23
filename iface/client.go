package iface

// Client is a goroutine-safe, minimal, and pooled Redis client.
type Client interface {
	// Close will close all open connections to the remote Redis server.
	Close()

	// ReadReplica returns a host that points to the set of configured
	// read replicas. If no read replicas are configured, this returns
	// the current client. The client returned from this method does NOT
	// need to be independently closed (closing the source client will
	// also close replica clients).
	ReadReplica() Client

	// Do runs the command on the remote Redis server and returns its raw
	// response.
	Do(command string, args ...interface{}) (interface{}, error)

	// Pipeline returns a builder object to which commands can be attached.
	// All commands in the pipeline are sent to the remote server in a
	// single request and all results will be returned in a single response.
	// The MULTI/EXEC commands are added implicitly by the client. A pipeline
	// does NOT guarantee atomicity. If you require multiple commands to be
	// run atomically, bundle them in a Lua script and run it on the remote
	// server with the EVAL command.
	Pipeline() Pipeline
}
