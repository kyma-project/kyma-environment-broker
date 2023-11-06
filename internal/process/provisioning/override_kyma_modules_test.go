package provisioning

import (
	"testing"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	technicalNameBtpManager         = "btp-operator"
	technicalNameKeda               = "keda"
	givenKymaTemplateWithModules    = "kyma-with-keda-and-btp-operator.yaml"
	defaultModules                  = givenKymaTemplateWithModules
	givenKymaTemplateWithoutModules = "kyma-no-modules.yaml"
	kymaExpectedNamespace           = "kyma-system"
)

func execTest(t *testing.T, parameters *internal.ModulesDTO, in, out string) {
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation(uuid.NewString(), uuid.NewString(), internal.OperationTypeProvision)
	operation.KymaTemplate = internal.GetKymaTemplateForTests(t, in)
	expectedKymaTemplate := internal.GetKymaTemplateForTests(t, out)
	operation.ProvisioningParameters.Parameters.Modules = parameters
	err := db.Operations().InsertOperation(operation)
	assert.NoError(t, err)
	svc := NewOverrideKymaModules(db.Operations())

	// when
	op, backoff, err := svc.Run(operation, logrus.New())
	assert.Zero(t, backoff)
	assert.Equal(t, kymaExpectedNamespace, op.KymaResourceNamespace)
	assert.YAMLEq(t, expectedKymaTemplate, op.KymaTemplate)
	require.NoError(t, err)
}

func TestOverrideKymaModules(t *testing.T) {
	t.Run("default modules are installed when default is true", func(t *testing.T) {
		modules := &internal.ModulesDTO{}
		modules.Default = ptr.Bool(true)
		execTest(t, modules, givenKymaTemplateWithModules, defaultModules)
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})

	t.Run("no modules are installed when default is false", func(t *testing.T) {
		modules := &internal.ModulesDTO{}
		modules.Default = ptr.Bool(false)
		execTest(t, modules, givenKymaTemplateWithModules, "kyma-no-modules.yaml")
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})

	t.Run("custom list of modules are installed when given custom list not empty", func(t *testing.T) {
		modules := &internal.ModulesDTO{}
		modules.List = make([]*internal.ModuleDTO, 0)
		m1 := internal.ModuleDTO{
			Name:                 technicalNameBtpManager,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		m2 := internal.ModuleDTO{
			Name:    technicalNameKeda,
			Channel: internal.Fast,
		}
		modules.List = append(modules.List, &m1, &m2)
		execTest(t, modules, givenKymaTemplateWithModules, defaultModules)
		execTest(t, modules, givenKymaTemplateWithoutModules, defaultModules)
	})

	t.Run("custom list of modules are installed when given custom list not empty", func(t *testing.T) {
		modules := &internal.ModulesDTO{}
		modules.List = make([]*internal.ModuleDTO, 0)
		m1 := internal.ModuleDTO{
			Name:                 technicalNameBtpManager,
			Channel:              internal.Fast,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		modules.List = append(modules.List, &m1)
		execTest(t, modules, givenKymaTemplateWithModules, "kyma-with-btp-manager.yaml")
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-with-btp-manager.yaml")
	})

	t.Run("no modules are installed when given custom list is empty", func(t *testing.T) {
		modules := &internal.ModulesDTO{}
		modules.List = make([]*internal.ModuleDTO, 0)
		execTest(t, modules, givenKymaTemplateWithModules, "kyma-no-modules.yaml")
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})

	t.Run("default modules are installed when modules are empty", func(t *testing.T) {
		execTest(t, &internal.ModulesDTO{}, givenKymaTemplateWithModules, defaultModules)
		execTest(t, &internal.ModulesDTO{}, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})

	t.Run("default modules are installed when modules are not set", func(t *testing.T) {
		execTest(t, nil, givenKymaTemplateWithModules, defaultModules)
		execTest(t, nil, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})
}
