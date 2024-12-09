package orchestration

import (
	"fmt"
	"log/slog"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	runtimeInt "github.com/kyma-project/kyma-environment-broker/internal/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
)

type RuntimeLister struct {
	instancesDb  storage.Instances
	operationsDb storage.Operations
	converter    runtimeInt.Converter
	log          *slog.Logger
}

func NewRuntimeLister(instancesDb storage.Instances, operationsDb storage.Operations, converter runtimeInt.Converter, log *slog.Logger) *RuntimeLister {
	return &RuntimeLister{
		instancesDb:  instancesDb,
		operationsDb: operationsDb,
		converter:    converter,
		log:          log,
	}
}

func (rl RuntimeLister) ListAllRuntimes() ([]runtime.RuntimeDTO, error) {
	instances, _, _, err := rl.instancesDb.List(dbmodel.InstanceFilter{})
	if err != nil {
		return nil, fmt.Errorf("while listing instances from DB: %w", err)
	}

	runtimes := make([]runtime.RuntimeDTO, 0, len(instances))
	for _, inst := range instances {
		dto, err := rl.converter.NewDTO(inst)
		if err != nil {
			rl.log.Error(fmt.Sprintf("cannot convert instance to DTO: %s", err.Error()))
			continue
		}

		pOprs, err := rl.operationsDb.ListProvisioningOperationsByInstanceID(inst.InstanceID)
		if err != nil {
			rl.log.Error(fmt.Sprintf("while getting provision operation for instance %s: %s", inst.InstanceID, err.Error()))
			continue
		}
		if len(pOprs) > 0 {
			rl.converter.ApplyProvisioningOperation(&dto, &pOprs[len(pOprs)-1])
		}
		if len(pOprs) > 1 {
			rl.converter.ApplyUnsuspensionOperations(&dto, pOprs[:len(pOprs)-1])
		}

		dOprs, err := rl.operationsDb.ListDeprovisioningOperationsByInstanceID(inst.InstanceID)
		if err != nil && !dberr.IsNotFound(err) {
			rl.log.Error(fmt.Sprintf("while getting deprovision operation for instance %s: %s", inst.InstanceID, err.Error()))
			continue
		}
		if len(dOprs) > 0 {
			rl.converter.ApplyDeprovisioningOperation(&dto, &dOprs[0])
		}

		rl.converter.ApplySuspensionOperations(&dto, dOprs)

		runtimes = append(runtimes, dto)
	}

	return runtimes, nil
}
