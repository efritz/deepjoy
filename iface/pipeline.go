package iface

// Pipeline wraps an ordered sequence of commands to be processed
// with a single request/response exchange. This reduces bandwidth
// and latency around communication with the remote server.
type Pipeline interface {
	// Add will attach a command to this pipeline. This command is
	// not sent to the remote server until Run is invoked.
	Add(command string, args ...interface{})

	// Run will send all commands attached to this pipeline in a
	// single request and return a slice of the results of each
	// command.
	Run() (interface{}, error)
}
