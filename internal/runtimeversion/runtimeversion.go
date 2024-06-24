package runtimeversion

import (
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type RuntimeVersionConfigurator struct {
	defaultVersion string
	runtimeStateDB storage.RuntimeStates
}

func NewRuntimeVersionConfigurator(defaultVersion string, runtimeStates storage.RuntimeStates) *RuntimeVersionConfigurator {
	if defaultVersion == "" {
		panic("Default version not provided")
	}

	return &RuntimeVersionConfigurator{
		defaultVersion: defaultVersion,
		runtimeStateDB: runtimeStates,
	}
}

func (rvc *RuntimeVersionConfigurator) ForUpdating(op internal.Operation) (*internal.RuntimeVersionData, error) {
	r, err := rvc.runtimeStateDB.GetLatestWithKymaVersionByRuntimeID(op.RuntimeID)
	if dberr.IsNotFound(err) {
		return internal.NewEmptyRuntimeVersion(), nil
	}
	if err != nil {
		return nil, err
	}

	return internal.NewRuntimeVersionFromDefaults(r.GetKymaVersion()), nil
}

func (rvc *RuntimeVersionConfigurator) ForProvisioning(internal.Operation) (*internal.RuntimeVersionData, error) {
	return internal.NewRuntimeVersionFromDefaults(rvc.defaultVersion), nil
}

func (rvc *RuntimeVersionConfigurator) ForUpgrade(internal.UpgradeKymaOperation) (*internal.RuntimeVersionData, error) {
	return internal.NewRuntimeVersionFromDefaults(rvc.defaultVersion), nil
}
