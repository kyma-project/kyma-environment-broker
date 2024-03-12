// Code generated by mockery v2.10.0. DO NOT EDIT.

package automock

import (
	gqlschema "github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	gardener "github.com/kyma-project/kyma-environment-broker/common/gardener"

	internal "github.com/kyma-project/kyma-environment-broker/internal"

	keb "github.com/kyma-incubator/reconciler/pkg/keb"

	mock "github.com/stretchr/testify/mock"
)

// ProvisionerInputCreator is an autogenerated mock type for the ProvisionerInputCreator type
type ProvisionerInputCreator struct {
	mock.Mock
}

// AppendGlobalOverrides provides a mock function with given fields: overrides
func (_m *ProvisionerInputCreator) AppendGlobalOverrides(overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	ret := _m.Called(overrides)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func([]*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator); ok {
		r0 = rf(overrides)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// AppendOverrides provides a mock function with given fields: component, overrides
func (_m *ProvisionerInputCreator) AppendOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	ret := _m.Called(component, overrides)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string, []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator); ok {
		r0 = rf(component, overrides)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// CreateClusterConfiguration provides a mock function with given fields:
func (_m *ProvisionerInputCreator) CreateClusterConfiguration() (keb.Cluster, error) {
	ret := _m.Called()

	var r0 keb.Cluster
	if rf, ok := ret.Get(0).(func() keb.Cluster); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(keb.Cluster)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateProvisionClusterInput provides a mock function with given fields:
func (_m *ProvisionerInputCreator) CreateProvisionClusterInput() (gqlschema.ProvisionRuntimeInput, error) {
	ret := _m.Called()

	var r0 gqlschema.ProvisionRuntimeInput
	if rf, ok := ret.Get(0).(func() gqlschema.ProvisionRuntimeInput); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gqlschema.ProvisionRuntimeInput)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateProvisionRuntimeInput provides a mock function with given fields:
func (_m *ProvisionerInputCreator) CreateProvisionRuntimeInput() (gqlschema.ProvisionRuntimeInput, error) {
	ret := _m.Called()

	var r0 gqlschema.ProvisionRuntimeInput
	if rf, ok := ret.Get(0).(func() gqlschema.ProvisionRuntimeInput); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gqlschema.ProvisionRuntimeInput)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateUpgradeRuntimeInput provides a mock function with given fields:
func (_m *ProvisionerInputCreator) CreateUpgradeRuntimeInput() (gqlschema.UpgradeRuntimeInput, error) {
	ret := _m.Called()

	var r0 gqlschema.UpgradeRuntimeInput
	if rf, ok := ret.Get(0).(func() gqlschema.UpgradeRuntimeInput); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gqlschema.UpgradeRuntimeInput)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateUpgradeShootInput provides a mock function with given fields:
func (_m *ProvisionerInputCreator) CreateUpgradeShootInput() (gqlschema.UpgradeShootInput, error) {
	ret := _m.Called()

	var r0 gqlschema.UpgradeShootInput
	if rf, ok := ret.Get(0).(func() gqlschema.UpgradeShootInput); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gqlschema.UpgradeShootInput)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DisableOptionalComponent provides a mock function with given fields: componentName
func (_m *ProvisionerInputCreator) DisableOptionalComponent(componentName string) internal.ProvisionerInputCreator {
	ret := _m.Called(componentName)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string) internal.ProvisionerInputCreator); ok {
		r0 = rf(componentName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// EnableOptionalComponent provides a mock function with given fields: componentName
func (_m *ProvisionerInputCreator) EnableOptionalComponent(componentName string) internal.ProvisionerInputCreator {
	ret := _m.Called(componentName)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string) internal.ProvisionerInputCreator); ok {
		r0 = rf(componentName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// Configuration provides a mock function with given fields:
func (_m *ProvisionerInputCreator) Configuration() *internal.ConfigForPlan {
	ret := _m.Called()

	var r0 *internal.ConfigForPlan
	if rf, ok := ret.Get(0).(func() *internal.ConfigForPlan); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(*internal.ConfigForPlan)
	}

	return r0
}

// Provider provides a mock function with given fields:
func (_m *ProvisionerInputCreator) Provider() internal.CloudProvider {
	ret := _m.Called()

	var r0 internal.CloudProvider
	if rf, ok := ret.Get(0).(func() internal.CloudProvider); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(internal.CloudProvider)
	}

	return r0
}

// SetClusterName provides a mock function with given fields: name
func (_m *ProvisionerInputCreator) SetClusterName(name string) internal.ProvisionerInputCreator {
	ret := _m.Called(name)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string) internal.ProvisionerInputCreator); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetInstanceID provides a mock function with given fields: instanceID
func (_m *ProvisionerInputCreator) SetInstanceID(instanceID string) internal.ProvisionerInputCreator {
	ret := _m.Called(instanceID)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string) internal.ProvisionerInputCreator); ok {
		r0 = rf(instanceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetKubeconfig provides a mock function with given fields: kcfg
func (_m *ProvisionerInputCreator) SetKubeconfig(kcfg string) internal.ProvisionerInputCreator {
	ret := _m.Called(kcfg)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string) internal.ProvisionerInputCreator); ok {
		r0 = rf(kcfg)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetLabel provides a mock function with given fields: key, value
func (_m *ProvisionerInputCreator) SetLabel(key string, value string) internal.ProvisionerInputCreator {
	ret := _m.Called(key, value)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string, string) internal.ProvisionerInputCreator); ok {
		r0 = rf(key, value)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetOIDCLastValues provides a mock function with given fields: oidcConfig
func (_m *ProvisionerInputCreator) SetOIDCLastValues(oidcConfig gqlschema.OIDCConfigInput) internal.ProvisionerInputCreator {
	ret := _m.Called(oidcConfig)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(gqlschema.OIDCConfigInput) internal.ProvisionerInputCreator); ok {
		r0 = rf(oidcConfig)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetOverrides provides a mock function with given fields: component, overrides
func (_m *ProvisionerInputCreator) SetOverrides(component string, overrides []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator {
	ret := _m.Called(component, overrides)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string, []*gqlschema.ConfigEntryInput) internal.ProvisionerInputCreator); ok {
		r0 = rf(component, overrides)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetProvisioningParameters provides a mock function with given fields: params
func (_m *ProvisionerInputCreator) SetProvisioningParameters(params internal.ProvisioningParameters) internal.ProvisionerInputCreator {
	ret := _m.Called(params)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(internal.ProvisioningParameters) internal.ProvisionerInputCreator); ok {
		r0 = rf(params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetRuntimeID provides a mock function with given fields: runtimeID
func (_m *ProvisionerInputCreator) SetRuntimeID(runtimeID string) internal.ProvisionerInputCreator {
	ret := _m.Called(runtimeID)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string) internal.ProvisionerInputCreator); ok {
		r0 = rf(runtimeID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetShootDNSProviders provides a mock function with given fields: dnsProviders
func (_m *ProvisionerInputCreator) SetShootDNSProviders(dnsProviders gardener.DNSProvidersData) internal.ProvisionerInputCreator {
	ret := _m.Called(dnsProviders)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(gardener.DNSProvidersData) internal.ProvisionerInputCreator); ok {
		r0 = rf(dnsProviders)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetShootDomain provides a mock function with given fields: shootDomain
func (_m *ProvisionerInputCreator) SetShootDomain(shootDomain string) internal.ProvisionerInputCreator {
	ret := _m.Called(shootDomain)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string) internal.ProvisionerInputCreator); ok {
		r0 = rf(shootDomain)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}

// SetShootName provides a mock function with given fields: _a0
func (_m *ProvisionerInputCreator) SetShootName(_a0 string) internal.ProvisionerInputCreator {
	ret := _m.Called(_a0)

	var r0 internal.ProvisionerInputCreator
	if rf, ok := ret.Get(0).(func(string) internal.ProvisionerInputCreator); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(internal.ProvisionerInputCreator)
		}
	}

	return r0
}
