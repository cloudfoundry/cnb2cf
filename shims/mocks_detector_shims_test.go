// Code generated by MockGen. DO NOT EDIT.
// Source: detector.go

// Package shims_test is a generated GoMock package.
package shims_test

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockEnvironment is a mock of Environment interface
type MockEnvironment struct {
	ctrl     *gomock.Controller
	recorder *MockEnvironmentMockRecorder
}

// MockEnvironmentMockRecorder is the mock recorder for MockEnvironment
type MockEnvironmentMockRecorder struct {
	mock *MockEnvironment
}

// NewMockEnvironment creates a new mock instance
func NewMockEnvironment(ctrl *gomock.Controller) *MockEnvironment {
	mock := &MockEnvironment{ctrl: ctrl}
	mock.recorder = &MockEnvironmentMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockEnvironment) EXPECT() *MockEnvironmentMockRecorder {
	return m.recorder
}

// Services mocks base method
func (m *MockEnvironment) Services() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Services")
	ret0, _ := ret[0].(string)
	return ret0
}

// Services indicates an expected call of Services
func (mr *MockEnvironmentMockRecorder) Services() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Services", reflect.TypeOf((*MockEnvironment)(nil).Services))
}

// Stack mocks base method
func (m *MockEnvironment) Stack() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stack")
	ret0, _ := ret[0].(string)
	return ret0
}

// Stack indicates an expected call of Stack
func (mr *MockEnvironmentMockRecorder) Stack() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stack", reflect.TypeOf((*MockEnvironment)(nil).Stack))
}

// MockInstaller is a mock of Installer interface
type MockInstaller struct {
	ctrl     *gomock.Controller
	recorder *MockInstallerMockRecorder
}

// MockInstallerMockRecorder is the mock recorder for MockInstaller
type MockInstallerMockRecorder struct {
	mock *MockInstaller
}

// NewMockInstaller creates a new mock instance
func NewMockInstaller(ctrl *gomock.Controller) *MockInstaller {
	mock := &MockInstaller{ctrl: ctrl}
	mock.recorder = &MockInstallerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockInstaller) EXPECT() *MockInstallerMockRecorder {
	return m.recorder
}

// InstallCNBs mocks base method
func (m *MockInstaller) InstallCNBs(orderFile, installDir string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstallCNBs", orderFile, installDir)
	ret0, _ := ret[0].(error)
	return ret0
}

// InstallCNBs indicates an expected call of InstallCNBs
func (mr *MockInstallerMockRecorder) InstallCNBs(orderFile, installDir interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstallCNBs", reflect.TypeOf((*MockInstaller)(nil).InstallCNBs), orderFile, installDir)
}

// InstallLifecycle mocks base method
func (m *MockInstaller) InstallLifecycle(dst string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InstallLifecycle", dst)
	ret0, _ := ret[0].(error)
	return ret0
}

// InstallLifecycle indicates an expected call of InstallLifecycle
func (mr *MockInstallerMockRecorder) InstallLifecycle(dst interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InstallLifecycle", reflect.TypeOf((*MockInstaller)(nil).InstallLifecycle), dst)
}
