package iface

// Logger is an interface to the logger the client writes to.
type Logger interface {
	// Printf logs a message. Arguments should be handled in the manner of fmt.Printf.
	Printf(format string, args ...interface{})
}
