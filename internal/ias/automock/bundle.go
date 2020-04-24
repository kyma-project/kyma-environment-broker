// Code generated by mockery v1.0.0. DO NOT EDIT.

package automock

import ias "github.com/kyma-incubator/compass/components/kyma-environment-broker/internal/ias"
import mock "github.com/stretchr/testify/mock"

// Bundle is an autogenerated mock type for the Bundle type
type Bundle struct {
	mock.Mock
}

// ConfigureServiceProvider provides a mock function with given fields:
func (_m *Bundle) ConfigureServiceProvider() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ConfigureServiceProviderType provides a mock function with given fields: path
func (_m *Bundle) ConfigureServiceProviderType(path string) error {
	ret := _m.Called(path)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(path)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateServiceProvider provides a mock function with given fields:
func (_m *Bundle) CreateServiceProvider() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteServiceProvider provides a mock function with given fields:
func (_m *Bundle) DeleteServiceProvider() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FetchServiceProviderData provides a mock function with given fields:
func (_m *Bundle) FetchServiceProviderData() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GenerateSecret provides a mock function with given fields:
func (_m *Bundle) GenerateSecret() (*ias.ServiceProviderSecret, error) {
	ret := _m.Called()

	var r0 *ias.ServiceProviderSecret
	if rf, ok := ret.Get(0).(func() *ias.ServiceProviderSecret); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ias.ServiceProviderSecret)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ServiceProviderExist provides a mock function with given fields:
func (_m *Bundle) ServiceProviderExist() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// ServiceProviderName provides a mock function with given fields:
func (_m *Bundle) ServiceProviderName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
