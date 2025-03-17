// Code generated by mockery v2.43.0. DO NOT EDIT.

package automock

import mock "github.com/stretchr/testify/mock"

// Queue is an autogenerated mock type for the Queue type
type Queue struct {
	mock.Mock
}

// Add provides a mock function with given fields: operationId
func (_m *Queue) Add(operationId string) {
	_m.Called(operationId)
}

// NewQueue creates a new instance of Queue. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewQueue(t interface {
	mock.TestingT
	Cleanup(func())
}) *Queue {
	mock := &Queue{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
