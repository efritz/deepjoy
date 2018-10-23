// Code generated by github.com/efritz/go-mockgen; DO NOT EDIT.
// This file was generated by robots at
// 2018-10-23T15:57:11-05:00
// using the command
// $ go-mockgen -f github.com/efritz/deepjoy/iface

package mocks

import (
	iface "github.com/efritz/deepjoy/iface"
	"time"
)

// MockPool is a mock impelementation of the Pool interface (from the
// package github.com/efritz/deepjoy/iface) used for unit testing.
type MockPool struct {
	// BorrowFunc is an instance of a mock function object controlling the
	// behavior of the method Borrow.
	BorrowFunc *PoolBorrowFunc
	// BorrowTimeoutFunc is an instance of a mock function object
	// controlling the behavior of the method BorrowTimeout.
	BorrowTimeoutFunc *PoolBorrowTimeoutFunc
	// CloseFunc is an instance of a mock function object controlling the
	// behavior of the method Close.
	CloseFunc *PoolCloseFunc
	// ReleaseFunc is an instance of a mock function object controlling the
	// behavior of the method Release.
	ReleaseFunc *PoolReleaseFunc
}

// NewMockPool creates a new mock of the Pool interface. All methods return
// zero values for all results, unless overwritten.
func NewMockPool() *MockPool {
	return &MockPool{
		BorrowFunc: &PoolBorrowFunc{
			defaultHook: func() (iface.Conn, bool) {
				return nil, false
			},
		},
		BorrowTimeoutFunc: &PoolBorrowTimeoutFunc{
			defaultHook: func(time.Duration) (iface.Conn, bool) {
				return nil, false
			},
		},
		CloseFunc: &PoolCloseFunc{
			defaultHook: func() {
				return
			},
		},
		ReleaseFunc: &PoolReleaseFunc{
			defaultHook: func(iface.Conn) {
				return
			},
		},
	}
}

// NewMockPoolFrom creates a new mock of the MockPool interface. All methods
// delegate to the given implementation, unless overwritten.
func NewMockPoolFrom(i iface.Pool) *MockPool {
	return &MockPool{
		BorrowFunc: &PoolBorrowFunc{
			defaultHook: i.Borrow,
		},
		BorrowTimeoutFunc: &PoolBorrowTimeoutFunc{
			defaultHook: i.BorrowTimeout,
		},
		CloseFunc: &PoolCloseFunc{
			defaultHook: i.Close,
		},
		ReleaseFunc: &PoolReleaseFunc{
			defaultHook: i.Release,
		},
	}
}

// PoolBorrowFunc describes the behavior when the Borrow method of the
// parent MockPool instance is invoked.
type PoolBorrowFunc struct {
	defaultHook func() (iface.Conn, bool)
	hooks       []func() (iface.Conn, bool)
	history     []PoolBorrowFuncCall
}

// Borrow delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockPool) Borrow() (iface.Conn, bool) {
	r0, r1 := m.BorrowFunc.nextHook()()
	m.BorrowFunc.history = append(m.BorrowFunc.history, PoolBorrowFuncCall{r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the Borrow method of the
// parent MockPool instance is invoked and the hook queue is empty.
func (f *PoolBorrowFunc) SetDefaultHook(hook func() (iface.Conn, bool)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Borrow method of the parent MockPool instance inovkes the hook at the
// front of the queue and discards it. After the queue is empty, the default
// hook function is invoked for any future action.
func (f *PoolBorrowFunc) PushHook(hook func() (iface.Conn, bool)) {
	f.hooks = append(f.hooks, hook)
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *PoolBorrowFunc) SetDefaultReturn(r0 iface.Conn, r1 bool) {
	f.SetDefaultHook(func() (iface.Conn, bool) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *PoolBorrowFunc) PushReturn(r0 iface.Conn, r1 bool) {
	f.PushHook(func() (iface.Conn, bool) {
		return r0, r1
	})
}

func (f *PoolBorrowFunc) nextHook() func() (iface.Conn, bool) {
	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

// History returns a sequence of PoolBorrowFuncCall objects describing the
// invocations of this function.
func (f *PoolBorrowFunc) History() []PoolBorrowFuncCall {
	return f.history
}

// PoolBorrowFuncCall is an object that describes an invocation of method
// Borrow on an instance of MockPool.
type PoolBorrowFuncCall struct {
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 iface.Conn
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 bool
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c PoolBorrowFuncCall) Args() []interface{} {
	return []interface{}{}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c PoolBorrowFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

// PoolBorrowTimeoutFunc describes the behavior when the BorrowTimeout
// method of the parent MockPool instance is invoked.
type PoolBorrowTimeoutFunc struct {
	defaultHook func(time.Duration) (iface.Conn, bool)
	hooks       []func(time.Duration) (iface.Conn, bool)
	history     []PoolBorrowTimeoutFuncCall
}

// BorrowTimeout delegates to the next hook function in the queue and stores
// the parameter and result values of this invocation.
func (m *MockPool) BorrowTimeout(v0 time.Duration) (iface.Conn, bool) {
	r0, r1 := m.BorrowTimeoutFunc.nextHook()(v0)
	m.BorrowTimeoutFunc.history = append(m.BorrowTimeoutFunc.history, PoolBorrowTimeoutFuncCall{v0, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the BorrowTimeout method
// of the parent MockPool instance is invoked and the hook queue is empty.
func (f *PoolBorrowTimeoutFunc) SetDefaultHook(hook func(time.Duration) (iface.Conn, bool)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// BorrowTimeout method of the parent MockPool instance inovkes the hook at
// the front of the queue and discards it. After the queue is empty, the
// default hook function is invoked for any future action.
func (f *PoolBorrowTimeoutFunc) PushHook(hook func(time.Duration) (iface.Conn, bool)) {
	f.hooks = append(f.hooks, hook)
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *PoolBorrowTimeoutFunc) SetDefaultReturn(r0 iface.Conn, r1 bool) {
	f.SetDefaultHook(func(time.Duration) (iface.Conn, bool) {
		return r0, r1
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *PoolBorrowTimeoutFunc) PushReturn(r0 iface.Conn, r1 bool) {
	f.PushHook(func(time.Duration) (iface.Conn, bool) {
		return r0, r1
	})
}

func (f *PoolBorrowTimeoutFunc) nextHook() func(time.Duration) (iface.Conn, bool) {
	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

// History returns a sequence of PoolBorrowTimeoutFuncCall objects
// describing the invocations of this function.
func (f *PoolBorrowTimeoutFunc) History() []PoolBorrowTimeoutFuncCall {
	return f.history
}

// PoolBorrowTimeoutFuncCall is an object that describes an invocation of
// method BorrowTimeout on an instance of MockPool.
type PoolBorrowTimeoutFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 time.Duration
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 iface.Conn
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 bool
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c PoolBorrowTimeoutFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c PoolBorrowTimeoutFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

// PoolCloseFunc describes the behavior when the Close method of the parent
// MockPool instance is invoked.
type PoolCloseFunc struct {
	defaultHook func()
	hooks       []func()
	history     []PoolCloseFuncCall
}

// Close delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockPool) Close() {
	m.CloseFunc.nextHook()()
	m.CloseFunc.history = append(m.CloseFunc.history, PoolCloseFuncCall{})
	return
}

// SetDefaultHook sets function that is called when the Close method of the
// parent MockPool instance is invoked and the hook queue is empty.
func (f *PoolCloseFunc) SetDefaultHook(hook func()) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Close method of the parent MockPool instance inovkes the hook at the
// front of the queue and discards it. After the queue is empty, the default
// hook function is invoked for any future action.
func (f *PoolCloseFunc) PushHook(hook func()) {
	f.hooks = append(f.hooks, hook)
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *PoolCloseFunc) SetDefaultReturn() {
	f.SetDefaultHook(func() {
		return
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *PoolCloseFunc) PushReturn() {
	f.PushHook(func() {
		return
	})
}

func (f *PoolCloseFunc) nextHook() func() {
	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

// History returns a sequence of PoolCloseFuncCall objects describing the
// invocations of this function.
func (f *PoolCloseFunc) History() []PoolCloseFuncCall {
	return f.history
}

// PoolCloseFuncCall is an object that describes an invocation of method
// Close on an instance of MockPool.
type PoolCloseFuncCall struct{}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c PoolCloseFuncCall) Args() []interface{} {
	return []interface{}{}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c PoolCloseFuncCall) Results() []interface{} {
	return []interface{}{}
}

// PoolReleaseFunc describes the behavior when the Release method of the
// parent MockPool instance is invoked.
type PoolReleaseFunc struct {
	defaultHook func(iface.Conn)
	hooks       []func(iface.Conn)
	history     []PoolReleaseFuncCall
}

// Release delegates to the next hook function in the queue and stores the
// parameter and result values of this invocation.
func (m *MockPool) Release(v0 iface.Conn) {
	m.ReleaseFunc.nextHook()(v0)
	m.ReleaseFunc.history = append(m.ReleaseFunc.history, PoolReleaseFuncCall{v0})
	return
}

// SetDefaultHook sets function that is called when the Release method of
// the parent MockPool instance is invoked and the hook queue is empty.
func (f *PoolReleaseFunc) SetDefaultHook(hook func(iface.Conn)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// Release method of the parent MockPool instance inovkes the hook at the
// front of the queue and discards it. After the queue is empty, the default
// hook function is invoked for any future action.
func (f *PoolReleaseFunc) PushHook(hook func(iface.Conn)) {
	f.hooks = append(f.hooks, hook)
}

// SetDefaultReturn calls SetDefaultDefaultHook with a function that returns
// the given values.
func (f *PoolReleaseFunc) SetDefaultReturn() {
	f.SetDefaultHook(func(iface.Conn) {
		return
	})
}

// PushReturn calls PushDefaultHook with a function that returns the given
// values.
func (f *PoolReleaseFunc) PushReturn() {
	f.PushHook(func(iface.Conn) {
		return
	})
}

func (f *PoolReleaseFunc) nextHook() func(iface.Conn) {
	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

// History returns a sequence of PoolReleaseFuncCall objects describing the
// invocations of this function.
func (f *PoolReleaseFunc) History() []PoolReleaseFuncCall {
	return f.history
}

// PoolReleaseFuncCall is an object that describes an invocation of method
// Release on an instance of MockPool.
type PoolReleaseFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 iface.Conn
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c PoolReleaseFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c PoolReleaseFuncCall) Results() []interface{} {
	return []interface{}{}
}
