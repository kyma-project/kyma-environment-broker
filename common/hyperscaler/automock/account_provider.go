// Code generated by mockery v2.14.0. DO NOT EDIT.

package automock

import (
	hyperscaler "github.com/kyma-project/kyma-environment-broker/common/hyperscaler"
	mock "github.com/stretchr/testify/mock"
)

// AccountProvider is an autogenerated mock type for the AccountProvider type
type AccountProvider struct {
	mock.Mock
}

// GardenerSecretName provides a mock function with given fields: hyperscalerType, tenantName, euAccess
func (_m *AccountProvider) GardenerSecretName(hyperscalerType hyperscaler.Type, tenantName string, euAccess bool, shared bool) (string, error) {
	ret := _m.Called(hyperscalerType, tenantName, euAccess, shared)

	var r0 string
	if rf, ok := ret.Get(0).(func(hyperscaler.Type, string, bool, bool) string); ok {
		r0 = rf(hyperscalerType, tenantName, euAccess, shared)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(hyperscaler.Type, string, bool, bool) error); ok {
		r1 = rf(hyperscalerType, tenantName, euAccess, shared)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GardenerSharedSecretName provides a mock function with given fields: hyperscalerType, euAccess
func (_m *AccountProvider) GardenerSharedSecretName(hyperscalerType hyperscaler.Type, euAccess bool) (string, error) {
	ret := _m.Called(hyperscalerType, euAccess)

	var r0 string
	if rf, ok := ret.Get(0).(func(hyperscaler.Type, bool) string); ok {
		r0 = rf(hyperscalerType, euAccess)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(hyperscaler.Type, bool) error); ok {
		r1 = rf(hyperscalerType, euAccess)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MarkUnusedGardenerSecretBindingAsDirty provides a mock function with given fields: hyperscalerType, tenantName, euAccess
func (_m *AccountProvider) MarkUnusedGardenerSecretBindingAsDirty(hyperscalerType hyperscaler.Type, tenantName string, euAccess bool) error {
	ret := _m.Called(hyperscalerType, tenantName, euAccess)

	var r0 error
	if rf, ok := ret.Get(0).(func(hyperscaler.Type, string, bool) error); ok {
		r0 = rf(hyperscalerType, tenantName, euAccess)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewAccountProvider interface {
	mock.TestingT
	Cleanup(func())
}

// NewAccountProvider creates a new instance of AccountProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewAccountProvider(t mockConstructorTestingTNewAccountProvider) *AccountProvider {
	mock := &AccountProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
