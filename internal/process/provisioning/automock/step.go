// Code generated by mockery v1.0.0. DO NOT EDIT.

package automock

import (
	internal "github.com/kyma-project/kyma-environment-broker/internal"
	logrus "github.com/sirupsen/logrus"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// Step is an autogenerated mock type for the Step type
type Step struct {
	mock.Mock
}

// Name provides a mock function with given fields:
func (_m *Step) Name() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Run provides a mock function with given fields: operation, logger
func (_m *Step) Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error) {
	ret := _m.Called(operation, logger)

	var r0 internal.Operation
	if rf, ok := ret.Get(0).(func(internal.Operation, logrus.FieldLogger) internal.Operation); ok {
		r0 = rf(operation, logger)
	} else {
		r0 = ret.Get(0).(internal.Operation)
	}

	var r1 time.Duration
	if rf, ok := ret.Get(1).(func(internal.Operation, logrus.FieldLogger) time.Duration); ok {
		r1 = rf(operation, logger)
	} else {
		r1 = ret.Get(1).(time.Duration)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(internal.Operation, logrus.FieldLogger) error); ok {
		r2 = rf(operation, logger)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
