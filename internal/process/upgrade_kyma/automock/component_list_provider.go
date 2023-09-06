// Code generated by mockery v2.9.4. DO NOT EDIT.

package automock

import (
	internal "github.com/kyma-project/kyma-environment-broker/internal"
	mock "github.com/stretchr/testify/mock"

	v1alpha1 "github.com/kyma-project/kyma/components/kyma-operator/pkg/apis/installer/v1alpha1"
)

// ComponentListProvider is an autogenerated mock type for the ComponentListProvider type
type ComponentListProvider struct {
	mock.Mock
}

// AllComponents provides a mock function with given fields: kymaVersion
func (_m *ComponentListProvider) AllComponents(kymaVersion internal.RuntimeVersionData) ([]v1alpha1.KymaComponent, error) {
	ret := _m.Called(kymaVersion)

	var r0 []v1alpha1.KymaComponent
	if rf, ok := ret.Get(0).(func(internal.RuntimeVersionData) []v1alpha1.KymaComponent); ok {
		r0 = rf(kymaVersion)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]v1alpha1.KymaComponent)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(internal.RuntimeVersionData) error); ok {
		r1 = rf(kymaVersion)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
