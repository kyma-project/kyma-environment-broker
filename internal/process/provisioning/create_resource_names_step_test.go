package provisioning

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	kymaVersion                   = "1.10"
	k8sVersion                    = "1.16.9"
	shootName                     = "c-1234567"
	instanceID                    = "58f8c703-1756-48ab-9299-a847974d1fee"
	operationID                   = "fd5cee4d-0eeb-40d0-a7a7-0708e5eba470"
	globalAccountID               = "80ac17bd-33e8-4ffa-8d56-1d5367755723"
	subAccountID                  = "12df5747-3efb-4df6-ad6f-4414bb661ce3"
	provisionerOperationID        = "1a0ed09b-9bb9-4e6f-a88c-01955c5f1129"
	runtimeID                     = "2498c8ee-803a-43c2-8194-6d6dd0354c30"
	autoUpdateKubernetesVersion   = true
	autoUpdateMachineImageVersion = true
)

var shootPurpose = "evaluation"

func TestCreateResourceNamesStep_HappyPath(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixProvisioningOperationWithEmptyResourceName()
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	step := NewCreateResourceNamesStep(memoryStorage.Operations())

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	postOperation, backoff, err := step.Run(operation, entry)

	// then
	assert.NoError(t, err)
	assert.Zero(t, backoff)
	assert.Equal(t, operation.RuntimeID, postOperation.RuntimeID)
	assert.Equal(t, postOperation.KymaResourceName, operation.RuntimeID)
	assert.Equal(t, postOperation.KymaResourceNamespace, "kyma-system")
	assert.Equal(t, postOperation.RuntimeResourceName, operation.RuntimeID)
	_, err = memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)

}

func fixProvisioningOperationWithEmptyResourceName() internal.Operation {
	operation := fixture.FixProvisioningOperation(operationID, instanceID)
	operation.KymaResourceName = ""
	operation.RuntimeResourceName = ""
	return operation
}

func TestCreateResourceNamesStep_NoRuntimeID(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixProvisioningOperationWithEmptyResourceName()
	operation.RuntimeID = ""

	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	step := NewCreateResourceNamesStep(memoryStorage.Operations())

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	_, backoff, err := step.Run(operation, entry)

	// then
	assert.ErrorContains(t, err, "RuntimeID not set")
	assert.Zero(t, backoff)
}

func fixOperationCreateRuntime(t *testing.T, planID, region string) internal.Operation {
	return fixOperationCreateRuntimeWithPlatformRegion(t, planID, region, "")
}

func fixOperationCreateRuntimeWithPlatformRegion(t *testing.T, planID, region, platformRegion string) internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperation(operationID, instanceID)
	provisioningOperation.State = domain.InProgress
	provisioningOperation.InputCreator = fixInputCreator(t)
	provisioningOperation.InstanceDetails.ShootName = shootName
	provisioningOperation.InstanceDetails.ShootDNSProviders = gardener.DNSProvidersData{
		Providers: []gardener.DNSProviderData{
			{
				DomainsInclude: []string{"devtest.kyma.ondemand.com"},
				Primary:        true,
				SecretName:     "aws_dns_domain_secrets_test_intest",
				Type:           "route53_type_test",
			},
		},
	}
	provisioningOperation.InstanceDetails.EuAccess = euaccess.IsEURestrictedAccess(platformRegion)
	provisioningOperation.ProvisioningParameters = FixProvisioningParameters(planID, region, platformRegion)
	provisioningOperation.RuntimeID = ""

	return provisioningOperation
}

func fixInstance() internal.Instance {
	instance := fixture.FixInstance(instanceID)
	instance.GlobalAccountID = globalAccountID

	return instance
}

func FixProvisioningParameters(planID, region, platformRegion string) internal.ProvisioningParameters {
	return fixProvisioningParametersWithPlanID(planID, region, platformRegion)
}

func fixProvisioningParametersWithPlanID(planID, region string, platformRegion string) internal.ProvisioningParameters {
	return internal.ProvisioningParameters{
		PlanID:    planID,
		ServiceID: "",
		ErsContext: internal.ERSContext{
			GlobalAccountID: globalAccountID,
			SubAccountID:    subAccountID,
		},
		PlatformRegion: platformRegion,
		Parameters: pkg.ProvisioningParametersDTO{
			Region: ptr.String(region),
			Name:   "dummy",
			Zones:  []string{"europe-west3-b", "europe-west3-c"},
		},
	}
}

func fixInputCreator(t *testing.T) internal.ProvisionerInputCreator {
	cli := fake.NewClientBuilder().WithRuntimeObjects(fixConfigMap(kymaVersion)).Build()
	configProvider := kebConfig.NewConfigProvider(
		kebConfig.NewConfigMapReader(context.TODO(), cli, logrus.New(), "keb-config"),
		kebConfig.NewConfigMapKeysValidator(),
		kebConfig.NewConfigMapConverter())
	ibf, err := input.NewInputBuilderFactory(configProvider, input.Config{
		KubernetesVersion:             k8sVersion,
		DefaultGardenerShootPurpose:   shootPurpose,
		AutoUpdateKubernetesVersion:   autoUpdateKubernetesVersion,
		AutoUpdateMachineImageVersion: autoUpdateMachineImageVersion,
		MultiZoneCluster:              true,
		ControlPlaneFailureTolerance:  "zone",
	}, fixTrialRegionMapping(), fixFreemiumProviders(), fixture.FixOIDCConfigDTO(), false)
	assert.NoError(t, err)

	pp := internal.ProvisioningParameters{
		PlanID:     broker.GCPPlanID,
		Parameters: pkg.ProvisioningParametersDTO{},
	}
	creator, err := ibf.CreateProvisionInput(pp)
	if err != nil {
		t.Errorf("cannot create input creator for %q plan", broker.GCPPlanID)
	}

	return creator
}

func fixTrialRegionMapping() map[string]string {
	return map[string]string{}
}

func fixFreemiumProviders() []string {
	return []string{"azure", "aws"}
}

func fixConfigMap(defaultKymaVersion string) k8sruntime.Object {
	kebCfg := &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "keb-config",
			Namespace: "kcp-system",
			Labels: map[string]string{
				"keb-config": "true",
				fmt.Sprintf("runtime-version-%s", defaultKymaVersion): "true",
			},
		},
		Data: map[string]string{
			"default": `kyma-template: "---"`},
	}
	return kebCfg
}
