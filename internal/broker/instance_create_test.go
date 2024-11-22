package broker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"

	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/broker/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	kcMock "github.com/kyma-project/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v8/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	serviceID       = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
	planID          = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	clusterRegion   = "westeurope"
	globalAccountID = "e8f7ec0a-0cd6-41f0-905d-5d1efa9fb6c4"
	subAccountID    = "3cb65e5b-e455-4799-bf35-be46e8f5a533"
	userID          = "test@test.pl"

	instanceID           = "d3d5dca4-5dc8-44ee-a825-755c2a3fb839"
	otherInstanceID      = "87bfaeaa-48eb-40d6-84f3-3d5368eed3eb"
	existOperationID     = "920cbfd9-24e9-4aa2-aa77-879e9aabe140"
	clusterName          = "cluster-testing"
	region               = "eu"
	brokerURL            = "example.com"
	notEncodedKubeconfig = "apiVersion: v1\\nkind: Config"
	encodedKubeconfig    = "YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCmN1cnJlbnQtY29udGV4dDogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKY29udGV4dHM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY29udGV4dDoKICAgICAgY2x1c3Rlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUKICAgICAgdXNlcjogc2hvb3QtLWt5bWEtZGV2LS1jbHVzdGVyLW5hbWUtdG9rZW4KY2x1c3RlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZQogICAgY2x1c3RlcjoKICAgICAgc2VydmVyOiBodHRwczovL2FwaS5jbHVzdGVyLW5hbWUua3ltYS1kZXYuc2hvb3QuY2FuYXJ5Lms4cy1oYW5hLm9uZGVtYW5kLmNvbQogICAgICBjZXJ0aWZpY2F0ZS1hdXRob3JpdHktZGF0YTogPi0KICAgICAgICBMUzB0TFMxQ1JVZEpUaUJEUlZKVVNVWkpRMEZVUlMwdExTMHQKdXNlcnM6CiAgLSBuYW1lOiBzaG9vdC0ta3ltYS1kZXYtLWNsdXN0ZXItbmFtZS10b2tlbgogICAgdXNlcjoKICAgICAgdG9rZW46ID4tCiAgICAgICAgdE9rRW4K"
	shootName            = "own-cluster-name"
	shootDomain          = "kyma-dev.shoot.canary.k8s-hana.ondemand.com"
)

var dashboardConfig = dashboard.Config{LandscapeURL: "https://dashboard.example.com"}

func TestProvision_Provision(t *testing.T) {
	t.Run("new operation will be created", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
		assert.Regexp(t, "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$", response.OperationData)
		assert.NotEqual(t, instanceID, response.OperationData)
		assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, response.DashboardURL)
		assert.Equal(t, clusterName, response.Metadata.Labels["Name"])
		assert.Equal(t, fmt.Sprintf("https://%s/kubeconfig/%s", brokerURL, instanceID), response.Metadata.Labels["KubeconfigURL"])
		assert.NotContains(t, response.Metadata.Labels, "APIServerURL")

		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, operation.InstanceID, instanceID)

		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
		assert.Equal(t, clusterName, operation.ProvisioningParameters.Parameters.Name)
		assert.Equal(t, userID, operation.ProvisioningParameters.ErsContext.UserID)
		assert.Equal(t, "req-region", operation.ProvisioningParameters.PlatformRegion)

		assert.Equal(t, fixDNSProviders(), operation.ShootDNSProviders)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		assert.Equal(t, instance.Parameters, operation.ProvisioningParameters)
		assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, response.DashboardURL)
		assert.Equal(t, instance.GlobalAccountID, globalAccountID)
		assert.Equal(t, fixDNSProviders(), instance.InstanceDetails.ShootDNSProviders)
	})

	t.Run("new operation for own_cluster plan with kubeconfig will be created", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.OwnClusterPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure", "own_cluster"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.OwnClusterPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "kubeconfig": "%s", "shootName":"%s", "shootDomain":"%s"}`, clusterName, encodedKubeconfig, shootName, shootDomain)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
		// UUID with version 4 and variant 1 i.e RFC. 4122/DCE
		assert.Regexp(t, "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$", response.OperationData)
		assert.NotEqual(t, instanceID, response.OperationData)
		assert.Equal(t, `https://dashboard.example.com`, response.DashboardURL)
		assert.Equal(t, clusterName, response.Metadata.Labels["Name"])
		assert.NotContains(t, response.Metadata.Labels, "KubeconfigURL")
		assert.NotContains(t, response.Metadata.Labels, "APIServerURL")

		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, operation.InstanceID, instanceID)

		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
		assert.Equal(t, clusterName, operation.ProvisioningParameters.Parameters.Name)
		assert.Equal(t, userID, operation.ProvisioningParameters.ErsContext.UserID)
		assert.Equal(t, "req-region", operation.ProvisioningParameters.PlatformRegion)

		require.NoError(t, err)

		assert.Equal(t, fixDNSProviders(), operation.ShootDNSProviders)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		assert.Equal(t, instance.Parameters, operation.ProvisioningParameters)
		assert.Equal(t, `https://dashboard.example.com`, response.DashboardURL)
		assert.Equal(t, instance.GlobalAccountID, globalAccountID)
		assert.Equal(t, fixDNSProviders(), instance.InstanceDetails.ShootDNSProviders)
		assert.Equal(t, shootDomain, operation.ShootDomain)
	})

	t.Run("new operation for own_cluster plan with not encoded kubeconfig will not be created", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.OwnClusterPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure", "own_cluster"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.OwnClusterPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "kubeconfig": "%s", "shootName":"%s", "shootDomain":"%s"}`, clusterName, notEncodedKubeconfig, shootName, shootDomain)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.ErrorContains(t, err, "while decoding kubeconfig")
	})

	t.Run("new operation for own_cluster plan will not be created without required fields", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.OwnClusterPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure", "own_cluster"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when shootDomain is missing
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.OwnClusterPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "kubeconfig": "%s", "shootName":"%s"}`, clusterName, encodedKubeconfig, shootName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.ErrorContains(t, err, "while validating input parameters: (root): shootDomain is required")

		// when shootName is missing
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.OwnClusterPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "kubeconfig": "%s", "shootDomain":"%s"}`, clusterName, encodedKubeconfig, shootDomain)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.ErrorContains(t, err, "while validating input parameters: (root): shootName is required")

		// when shootDomain is missing
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.OwnClusterPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "shootDomain": "%s", "shootName":"%s"}`, clusterName, shootDomain, shootName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.ErrorContains(t, err, "while validating input parameters: (root): kubeconfig is required")
	})

	t.Run("for plan other than own_cluster invalid kubeconfig will be ignored", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "kubeconfig": "%s", "shootName":"%s", "shootDomain":"%s"}`, clusterName, clusterRegion, notEncodedKubeconfig, shootName, shootDomain)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
		// UUID with version 4 and variant 1 i.e RFC. 4122/DCE
		assert.Regexp(t, "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$", response.OperationData)
		assert.NotEqual(t, instanceID, response.OperationData)
		assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, response.DashboardURL)
		assert.Equal(t, clusterName, response.Metadata.Labels["Name"])
		assert.Equal(t, fmt.Sprintf("https://%s/kubeconfig/%s", brokerURL, instanceID), response.Metadata.Labels["KubeconfigURL"])
		assert.NotContains(t, response.Metadata.Labels, "APIServerURL")

		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, operation.InstanceID, instanceID)

		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
		assert.Equal(t, clusterName, operation.ProvisioningParameters.Parameters.Name)
		assert.Equal(t, userID, operation.ProvisioningParameters.ErsContext.UserID)
		assert.Equal(t, "req-region", operation.ProvisioningParameters.PlatformRegion)

		require.NoError(t, err)

		assert.Equal(t, fixDNSProviders(), operation.ShootDNSProviders)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		assert.Equal(t, instance.Parameters, operation.ProvisioningParameters)
		assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, response.DashboardURL)
		assert.Equal(t, instance.GlobalAccountID, globalAccountID)
		assert.Equal(t, fixDNSProviders(), instance.InstanceDetails.ShootDNSProviders)
	})

	t.Run("existing operation ID will be return", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertOperation(fixExistOperation())
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(fixInstance())

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", mock.AnythingOfType("string")).Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure", "azure_lite"},
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			nil,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, region), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)

		// then
		require.NoError(t, err)
		assert.Equal(t, existOperationID, response.OperationData)
		assert.Len(t, response.Metadata.Labels, 2)
	})

	t.Run("more than one trial is not allowed", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertOperation(fixExistOperation())
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID:      instanceID,
			GlobalAccountID: globalAccountID,
			ServiceID:       serviceID,
			ServicePlanID:   broker.TrialPlanID,
		})
		assert.NoError(t, err)

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.TrialPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.TrialPlanName}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			nil,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "dummy"), "new-instance-id", domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.TrialPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)

		// then
		assert.EqualError(t, err, "trial Kyma was created for the global account, but there is only one allowed")
	})

	t.Run("more than one trial is allowed", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertOperation(fixExistOperation())
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(internal.Instance{
			InstanceID:      instanceID,
			GlobalAccountID: globalAccountID,
			ServiceID:       serviceID,
			ServicePlanID:   broker.TrialPlanID,
		})
		assert.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.TrialPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.TrialPlanName}, OnlySingleTrialPerGA: false},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), otherInstanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.TrialPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)

		// then
		require.NoError(t, err)
		assert.Regexp(t, "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$", response.OperationData)
		assert.NotEqual(t, instanceID, response.OperationData)

		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, operation.InstanceID, otherInstanceID)

		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
		assert.Equal(t, clusterName, operation.ProvisioningParameters.Parameters.Name)
		assert.Equal(t, "req-region", operation.ProvisioningParameters.PlatformRegion)

		instance, err := memoryStorage.Instances().GetByID(otherInstanceID)
		require.NoError(t, err)

		assert.Equal(t, instance.Parameters, operation.ProvisioningParameters)
		assert.Equal(t, instance.GlobalAccountID, globalAccountID)
	})

	t.Run("provision trial", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Instances().Insert(internal.Instance{
			InstanceID:      instanceID,
			GlobalAccountID: "other-global-account",
			ServiceID:       serviceID,
			ServicePlanID:   broker.TrialPlanID,
		})
		require.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.TrialPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "trial"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.TrialPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)

		// then
		require.NoError(t, err)
		assert.Regexp(t, "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$", response.OperationData)
		assert.NotEqual(t, instanceID, response.OperationData)

		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, operation.InstanceID, instanceID)

		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)
		assert.Equal(t, clusterName, operation.ProvisioningParameters.Parameters.Name)
		assert.Equal(t, "req-region", operation.ProvisioningParameters.PlatformRegion)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)

		assert.Equal(t, instance.Parameters, operation.ProvisioningParameters)
		assert.Equal(t, instance.GlobalAccountID, globalAccountID)
	})

	t.Run("fail if trial with invalid region", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Instances().Insert(internal.Instance{
			InstanceID:      instanceID,
			GlobalAccountID: "other-global-account",
			ServiceID:       serviceID,
			ServicePlanID:   broker.TrialPlanID,
		})
		require.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.TrialPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "trial"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.TrialPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region":"invalid-region"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)

		// then
		require.ErrorContains(t, err, "invalid region specified in request for trial")
	})

	t.Run("conflict should be handled", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.Operations().InsertOperation(fixExistOperation())
		assert.NoError(t, err)
		err = memoryStorage.Instances().Insert(fixInstance())
		assert.NoError(t, err)

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			nil,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID, userID)),
		}, true)

		// then
		assert.EqualError(t, err, "provisioning operation already exist")
		assert.Empty(t, response.OperationData)
	})

	t.Run("should return error when region is not specified", func(t *testing.T) {
		// given
		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			nil,
			nil,
			nil,
			nil,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		_, provisionErr := provisionEndpoint.Provision(context.Background(), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID, userID)),
		}, true)

		// then
		require.EqualError(t, provisionErr, "No region specified in request.")
	})

	t.Run("licence type lite should be saved in parameters", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.AzureLitePlanID).Return(true)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.AzureLitePlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID, userID)),
		}, true)
		assert.NoError(t, err)

		// then
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)

		assert.Equal(t, ptr.String(internal.LicenceTypeLite), operation.ProvisioningParameters.Parameters.LicenceType)
	})

	t.Run("licence type lite should be saved in parameters for Trial Plan", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.TrialPlanID).Return(true)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", "trial"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.TrialPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, clusterName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "1cafb9c8-c8f8-478a-948a-9cb53bb76aa4", subAccountID, userID)),
		}, true)
		assert.NoError(t, err)

		// then
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)

		assert.Equal(t, ptr.String(internal.LicenceTypeLite), operation.ProvisioningParameters.Parameters.LicenceType)
	})

	t.Run("Should fail on insufficient OIDC params (missing issuerURL)", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		oidcParams := `"clientID":"client-id"`
		err := fmt.Errorf("issuerURL must not be empty")
		errMsg := fmt.Sprintf("[instanceID: %s] %s", instanceID, err)
		expectedErr := apiresponses.NewFailureResponse(err, http.StatusBadRequest, errMsg)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, clusterRegion, oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on insufficient OIDC params (missing clientID)", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		oidcParams := `"issuerURL":"https://test.local"`
		err := fmt.Errorf("clientID must not be empty")
		errMsg := fmt.Sprintf("[instanceID: %s] %s", instanceID, err)
		expectedErr := apiresponses.NewFailureResponse(err, http.StatusBadRequest, errMsg)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, clusterRegion, oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on invalid OIDC signingAlgs param", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256","notValid"]`
		err := fmt.Errorf("signingAlgs must contain valid signing algorithm(s)")
		errMsg := fmt.Sprintf("[instanceID: %s] %s", instanceID, err)
		expectedErr := apiresponses.NewFailureResponse(err, http.StatusBadRequest, errMsg)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, clusterRegion, oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should pass for any globalAccountId - EU Access", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu11"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, "switzerlandnorth", oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
	})

	t.Run("first freemium is allowed", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.FreemiumPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.FreemiumPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)
		assert.NoError(t, err)

		// then
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, instanceID, operation.InstanceID)
		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, broker.FreemiumPlanID, instance.ServicePlanID)
	})

	t.Run("freemium is allowed if provisioning failed", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.InstancesArchived().Insert(internal.InstanceArchived{
			InstanceID:        instanceID,
			GlobalAccountID:   globalAccountID,
			PlanID:            broker.FreemiumPlanID,
			ProvisioningState: domain.Failed,
		})
		assert.NoError(t, err)
		ins := fixInstance()
		ins.InstanceID = instID
		ins.ServicePlanID = broker.FreemiumPlanID
		err = memoryStorage.Instances().Insert(ins)
		assert.NoError(t, err)
		op := fixOperation()
		op.State = domain.Failed
		op.ProvisioningParameters.PlanID = broker.FreemiumPlanID
		err = memoryStorage.Operations().InsertOperation(op)
		assert.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.FreemiumPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.FreemiumPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)
		assert.NoError(t, err)

		// then
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, instanceID, operation.InstanceID)
		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, broker.FreemiumPlanID, instance.ServicePlanID)
	})

	t.Run("more than one freemium allowed for whitelisted global account", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.InstancesArchived().Insert(internal.InstanceArchived{
			InstanceID:        instanceID,
			GlobalAccountID:   globalAccountID,
			PlanID:            broker.FreemiumPlanID,
			ProvisioningState: domain.Succeeded,
		})
		assert.NoError(t, err)

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.FreemiumPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{globalAccountID: struct{}{}},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.FreemiumPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)

		// then
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, instanceID, operation.InstanceID)
		assert.Equal(t, globalAccountID, operation.ProvisioningParameters.ErsContext.GlobalAccountID)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, broker.FreemiumPlanID, instance.ServicePlanID)
	})

	t.Run("more than one freemium in instances is not allowed", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		ins := fixInstance()
		ins.InstanceID = instID
		ins.ServicePlanID = broker.FreemiumPlanID
		err := memoryStorage.Instances().Insert(ins)
		assert.NoError(t, err)
		op := fixOperation()
		op.ProvisioningParameters.PlanID = broker.FreemiumPlanID
		err = memoryStorage.Operations().InsertOperation(op)
		assert.NoError(t, err)

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.FreemiumPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			nil,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.FreemiumPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)

		// then
		assert.EqualError(t, err, "provisioning request rejected, you have already used the available free service plan quota in this global account")
	})

	t.Run("more than one freemium in instances archive is not allowed", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()
		err := memoryStorage.InstancesArchived().Insert(internal.InstanceArchived{
			InstanceID:        instanceID,
			GlobalAccountID:   globalAccountID,
			PlanID:            broker.FreemiumPlanID,
			ProvisioningState: domain.Succeeded,
		})
		assert.NoError(t, err)

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.FreemiumPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			nil,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "dummy"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.FreemiumPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
		}, true)

		// then
		assert.EqualError(t, err, "provisioning request rejected, you have already used the available free service plan quota in this global account")
	})

	t.Run("Should pass for assured workloads in me-central2 region", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.GCPPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-sa30"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.GCPPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, "me-central2", oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
	})

	t.Run("Should fail for assured workloads in us-central1 region", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.GCPPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp"},
				URL:                      brokerURL,
				OnlySingleTrialPerGA:     true,
				EnableKubeconfigURLLabel: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-sa30"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.GCPPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, "us-central1", oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.EqualError(t, err, "while validating input parameters: region: region must be one of the following: \"me-central2\"")
	})
}

func TestNetworkingValidation(t *testing.T) {
	for tn, tc := range map[string]struct {
		givenNetworking string

		expectedError bool
	}{
		"Invalid nodes CIDR": {
			givenNetworking: `{"nodes": 1abcd"}`,
			expectedError:   true,
		},
		"Invalid nodes CIDR - wrong IP range": {
			givenNetworking: `{"nodes": "10.250.0.1/22"}`,
			expectedError:   true,
		},
		"Valid CIDRs": {
			givenNetworking: `{"nodes": "10.250.0.0/20"}`,
			expectedError:   false,
		},
		"Overlaps with seed cidr": {
			givenNetworking: `{"nodes": "10.243.128.0/18"}`,
			expectedError:   true,
		},
		/*"Invalid pods CIDR": {
		  	givenNetworking: `{"nodes": "10.250.0.0/16", "pods": "10abcd/16", "services": "100.104.0.0/13"}`,
		  	expectedError:   true,
		  },
		  "Invalid pods CIDR - wrong IP range": {
		  	givenNetworking: `{"nodes": "10.250.0.0/16", "pods": "10.250.0.1/19", "services": "100.104.0.0/13"}`,
		  	expectedError:   true,
		  },
		  "Invalid services CIDR": {
		  	givenNetworking: `{"nodes": "10.250.0.0/16", "pods": "10.250.0.1/19", "services": "abcd"}`,
		  	expectedError:   true,
		  },
		  "Invalid services CIDR - wrong IP range": {
		  	givenNetworking: `{"nodes": "10.250.0.0/16", "pods": "10.250.0.1/19", "services": "10.250.0.1/19"}`,
		  	expectedError:   true,
		  },
		  "Pods and Services overlaps": {
		  	givenNetworking: `{"nodes": "10.250.0.0/22", "pods": "10.64.0.0/19", "services": "10.64.0.0/16"}`,
		  	expectedError:   true,
		  },*/
		"Pods and Nodes overlaps": {
			givenNetworking: `{"nodes": "10.96.0.0/16"}`,
			expectedError:   true,
		},
		"Services and Nodes overlaps": {
			givenNetworking: `{"nodes": "10.104.0.0/13"}`,
			expectedError:   true,
		},
		"Suffix too big": {
			givenNetworking: `{"nodes": "10.250.0.0/25"}`,
			expectedError:   true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			// #setup memory storage
			memoryStorage := storage.NewMemoryStorage()

			queue := &automock.Queue{}
			queue.On("Add", mock.AnythingOfType("string"))

			factoryBuilder := &automock.PlanValidator{}
			factoryBuilder.On("IsPlanSupport", mock.AnythingOfType("string")).Return(true)

			planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
				return &gqlschema.ClusterConfigInput{}, nil
			}
			kcBuilder := &kcMock.KcBuilder{}
			// #create provisioner endpoint
			provisionEndpoint := broker.NewProvision(
				broker.Config{EnablePlans: []string{"gcp", "azure", "free"}},
				gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
				memoryStorage.Operations(),
				memoryStorage.Instances(),
				memoryStorage.InstancesArchived(),
				queue,
				factoryBuilder,
				broker.PlansConfig{},
				planDefaults,
				logrus.StandardLogger(),
				dashboardConfig,
				kcBuilder,
				whitelist.Set{},
				&broker.OneForAllConvergedCloudRegionsProvider{},
			)

			// when
			_, err := provisionEndpoint.Provision(fixRequestContextWithProvider(t, "cf-eu10", "azure"), instanceID,
				domain.ProvisionDetails{
					ServiceID:     serviceID,
					PlanID:        broker.AzurePlanID,
					RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "cluster-name", "region": "%s", "networking": %s}`, clusterRegion, tc.givenNetworking)),
					RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
				}, true)

			// then
			assert.Equal(t, tc.expectedError, err != nil)
		})
	}

}

func TestRegionValidation(t *testing.T) {

	for tn, tc := range map[string]struct {
		planID           string
		parameters       string
		platformProvider pkg.CloudProvider

		expectedErrorStatusCode int
		expectedError           bool
	}{
		"invalid region": {
			planID:           broker.AzurePlanID,
			platformProvider: pkg.Azure,
			parameters:       `{"name": "cluster-name", "region":"not-valid"}`,

			expectedErrorStatusCode: http.StatusBadRequest,
			expectedError:           true,
		},
		"Azure region for AWS freemium": {
			planID:           broker.FreemiumPlanID,
			platformProvider: pkg.AWS,
			parameters:       `{"name": "cluster-name", "region": "eastus"}`,

			expectedErrorStatusCode: http.StatusBadRequest,
			expectedError:           true,
		},
		"Azure region for Azure freemium": {
			planID:           broker.FreemiumPlanID,
			platformProvider: pkg.Azure,
			parameters:       `{"name": "cluster-name", "region": "eastus"}`,

			expectedError: false,
		},
		"AWS region for AWS freemium": {
			planID:           broker.FreemiumPlanID,
			platformProvider: pkg.AWS,
			parameters:       `{"name": "cluster-name", "region": "eu-central-1"}`,

			expectedError: false,
		},
		"AWS region for Azure freemium": {
			planID:           broker.FreemiumPlanID,
			platformProvider: pkg.Azure,
			parameters:       `{"name": "cluster-name", "region": "eu-central-1"}`,

			expectedErrorStatusCode: http.StatusBadRequest,
			expectedError:           true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			// #setup memory storage
			memoryStorage := storage.NewMemoryStorage()

			queue := &automock.Queue{}
			queue.On("Add", mock.AnythingOfType("string"))

			factoryBuilder := &automock.PlanValidator{}
			factoryBuilder.On("IsPlanSupport", tc.planID).Return(true)

			planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
				return &gqlschema.ClusterConfigInput{}, nil
			}
			kcBuilder := &kcMock.KcBuilder{}
			// #create provisioner endpoint
			provisionEndpoint := broker.NewProvision(
				broker.Config{EnablePlans: []string{"gcp", "azure", "free"}, OnlySingleTrialPerGA: true},
				gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
				memoryStorage.Operations(),
				memoryStorage.Instances(),
				memoryStorage.InstancesArchived(),
				queue,
				factoryBuilder,
				broker.PlansConfig{},
				planDefaults,
				logrus.StandardLogger(),
				dashboardConfig,
				kcBuilder,
				whitelist.Set{},
				&broker.OneForAllConvergedCloudRegionsProvider{},
			)

			// when
			_, err := provisionEndpoint.Provision(fixRequestContextWithProvider(t, "cf-eu10", tc.platformProvider), instanceID,
				domain.ProvisionDetails{
					ServiceID:     serviceID,
					PlanID:        tc.planID,
					RawParameters: json.RawMessage(tc.parameters),
					RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
				}, true)

			// then
			if tc.expectedError {
				require.Error(t, err)
				assert.Equal(t, tc.expectedErrorStatusCode, err.(*apiresponses.FailureResponse).ValidatedStatusCode(nil))
			} else {
				assert.NoError(t, err)
			}

		})
	}

}

func TestSapConvergedCloudBlocking(t *testing.T) {
	t.Run("Should succeed if converged cloud is enabled and converged plan selected", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.SapConvergedCloudPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{broker.SapConvergedCloudPlanName},
				URL:                      brokerURL,
				DisableSapConvergedCloud: false,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "eu-de-1"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.SapConvergedCloudPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, "eu-de-1", oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
	})

	t.Run("Should succeed if converged cloud is disabled and converged plan not selected", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"gcp", "azure"},
				URL:                      brokerURL,
				DisableSapConvergedCloud: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu11"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, "switzerlandnorth", oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
	})

	t.Run("Should fail if converged cloud is disabled and converged plan selected", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.SapConvergedCloudPlanID).Return(true)

		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:              []string{"sap-converged-cloud"},
				URL:                      brokerURL,
				DisableSapConvergedCloud: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			memoryStorage.Operations(),
			memoryStorage.Instances(),
			memoryStorage.InstancesArchived(),
			queue,
			factoryBuilder,
			broker.PlansConfig{},
			planDefaults,
			logrus.StandardLogger(),
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			&broker.OneForAllConvergedCloudRegionsProvider{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`
		err := fmt.Errorf(broker.CONVERGED_CLOUD_BLOCKED_MSG)
		errMsg := broker.CONVERGED_CLOUD_BLOCKED_MSG
		expectedErr := apiresponses.NewFailureResponse(err, http.StatusBadRequest, errMsg)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "eu-de-1"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.SapConvergedCloudPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, "eu-de-1", oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.LoggerAction(), apierr.LoggerAction())
	})
}

func fixExistOperation() internal.Operation {
	provisioningOperation := fixture.FixProvisioningOperation(existOperationID, instanceID)
	ptrClusterRegion := clusterRegion
	provisioningOperation.ProvisioningParameters = internal.ProvisioningParameters{
		PlanID:    planID,
		ServiceID: serviceID,
		ErsContext: internal.ERSContext{
			SubAccountID:    subAccountID,
			GlobalAccountID: globalAccountID,
			UserID:          userID,
		},
		Parameters: pkg.ProvisioningParametersDTO{
			Name:   clusterName,
			Region: &ptrClusterRegion,
		},
		PlatformRegion: region,
	}

	return provisioningOperation
}

func fixInstance() internal.Instance {
	return fixture.FixInstance(instanceID)
}

func fixRequestContext(t *testing.T, region string) context.Context {
	t.Helper()
	return fixRequestContextWithProvider(t, region, pkg.Azure)
}

func fixRequestContextWithProvider(t *testing.T, region string, provider pkg.CloudProvider) context.Context {
	t.Helper()

	ctx := context.TODO()
	ctx = middleware.AddRegionToCtx(ctx, region)
	ctx = middleware.AddProviderToCtx(ctx, provider)
	return ctx
}

func fixDNSProviders() gardener.DNSProvidersData {
	return gardener.DNSProvidersData{
		Providers: []gardener.DNSProviderData{
			{
				DomainsInclude: []string{"dev.example.com"},
				Primary:        true,
				SecretName:     "aws_dns_domain_secrets_test_instance",
				Type:           "route53_type_test",
			},
		},
	}
}
