package provisioning

import (
	"testing"

	"github.com/google/uuid"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
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

func execTest(t *testing.T, parameters *pkg.ModulesDTO, in, out string) {
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
		modules := &pkg.ModulesDTO{}
		modules.Default = ptr.Bool(true)
		execTest(t, modules, givenKymaTemplateWithModules, defaultModules)
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})

	t.Run("no modules are installed when default is false", func(t *testing.T) {
		modules := &pkg.ModulesDTO{}
		modules.Default = ptr.Bool(false)
		execTest(t, modules, givenKymaTemplateWithModules, "kyma-no-modules.yaml")
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})

	t.Run("custom list of modules are installed when given custom list is not empty", func(t *testing.T) {
		modules := &pkg.ModulesDTO{}
		modules.List = make([]pkg.ModuleDTO, 0)
		m1 := pkg.ModuleDTO{
			Name:                 technicalNameBtpManager,
			CustomResourcePolicy: internal.Ignore,
			Channel:              internal.Fast,
		}
		m2 := pkg.ModuleDTO{
			Name:                 technicalNameKeda,
			CustomResourcePolicy: internal.CreateAndDelete,
			Channel:              internal.Regular,
		}
		modules.List = append(modules.List, m1, m2)
		execTest(t, modules, givenKymaTemplateWithModules, "kyma-with-keda-and-btp-operator-all-params-set.yaml")
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-with-keda-and-btp-operator-all-params-set.yaml")
	})

	t.Run("custom list of modules are installed when given custom list is not empty", func(t *testing.T) {
		modules := &pkg.ModulesDTO{}
		modules.List = make([]pkg.ModuleDTO, 0)
		m1 := pkg.ModuleDTO{
			Name:                 technicalNameBtpManager,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		m2 := pkg.ModuleDTO{
			Name:    technicalNameKeda,
			Channel: internal.Fast,
		}
		modules.List = append(modules.List, m1, m2)
		execTest(t, modules, givenKymaTemplateWithModules, "kyma-with-keda-and-btp-operator.yaml")
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-with-keda-and-btp-operator.yaml")
	})

	t.Run("custom list of modules are installed when given custom list is not empty and not given channel and crPolicy", func(t *testing.T) {
		modules := &pkg.ModulesDTO{}
		modules.List = make([]pkg.ModuleDTO, 0)
		m1 := pkg.ModuleDTO{
			Name: technicalNameBtpManager,
		}
		m2 := pkg.ModuleDTO{
			Name:    technicalNameKeda,
			Channel: ptr.String(""),
		}
		modules.List = append(modules.List, m1, m2)
		execTest(t, modules, givenKymaTemplateWithModules, "kyma-with-keda-and-btp-operator-only-name.yaml")
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-with-keda-and-btp-operator-only-name.yaml")
	})

	t.Run("custom list of modules are installed when given custom list not empty", func(t *testing.T) {
		modules := &pkg.ModulesDTO{}
		modules.List = make([]pkg.ModuleDTO, 0)
		m1 := pkg.ModuleDTO{
			Name:                 technicalNameBtpManager,
			Channel:              internal.Fast,
			CustomResourcePolicy: internal.CreateAndDelete,
		}
		modules.List = append(modules.List, m1)
		execTest(t, modules, givenKymaTemplateWithModules, "kyma-with-btp-operator.yaml")
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-with-btp-operator.yaml")
	})

	t.Run("no modules are installed when given custom list is empty", func(t *testing.T) {
		modules := &pkg.ModulesDTO{}
		modules.List = make([]pkg.ModuleDTO, 0)
		execTest(t, modules, givenKymaTemplateWithModules, "kyma-no-modules.yaml")
		execTest(t, modules, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})

	t.Run("default modules are installed when modules are empty", func(t *testing.T) {
		execTest(t, &pkg.ModulesDTO{}, givenKymaTemplateWithModules, defaultModules)
		execTest(t, &pkg.ModulesDTO{}, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})

	t.Run("default modules are installed when modules are not set", func(t *testing.T) {
		execTest(t, nil, givenKymaTemplateWithModules, defaultModules)
		execTest(t, nil, givenKymaTemplateWithoutModules, "kyma-no-modules.yaml")
	})
}
