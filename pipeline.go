package deepjoy

import "github.com/efritz/deepjoy/iface"

type (
	// Pipeline wraps an ordered sequence of commands to be processed
	// with a single request/response exchange. This reduces bandwidth
	// and latency around communication with the remote server.
	Pipeline = iface.Pipeline

	pipeline struct {
		client   *client
		commands []commandPair
	}

	commandPair struct {
		command string
		args    []interface{}
	}
)

func newPipeline(client *client) Pipeline {
	return &pipeline{
		client:   client,
		commands: []commandPair{},
	}
}

// Add will attach a command to this pipeline. This command is
// not sent to the remote server until Run is invoked.
func (p *pipeline) Add(command string, args ...interface{}) {
	p.commands = append(p.commands, commandPair{
		command: command,
		args:    args,
	})
}

// Run will send all commands attached to this pipeline in a
// single request and return a slice of the results of each
// command.
func (p *pipeline) Run() (interface{}, error) {
	return p.client.pipeline(p.commands)
}
