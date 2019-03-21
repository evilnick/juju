// Code generated by MockGen. DO NOT EDIT.
// Source: gopkg.in/juju/worker.v1 (interfaces: Worker)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockWorker is a mock of Worker interface
type MockWorker struct {
	ctrl     *gomock.Controller
	recorder *MockWorkerMockRecorder
}

// MockWorkerMockRecorder is the mock recorder for MockWorker
type MockWorkerMockRecorder struct {
	mock *MockWorker
}

// NewMockWorker creates a new mock instance
func NewMockWorker(ctrl *gomock.Controller) *MockWorker {
	mock := &MockWorker{ctrl: ctrl}
	mock.recorder = &MockWorkerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockWorker) EXPECT() *MockWorkerMockRecorder {
	return m.recorder
}

// Kill mocks base method
func (m *MockWorker) Kill() {
	m.ctrl.Call(m, "Kill")
}

// Kill indicates an expected call of Kill
func (mr *MockWorkerMockRecorder) Kill() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Kill", reflect.TypeOf((*MockWorker)(nil).Kill))
}

// Wait mocks base method
func (m *MockWorker) Wait() error {
	ret := m.ctrl.Call(m, "Wait")
	ret0, _ := ret[0].(error)
	return ret0
}

// Wait indicates an expected call of Wait
func (mr *MockWorkerMockRecorder) Wait() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Wait", reflect.TypeOf((*MockWorker)(nil).Wait))
}