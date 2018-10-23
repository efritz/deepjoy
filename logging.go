package deepjoy

import (
	"log"

	"github.com/efritz/deepjoy/iface"
)

type (
	// Logger is an interface to the logger the client writes to.
	Logger = iface.Logger

	printLogger struct{}
	nilLogger   struct{}
)

// NilLogger is a singleton silent logger.
var NilLogger = NewNilLogger()

// NewPrintLogger creates a logger that prints to stdout.
func NewPrintLogger() Logger {
	return &printLogger{}
}

// NewNilLogger creates a silent logger.
func NewNilLogger() Logger {
	return &nilLogger{}
}

func (l *printLogger) Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (l *nilLogger) Printf(format string, args ...interface{}) {
}
