// DO NOT EDIT
// Code generated automatically by github.com/efritz/go-mockgen
// $ go-mockgen github.com/efritz/deepjoy -f -o mock_test.go -i Conn -i Pool

package deepjoy

import time "time"

type MockConn struct {
	CloseFunc func() error
	DoFunc    func(string, ...interface{}) (interface{}, error)
	SendFunc  func(string, ...interface{}) error
}

var _ Conn = NewMockConn()

func NewMockConn() *MockConn {
	return &MockConn{CloseFunc: func() error {
		return nil
	}, DoFunc: func(string, ...interface{}) (interface{}, error) {
		return nil, nil
	}, SendFunc: func(string, ...interface{}) error {
		return nil
	}}
}
func (m *MockConn) Close() error {
	return m.CloseFunc()
}
func (m *MockConn) Do(v0 string, v1 ...interface{}) (interface{}, error) {
	return m.DoFunc(v0, v1...)
}
func (m *MockConn) Send(v0 string, v1 ...interface{}) error {
	return m.SendFunc(v0, v1...)
}

type MockPool struct {
	BorrowFunc        func() (Conn, bool)
	BorrowTimeoutFunc func(time.Duration) (Conn, bool)
	CloseFunc         func()
	ReleaseFunc       func(Conn)
}

var _ Pool = NewMockPool()

func NewMockPool() *MockPool {
	return &MockPool{BorrowFunc: func() (Conn, bool) {
		return nil, false
	}, BorrowTimeoutFunc: func(time.Duration) (Conn, bool) {
		return nil, false
	}, CloseFunc: func() {}, ReleaseFunc: func(Conn) {}}
}
func (m *MockPool) Borrow() (Conn, bool) {
	return m.BorrowFunc()
}
func (m *MockPool) BorrowTimeout(v0 time.Duration) (Conn, bool) {
	return m.BorrowTimeoutFunc(v0)
}
func (m *MockPool) Close() {
	m.CloseFunc()
}
func (m *MockPool) Release(v0 Conn) {
	m.ReleaseFunc(v0)
}
