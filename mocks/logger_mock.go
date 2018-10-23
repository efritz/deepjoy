// Code generated by github.com/efritz/go-mockgen; DO NOT EDIT.
// This file was generated by robots at
// 2018-10-23T15:57:11-05:00
// using the command
// $ go-mockgen -f github.com/efritz/deepjoy/iface

package mocks

import iface "github.com/efritz/deepjoy/iface"

// MockLogger is a mock impelementation of the Logger interface (from the
// package github.com/efritz/deepjoy/iface) used for unit testing.
type MockLogger struct {
	// PrintfFunc is an instance of a mock function object controlling the
	// behavior of the method Printf.
	PrintfFunc *LoggerPrintfFunc
}

// NewMockLogger creates a new mock of the Logger interface. All methods
// return zero values for all results, unless overwritten.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		PrintfFunc: &LoggerPrintfFunc{
			defaultHook: func(string, ...interface{}) {
				return
			},
		},
	}
}

// NewMockLoggerFrom creates a new mock of the MockLogger interface. All
// methods delegate to the given implementation, unless overwritten.
func NewMockLoggerFrom(i iface.Logger) *MockLogger {
	return &MockLogger{
		PrintfFunc: &LoggerPrintfFunc{
			defaultHook: i.Printf,
		},
	}
}

// LoggerPrintfFunc describes the behavior when the Printf method of the
// parent MockLogger instance is invoked.
type LoggerPrintfFunc struct {
	defaultHook func(string, ...interface{})
	hooks       []func(string, ...interface{})
	history     []LoggerPrintfFuncCall
}

// Printf delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockLogger) Printf(v0 string, v1 ...interface{}) {
	m.PrintfFunc.nextHook()(v0, v1...)
	m.PrintfFunc.history = append(m.PrintfFunc.history, LoggerPrintfFuncCall{v0, v1})
	return
}

// SetDefaultHook sets function that is called when the Printf method of the
// parent MockLogger instance is invoked and the hook queue is empty.
func (f *LoggerPrintfFunc) SetDefaultHook(hook func(string, ...interface{})) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Printf method of the parent MockLogger instance inovkes the hook at the
// front of the queue and discards it. After the queue is empty, the default
// hook function is invoked for any future action.
func (f *LoggerPrintfFunc) PushHook(hook func(string, ...interface{})) {
	f.hooks = append(f.hooks, hook)
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *LoggerPrintfFunc) SetDefaultReturn() {
	f.SetDefaultHook(func(string, ...interface{}) {
		return
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *LoggerPrintfFunc) PushReturn() {
	f.PushHook(func(string, ...interface{}) {
		return
	})
}

func (f *LoggerPrintfFunc) nextHook() func(string, ...interface{}) {
	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

// History returns a sequence of LoggerPrintfFuncCall objects describing the
// invocations of this function.
func (f *LoggerPrintfFunc) History() []LoggerPrintfFuncCall {
	return f.history
}

// LoggerPrintfFuncCall is an object that describes an invocation of method
// Printf on an instance of MockLogger.
type LoggerPrintfFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 string
	// Arg1 is a slice containing the values of the variadic arguments
	// passed to this method invocation.
	Arg1 []interface{}
}

// Args returns an interface slice containing the arguments of this
// invocation. The variadic slice argument is flattened in this array such
// that one positional argument and three variadic arguments would result in
// a slice of four, not two.
func (c LoggerPrintfFuncCall) Args() []interface{} {
	return append([]interface{}{c.Arg0}, c.Arg1...)
}

// Results returns an interface slice containing the results of this
// invocation.
func (c LoggerPrintfFuncCall) Results() []interface{} {
	return []interface{}{}
}
