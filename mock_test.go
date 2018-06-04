// DO NOT EDIT
// Code generated automatically by github.com/efritz/go-mockgen
// $ go-mockgen github.com/efritz/deepjoy -o mock_test.go -i Conn -i Pool

package deepjoy

import time "time"

type MockConn struct {
	SendFunc            func(string, ...interface{}) error
	SendFuncCallCount   int
	SendFuncCallParams  []ConnSendParamSet
	CloseFunc           func() error
	CloseFuncCallCount  int
	CloseFuncCallParams []ConnCloseParamSet
	DoFunc              func(string, ...interface{}) (interface{}, error)
	DoFuncCallCount     int
	DoFuncCallParams    []ConnDoParamSet
}
type ConnCloseParamSet struct{}
type ConnDoParamSet struct {
	Arg0 string
	Arg1 []interface{}
}
type ConnSendParamSet struct {
	Arg0 string
	Arg1 []interface{}
}

var _ Conn = NewMockConn()

func NewMockConn() *MockConn {
	m := &MockConn{}
	m.CloseFunc = m.defaultCloseFunc
	m.DoFunc = m.defaultDoFunc
	m.SendFunc = m.defaultSendFunc
	return m
}
func (m *MockConn) Close() error {
	m.CloseFuncCallCount++
	m.CloseFuncCallParams = append(m.CloseFuncCallParams, ConnCloseParamSet{})
	return m.CloseFunc()
}
func (m *MockConn) Do(v0 string, v1 ...interface{}) (interface{}, error) {
	m.DoFuncCallCount++
	m.DoFuncCallParams = append(m.DoFuncCallParams, ConnDoParamSet{v0, v1})
	return m.DoFunc(v0, v1...)
}
func (m *MockConn) Send(v0 string, v1 ...interface{}) error {
	m.SendFuncCallCount++
	m.SendFuncCallParams = append(m.SendFuncCallParams, ConnSendParamSet{v0, v1})
	return m.SendFunc(v0, v1...)
}
func (m *MockConn) defaultDoFunc(v0 string, v1 ...interface{}) (interface{}, error) {
	return nil, nil
}
func (m *MockConn) defaultSendFunc(v0 string, v1 ...interface{}) error {
	return nil
}
func (m *MockConn) defaultCloseFunc() error {
	return nil
}

type MockPool struct {
	BorrowTimeoutFunc           func(time.Duration) (Conn, bool)
	BorrowTimeoutFuncCallCount  int
	BorrowTimeoutFuncCallParams []PoolBorrowTimeoutParamSet
	CloseFunc                   func()
	CloseFuncCallCount          int
	CloseFuncCallParams         []PoolCloseParamSet
	ReleaseFunc                 func(Conn)
	ReleaseFuncCallCount        int
	ReleaseFuncCallParams       []PoolReleaseParamSet
	BorrowFunc                  func() (Conn, bool)
	BorrowFuncCallCount         int
	BorrowFuncCallParams        []PoolBorrowParamSet
}
type PoolBorrowParamSet struct{}
type PoolBorrowTimeoutParamSet struct {
	Arg0 time.Duration
}
type PoolCloseParamSet struct{}
type PoolReleaseParamSet struct {
	Arg0 Conn
}

var _ Pool = NewMockPool()

func NewMockPool() *MockPool {
	m := &MockPool{}
	m.ReleaseFunc = m.defaultReleaseFunc
	m.BorrowFunc = m.defaultBorrowFunc
	m.BorrowTimeoutFunc = m.defaultBorrowTimeoutFunc
	m.CloseFunc = m.defaultCloseFunc
	return m
}
func (m *MockPool) Borrow() (Conn, bool) {
	m.BorrowFuncCallCount++
	m.BorrowFuncCallParams = append(m.BorrowFuncCallParams, PoolBorrowParamSet{})
	return m.BorrowFunc()
}
func (m *MockPool) BorrowTimeout(v0 time.Duration) (Conn, bool) {
	m.BorrowTimeoutFuncCallCount++
	m.BorrowTimeoutFuncCallParams = append(m.BorrowTimeoutFuncCallParams, PoolBorrowTimeoutParamSet{v0})
	return m.BorrowTimeoutFunc(v0)
}
func (m *MockPool) Close() {
	m.CloseFuncCallCount++
	m.CloseFuncCallParams = append(m.CloseFuncCallParams, PoolCloseParamSet{})
	m.CloseFunc()
}
func (m *MockPool) Release(v0 Conn) {
	m.ReleaseFuncCallCount++
	m.ReleaseFuncCallParams = append(m.ReleaseFuncCallParams, PoolReleaseParamSet{v0})
	m.ReleaseFunc(v0)
}
func (m *MockPool) defaultBorrowFunc() (Conn, bool) {
	return nil, false
}
func (m *MockPool) defaultBorrowTimeoutFunc(v0 time.Duration) (Conn, bool) {
	return nil, false
}
func (m *MockPool) defaultCloseFunc() {
	return
}
func (m *MockPool) defaultReleaseFunc(v0 Conn) {
	return
}
