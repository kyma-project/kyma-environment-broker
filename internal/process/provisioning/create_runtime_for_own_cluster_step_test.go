package provisioning

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal"
	kebConfig "github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/euaccess"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/process/input"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
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

func TestCreateRuntimeForOwnCluster_Run(t *testing.T) {
	// given
	log := logrus.New()
	memoryStorage := storage.NewMemoryStorage()

	operation := fixOperationCreateRuntime(t, broker.OwnClusterPlanID, "europe-west3")
	operation.ShootDomain = "kyma.org"
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)

	err = memoryStorage.Instances().Insert(fixInstance())
	assert.NoError(t, err)

	step := NewCreateRuntimeForOwnClusterStep(memoryStorage.Operations(), memoryStorage.Instances())

	// when
	entry := log.WithFields(logrus.Fields{"step": "TEST"})
	operation, _, err = step.Run(operation, entry)

	// then

	storedInstance, err := memoryStorage.Instances().GetByID(operation.InstanceID)
	assert.NoError(t, err)
	assert.NotEmpty(t, storedInstance.RuntimeID)

	storedOperation, err := memoryStorage.Operations().GetOperationByID(operationID)
	assert.NoError(t, err)
	assert.Empty(t, storedOperation.ProvisionerOperationID)
	assert.Equal(t, storedInstance.RuntimeID, storedOperation.RuntimeID)
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
		Parameters: internal.ProvisioningParametersDTO{
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
	}, kymaVersion, fixTrialRegionMapping(), fixFreemiumProviders(), fixture.FixOIDCConfigDTO())
	assert.NoError(t, err)

	pp := internal.ProvisioningParameters{
		PlanID:     broker.GCPPlanID,
		Parameters: internal.ProvisioningParametersDTO{},
	}
	version := internal.RuntimeVersionData{Version: kymaVersion, Origin: internal.Parameters}
	creator, err := ibf.CreateProvisionInput(pp, version)
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
			"default": `additional-components:
  - name: "additional-component1"
    namespace: "kyma-system"`,
		},
	}

	return kebCfg
}
