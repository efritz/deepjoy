package deepjoy

import "log"

type (
	// Logger is an interface to the logger the client writes to.
	Logger interface {
		// Prints logs a message. Arguments should be handled in the manner of fmt.Printf.
		Printf(format string, args ...interface{})
	}

	defaultLogger struct{}
)

func (l *defaultLogger) Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}
