package broker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/additionalproperties"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/broker/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	kcMock "github.com/kyma-project/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/middleware"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"

	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/sets"
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

func TestProvision_Provision(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	t.Run("new operation will be created", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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
		assert.Equal(t, pkg.Azure, instance.Provider)
	})

	t.Run("new operation for own_cluster plan with kubeconfig will be created", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.OwnClusterPlanID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure", "own_cluster"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure", "own_cluster"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure", "own_cluster"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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
		assert.ErrorContains(t, err, "while validating input parameters: at '': missing property 'shootDomain'")

		// when shootName is missing
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.OwnClusterPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "kubeconfig": "%s", "shootDomain":"%s"}`, clusterName, encodedKubeconfig, shootDomain)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.ErrorContains(t, err, "while validating input parameters: at '': missing property 'shootName'")

		// when shootDomain is missing
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.OwnClusterPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "shootDomain": "%s", "shootName":"%s"}`, clusterName, shootDomain, shootName)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.ErrorContains(t, err, "while validating input parameters: at '': missing property 'kubeconfig'")
	})

	t.Run("for plan other than own_cluster invalid kubeconfig will be ignored", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", mock.AnythingOfType("string")).Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure", "azure_lite"},
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			nil,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.TrialPlanName}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			nil,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.TrialPlanName}, OnlySingleTrialPerGA: false},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "trial"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "trial"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			nil,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		memoryStorage := storage.NewMemoryStorage()

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite"}, OnlySingleTrialPerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			nil,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

	t.Run("Should fail on insufficient OIDC params (missing clientID)", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on invalid OIDC signingAlgs param", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should accept an valid OIDC object type", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"],"requiredClaims":["claim=value"]`

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, clusterRegion, oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, "client-id", operation.ProvisioningParameters.Parameters.OIDC.OIDCConfigDTO.ClientID)
		assert.Equal(t, "https://test.local", operation.ProvisioningParameters.Parameters.OIDC.OIDCConfigDTO.IssuerURL)
		assert.Equal(t, []string{"RS256"}, operation.ProvisioningParameters.Parameters.OIDC.OIDCConfigDTO.SigningAlgs)
		assert.Equal(t, []string{"claim=value"}, operation.ProvisioningParameters.Parameters.OIDC.OIDCConfigDTO.RequiredClaims)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, operation.ProvisioningParameters.Parameters.OIDC.OIDCConfigDTO, instance.Parameters.Parameters.OIDC.OIDCConfigDTO)
	})

	t.Run("Should accept an valid OIDC list type", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"],"groupsPrefix":"-", "usernameClaim":"-", "usernamePrefix":"-", "requiredClaims":["claim=value"], "groupsClaim":"-"`

		// when
		response, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{"list":[{%s}]}}`, clusterName, clusterRegion, oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
		operation, err := memoryStorage.Operations().GetProvisioningOperationByID(response.OperationData)
		require.NoError(t, err)
		assert.Equal(t, "client-id", operation.ProvisioningParameters.Parameters.OIDC.List[0].ClientID)
		assert.Equal(t, "https://test.local", operation.ProvisioningParameters.Parameters.OIDC.List[0].IssuerURL)
		assert.Equal(t, []string{"RS256"}, operation.ProvisioningParameters.Parameters.OIDC.List[0].SigningAlgs)
		assert.Equal(t, []string{"claim=value"}, operation.ProvisioningParameters.Parameters.OIDC.List[0].RequiredClaims)

		instance, err := memoryStorage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, operation.ProvisioningParameters.Parameters.OIDC.List[0], instance.Parameters.Parameters.OIDC.List[0])
	})

	t.Run("Should fail on invalid OIDC issuerURL params", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		kcBuilder := &kcMock.KcBuilder{}
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		testCases := []struct {
			name          string
			oidcParams    string
			expectedError string
		}{
			{
				name:          "wrong scheme",
				oidcParams:    `"clientID":"client-id","issuerURL":"http://test.local","signingAlgs":["RS256"]`,
				expectedError: "issuerURL must have https scheme",
			},
			{
				name:          "missing issuerURL",
				oidcParams:    `"clientID":"client-id"`,
				expectedError: "issuerURL must not be empty",
			},
			{
				name:          "URL with fragment",
				oidcParams:    `"clientID":"client-id","issuerURL":"https://test.local#fragment","signingAlgs":["RS256"]`,
				expectedError: "issuerURL must not contain a fragment",
			},
			{
				name:          "URL with username",
				oidcParams:    `"clientID":"client-id","issuerURL":"https://user@test.local","signingAlgs":["RS256"]`,
				expectedError: "issuerURL must not contain a username or password",
			},
			{
				name:          "URL with query",
				oidcParams:    `"clientID":"client-id","issuerURL":"https://test.local?query=param","signingAlgs":["RS256"]`,
				expectedError: "issuerURL must not contain a query",
			},
			{
				name:          "URL with empty host",
				oidcParams:    `"clientID":"client-id","issuerURL":"https:///path","signingAlgs":["RS256"]`,
				expectedError: "issuerURL must be a valid URL, issuerURL must have https scheme",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := fmt.Errorf("%s", tc.expectedError)
				errMsg := fmt.Sprintf("[instanceID: %s] %s", instanceID, err)
				expectedErr := apiresponses.NewFailureResponse(err, http.StatusBadRequest, errMsg)

				// when
				_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
					ServiceID:     serviceID,
					PlanID:        planID,
					RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, clusterRegion, tc.oidcParams)),
					RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
				}, true)
				t.Logf("%+v\n", *provisionEndpoint)

				// then
				require.Error(t, err)
				assert.IsType(t, &apiresponses.FailureResponse{}, err)
				apierr := err.(*apiresponses.FailureResponse)
				assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
				assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).LoggerAction(), apierr.LoggerAction())
			})
		}
	})

	t.Run("Should fail on invalid OIDC requiredClaims params", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		testCases := []struct {
			name          string
			oidcParams    string
			expectedError string
		}{
			{
				name:       "only a string",
				oidcParams: `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"],"requiredClaims":["value"]`,
			},
			{
				name:       "only claim=",
				oidcParams: `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"],"requiredClaims":["claim="]`,
			},
			{
				name:       "only =value",
				oidcParams: `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"],"requiredClaims":["=value"]`,
			},
			{
				name:       "only =",
				oidcParams: `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"],"requiredClaims":["="]`,
			},
			{
				name:       "valid and invalid claim",
				oidcParams: `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"],"requiredClaims":["claim=value","claim="]`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := errors.New(tc.expectedError)
				errMsg := fmt.Sprintf("[instanceID: %s] %s", instanceID, err)
				expectedErr := apiresponses.NewFailureResponse(err, http.StatusBadRequest, errMsg)

				// when
				_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
					ServiceID:     serviceID,
					PlanID:        planID,
					RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, clusterRegion, tc.oidcParams)),
					RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
				}, true)
				t.Logf("%+v\n", *provisionEndpoint)

				// then
				require.Error(t, err)
				assert.IsType(t, &apiresponses.FailureResponse{}, err)
				apierr := err.(*apiresponses.FailureResponse)
				assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
			})
		}
	})

	t.Run("Should pass for any globalAccountId - EU Access", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", planID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-ch20"), instanceID, domain.ProvisionDetails{
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

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{globalAccountID: struct{}{}},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			nil,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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
		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{EnablePlans: []string{"gcp", "azure", "azure_lite", broker.FreemiumPlanName}, OnlyOneFreePerGA: true},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			nil,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
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
		require.EqualError(t, err, "while validating input parameters: at '/region': value must be 'me-central2'")
	})
}

func TestAdditionalWorkerNodePools(t *testing.T) {
	for tn, tc := range map[string]struct {
		additionalWorkerNodePools string
		expectedError             bool
	}{
		"Valid additional worker node pools": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "m6i.large", "haZones": false, "autoScalerMin": 1, "autoScalerMax": 20}]`,
			expectedError:             false,
		},
		"Empty additional worker node pools": {
			additionalWorkerNodePools: `[]`,
			expectedError:             false,
		},
		"Empty name": {
			additionalWorkerNodePools: `[{"name": "", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Missing name": {
			additionalWorkerNodePools: `[{"machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Not unique names": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Empty machine type": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Missing machine type": {
			additionalWorkerNodePools: `[{"name": "name-1", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Missing HA zones": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Missing autoScalerMin": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMax": 3}]`,
			expectedError:             true,
		},
		"Missing autoScalerMax": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 20}]`,
			expectedError:             true,
		},
		"AutoScalerMin smaller than 3 when HA zones are enabled": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 2, "autoScalerMax": 300}]`,
			expectedError:             true,
		},
		"AutoScalerMax bigger than 300": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 301}]`,
			expectedError:             true,
		},
		"AutoScalerMin bigger than autoScalerMax": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 20, "autoScalerMax": 3}]`,
			expectedError:             true,
		},
		"Name contains capital letter": {
			additionalWorkerNodePools: `[{"name": "workerName", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Name starts with hyphen": {
			additionalWorkerNodePools: `[{"name": "-name", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Name ends with hyphen": {
			additionalWorkerNodePools: `[{"name": "name-", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Name longer than 15 characters": {
			additionalWorkerNodePools: `[{"name": "aaaaaaaaaaaaaaaa", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Name equal cpu-worker-0": {
			additionalWorkerNodePools: `[{"name": "cpu-worker-0", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             true,
		},
		"Min values of AutoScalerMin and AutoScalerMax when HA zones are disabled": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": false, "autoScalerMin": 0, "autoScalerMax": 1}]`,
			expectedError:             false,
		},
		"AutoScalerMin set to zero when HA zones are enabled": {
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 0, "autoScalerMax": 3}]`,
			expectedError:             true,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			memoryStorage := storage.NewMemoryStorage()

			queue := &automock.Queue{}
			queue.On("Add", mock.AnythingOfType("string"))

			factoryBuilder := &automock.PlanValidator{}
			factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

			kcBuilder := &kcMock.KcBuilder{}
			kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
			// #create provisioner endpoint
			provisionEndpoint := broker.NewProvision(
				broker.Config{
					EnablePlans:          []string{"aws"},
					URL:                  brokerURL,
					OnlySingleTrialPerGA: true,
				},
				gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
				imConfigFixture,
				memoryStorage,
				queue,
				broker.PlansConfig{},
				log,
				dashboardConfig,
				kcBuilder,
				whitelist.Set{},
				newSchemaService(t),
				newProviderSpec(t),
				fixValueProvider(t),
				false,
				config.FakeProviderConfigProvider{},
				nil,
				nil,
				nil,
				nil,
				nil,
				map[string]string{},
			)

			// when
			_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
				ServiceID:     serviceID,
				PlanID:        broker.AWSPlanID,
				RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","additionalWorkerNodePools": %s }`, clusterName, "eu-central-1", tc.additionalWorkerNodePools)),
				RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
			}, true)
			t.Logf("%+v\n", *provisionEndpoint)

			// then
			assert.Equal(t, tc.expectedError, err != nil)
		})
	}
}

func TestAdditionalWorkerNodePoolsForUnsupportedPlans(t *testing.T) {
	for tn, tc := range map[string]struct {
		planID string
	}{
		"Trial": {
			planID: broker.TrialPlanID,
		},
		"Free": {
			planID: broker.FreemiumPlanID,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			memoryStorage := storage.NewMemoryStorage()

			queue := &automock.Queue{}
			queue.On("Add", mock.AnythingOfType("string"))

			factoryBuilder := &automock.PlanValidator{}
			factoryBuilder.On("IsPlanSupport", tc.planID).Return(true)

			kcBuilder := &kcMock.KcBuilder{}
			kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
			// #create provisioner endpoint
			provisionEndpoint := broker.NewProvision(
				broker.Config{
					EnablePlans:          []string{"trial", "free"},
					URL:                  brokerURL,
					OnlySingleTrialPerGA: true,
				},
				gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
				imConfigFixture,
				memoryStorage,
				queue,
				broker.PlansConfig{},
				log,
				dashboardConfig,
				kcBuilder,
				whitelist.Set{},
				newSchemaService(t),
				newProviderSpec(t),
				fixValueProvider(t),
				false,
				config.FakeProviderConfigProvider{},
				nil,
				nil,
				nil,
				nil,
				nil,
				map[string]string{},
			)

			additionalWorkerNodePools := `[{"name": "name-1", "machineType": "m6i.large", "autoScalerMin": 3, "autoScalerMax": 20}]`

			// when
			_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
				ServiceID:     serviceID,
				PlanID:        tc.planID,
				RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","additionalWorkerNodePools": %s }`, clusterName, "eu-central-1", additionalWorkerNodePools)),
				RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
			}, true)
			t.Logf("%+v\n", *provisionEndpoint)

			// then
			assert.EqualError(t, err, fmt.Sprintf("additional worker node pools are not supported for plan ID: %s", tc.planID))
		})
	}
}

func TestAreNamesUnique(t *testing.T) {
	tests := []struct {
		name     string
		pools    []pkg.AdditionalWorkerNodePool
		expected bool
	}{
		{
			name: "Unique names",
			pools: []pkg.AdditionalWorkerNodePool{
				{Name: "name-1", MachineType: "m6i.large", HAZones: true, AutoScalerMin: 5, AutoScalerMax: 5},
				{Name: "name-2", MachineType: "m6i.large", HAZones: false, AutoScalerMin: 2, AutoScalerMax: 10},
				{Name: "name-3", MachineType: "m6i.large", HAZones: true, AutoScalerMin: 3, AutoScalerMax: 15},
			},
			expected: true,
		},
		{
			name: "Duplicate names",
			pools: []pkg.AdditionalWorkerNodePool{
				{Name: "name-1", MachineType: "m6i.large", HAZones: true, AutoScalerMin: 5, AutoScalerMax: 5},
				{Name: "name-2", MachineType: "m6i.large", HAZones: false, AutoScalerMin: 2, AutoScalerMax: 10},
				{Name: "name-1", MachineType: "m6i.large", HAZones: true, AutoScalerMin: 3, AutoScalerMax: 5},
			},
			expected: false,
		},
		{
			name:     "Empty list",
			pools:    []pkg.AdditionalWorkerNodePool{},
			expected: true,
		},
		{
			name: "Single pool",
			pools: []pkg.AdditionalWorkerNodePool{
				{Name: "name-1", MachineType: "m6i.large", HAZones: false, AutoScalerMin: 1, AutoScalerMax: 5},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, broker.AreNamesUnique(tt.pools))
		})
	}
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

			log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			queue := &automock.Queue{}
			queue.On("Add", mock.AnythingOfType("string"))

			factoryBuilder := &automock.PlanValidator{}
			factoryBuilder.On("IsPlanSupport", mock.AnythingOfType("string")).Return(true)

			kcBuilder := &kcMock.KcBuilder{}
			// #create provisioner endpoint
			provisionEndpoint := broker.NewProvision(
				broker.Config{EnablePlans: []string{"gcp", "azure", "free"}},
				gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
				imConfigFixture,
				memoryStorage,
				queue,
				broker.PlansConfig{},
				log,
				dashboardConfig,
				kcBuilder,
				whitelist.Set{},
				newSchemaService(t),
				newProviderSpec(t),
				fixValueProvider(t),
				false,
				config.FakeProviderConfigProvider{},
				nil,
				nil,
				nil,
				nil,
				nil,
				map[string]string{},
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

			log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			queue := &automock.Queue{}
			queue.On("Add", mock.AnythingOfType("string"))

			factoryBuilder := &automock.PlanValidator{}
			factoryBuilder.On("IsPlanSupport", tc.planID).Return(true)

			kcBuilder := &kcMock.KcBuilder{}
			// #create provisioner endpoint
			provisionEndpoint := broker.NewProvision(
				broker.Config{EnablePlans: []string{"gcp", "azure", "free"}, OnlySingleTrialPerGA: true},
				gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
				imConfigFixture,
				memoryStorage,
				queue,
				broker.PlansConfig{},
				log,
				dashboardConfig,
				kcBuilder,
				whitelist.Set{},
				newSchemaService(t),
				newProviderSpec(t),
				fixValueProvider(t),
				false,
				config.FakeProviderConfigProvider{},
				nil,
				nil,
				nil,
				nil,
				nil,
				map[string]string{},
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
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	t.Run("Should succeed if converged cloud is enabled and converged plan selected", func(t *testing.T) {
		// given
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.SapConvergedCloudPlanID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans: []string{broker.SapConvergedCloudPlanName},
				URL:         brokerURL,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu20"), instanceID, domain.ProvisionDetails{
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

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans: []string{"gcp", "azure"},
				URL:         brokerURL,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu11"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s","oidc":{ %s }}`, clusterName, "eastus", oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
	})
}

func TestUnsupportedMachineType(t *testing.T) {
	// given
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	memoryStorage := storage.NewMemoryStorage()

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", broker.GCPPlanID).Return(true)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
	// #create provisioner endpoint
	provisionEndpoint := broker.NewProvision(
		broker.Config{
			EnablePlans:          []string{"gcp"},
			URL:                  brokerURL,
			OnlySingleTrialPerGA: true,
		},
		gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
		imConfigFixture,
		memoryStorage,
		queue,
		broker.PlansConfig{},
		log,
		dashboardConfig,
		kcBuilder,
		whitelist.Set{},
		newSchemaService(t),
		newProviderSpec(t),
		fixValueProvider(t),
		false,
		config.FakeProviderConfigProvider{},
		nil,
		nil,
		nil,
		nil,
		nil,
		map[string]string{},
	)

	// when
	_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
		ServiceID:     serviceID,
		PlanID:        broker.GCPPlanID,
		RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "machineType": "%s" }`, clusterName, "europe-west3", "c2d-highmem-32")),
		RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
	}, true)
	t.Logf("%+v\n", *provisionEndpoint)

	// then
	assert.EqualError(t, err, "In the region europe-west3, the machine type c2d-highmem-32 is not available, it is supported in the southamerica-east1, us-central1")
}

func TestUnsupportedMachineTypeInAdditionalWorkerNodePools(t *testing.T) {
	// given
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	memoryStorage := storage.NewMemoryStorage()

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
	// #create provisioner endpoint
	provisionEndpoint := broker.NewProvision(
		broker.Config{
			EnablePlans:          []string{"aws"},
			URL:                  brokerURL,
			OnlySingleTrialPerGA: true,
		},
		gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
		imConfigFixture,
		memoryStorage,
		queue,
		broker.PlansConfig{},
		log,
		dashboardConfig,
		kcBuilder,
		whitelist.Set{},
		newSchemaService(t),
		newProviderSpec(t),
		fixValueProvider(t),
		false,
		config.FakeProviderConfigProvider{},
		nil,
		nil,
		nil,
		nil,
		nil,
		map[string]string{},
	)

	testCases := []struct {
		name                      string
		additionalWorkerNodePools string
		expectedError             string
	}{
		{
			name:                      "Single unsupported machine type",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m8g.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the region eu-central-1, the following machine types are not available: m8g.large (used in: name-1), it is supported in the ap-northeast-1, ap-southeast-1, ca-central-1",
		},
		{
			name:                      "Multiple unsupported machine types",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m8g.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "m7g.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the region eu-central-1, the following machine types are not available: m8g.large (used in: name-1), it is supported in the ap-northeast-1, ap-southeast-1, ca-central-1; m7g.xlarge (used in: name-2), it is supported in the us-west-2",
		},
		{
			name:                      "Duplicate unsupported machine type",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m8g.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "m8g.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the region eu-central-1, the following machine types are not available: m8g.large (used in: name-1, name-2), it is supported in the ap-northeast-1, ap-southeast-1, ca-central-1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
				ServiceID:     serviceID,
				PlanID:        broker.AWSPlanID,
				RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "additionalWorkerNodePools": %s }`, clusterName, "eu-central-1", tc.additionalWorkerNodePools)),
				RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
			}, true)
			t.Logf("%+v\n", *provisionEndpoint)

			// then
			assert.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestGPUMachineForInternalUser(t *testing.T) {
	// given
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	memoryStorage := storage.NewMemoryStorage()

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
	// #create provisioner endpoint
	provisionEndpoint := broker.NewProvision(
		broker.Config{
			EnablePlans:          []string{"aws"},
			URL:                  brokerURL,
			OnlySingleTrialPerGA: true,
		},
		gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
		imConfigFixture,
		memoryStorage,
		queue,
		broker.PlansConfig{},
		log,
		dashboardConfig,
		kcBuilder,
		whitelist.Set{},
		newSchemaService(t),
		newProviderSpec(t),
		fixValueProvider(t),
		false,
		config.FakeProviderConfigProvider{},
		nil,
		nil,
		nil,
		nil,
		nil,
		map[string]string{},
	)

	additionalWorkerNodePools := `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`
	// when
	_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
		ServiceID:     serviceID,
		PlanID:        broker.AWSPlanID,
		RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "additionalWorkerNodePools": %s }`, clusterName, "eu-central-1", additionalWorkerNodePools)),
		RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s", "license_type": "SAPDEV"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
	}, true)
	t.Logf("%+v\n", *provisionEndpoint)

	// then
	assert.NoError(t, err)
}

func TestGPUMachinesForExternalCustomer(t *testing.T) {
	// given
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	memoryStorage := storage.NewMemoryStorage()

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
	// #create provisioner endpoint
	provisionEndpoint := broker.NewProvision(
		broker.Config{
			EnablePlans:          []string{"aws", "azure", "gcp"},
			URL:                  brokerURL,
			OnlySingleTrialPerGA: true,
		},
		gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
		imConfigFixture,
		memoryStorage,
		queue,
		broker.PlansConfig{},
		log,
		dashboardConfig,
		kcBuilder,
		whitelist.Set{},
		newSchemaService(t),
		newProviderSpec(t),
		fixValueProvider(t),
		false,
		config.FakeProviderConfigProvider{},
		nil,
		nil,
		nil,
		nil,
		nil,
		map[string]string{},
	)

	testCases := []struct {
		name                      string
		planID                    string
		additionalWorkerNodePools string
		expectedError             string
	}{
		{
			name:                      "Single AWS G6 GPU machine type",
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g6.xlarge (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Multiple AWS G6 GPU machine types",
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g6.2xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g6.xlarge (used in worker node pools: name-1), g6.2xlarge (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Duplicate AWS G6 GPU machine type",
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g6.xlarge (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Single AWS G4dn GPU machine type",
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g4dn.xlarge (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Multiple AWS G4dn GPU machine types",
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g4dn.2xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g4dn.xlarge (used in worker node pools: name-1), g4dn.2xlarge (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Duplicate AWS G4dn GPU machine type",
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g4dn.xlarge (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Single GCP GPU machine type",
			planID:                    broker.GCPPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g2-standard-4 (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Multiple GCP GPU machine types",
			planID:                    broker.GCPPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g2-standard-8", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g2-standard-4 (used in worker node pools: name-1), g2-standard-8 (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Duplicate GCP GPU machine type",
			planID:                    broker.GCPPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g2-standard-4 (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Single Azure GPU machine type",
			planID:                    broker.AzurePlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: Standard_NC4as_T4_v3 (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Multiple Azure GPU machine types",
			planID:                    broker.AzurePlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "Standard_NC8as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: Standard_NC4as_T4_v3 (used in worker node pools: name-1), Standard_NC8as_T4_v3 (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		{
			name:                      "Duplicate Azure GPU machine type",
			planID:                    broker.AzurePlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: Standard_NC4as_T4_v3 (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
				ServiceID:     serviceID,
				PlanID:        tc.planID,
				RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "additionalWorkerNodePools": %s }`, clusterName, "eu-central-1", tc.additionalWorkerNodePools)),
				RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s", "license_type": "CUSTOMER"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
			}, true)
			t.Logf("%+v\n", *provisionEndpoint)

			// then
			assert.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestAutoScalerConfigurationInAdditionalWorkerNodePools(t *testing.T) {
	// given
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	memoryStorage := storage.NewMemoryStorage()

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
	// #create provisioner endpoint
	provisionEndpoint := broker.NewProvision(
		broker.Config{
			EnablePlans:          []string{"aws"},
			URL:                  brokerURL,
			OnlySingleTrialPerGA: true,
		},
		gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
		imConfigFixture,
		memoryStorage,
		queue,
		broker.PlansConfig{},
		log,
		dashboardConfig,
		kcBuilder,
		whitelist.Set{},
		newSchemaService(t),
		newProviderSpec(t),
		fixValueProvider(t),
		false,
		config.FakeProviderConfigProvider{},
		nil,
		nil,
		nil,
		nil,
		nil,
		map[string]string{},
	)

	testCases := []struct {
		name                      string
		additionalWorkerNodePools string
		expectedError             string
	}{
		{
			name:                      "Single auto scaler configuration error",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 20, "autoScalerMax": 3}]`,
			expectedError:             "The following additionalWorkerPools have validation issues: AutoScalerMax value 3 should be larger than AutoScalerMin value 20 for name-1 additional worker node pool.",
		},
		{
			name:                      "Multiple auto scaler configuration errors",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 20, "autoScalerMax": 3}, {"name": "name-2", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 1, "autoScalerMax": 20}]`,
			expectedError:             "The following additionalWorkerPools have validation issues: AutoScalerMax value 3 should be larger than AutoScalerMin value 20 for name-1 additional worker node pool; AutoScalerMin value 1 should be at least 3 when HA zones are enabled for name-2 additional worker node pool.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
				ServiceID:     serviceID,
				PlanID:        broker.AWSPlanID,
				RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "additionalWorkerNodePools": %s }`, clusterName, "eu-central-1", tc.additionalWorkerNodePools)),
				RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
			}, true)
			t.Logf("%+v\n", *provisionEndpoint)

			// then
			assert.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestAvailableZonesValidation(t *testing.T) {
	// given
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	memoryStorage := storage.NewMemoryStorage()

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
	// #create provisioner endpoint
	provisionEndpoint := broker.NewProvision(
		broker.Config{
			EnablePlans:          []string{"aws"},
			URL:                  brokerURL,
			OnlySingleTrialPerGA: true,
		},
		gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
		imConfigFixture,
		memoryStorage,
		queue,
		broker.PlansConfig{},
		log,
		dashboardConfig,
		kcBuilder,
		whitelist.Set{},
		newSchemaService(t),
		newProviderSpec(t),
		fixValueProvider(t),
		false,
		config.FakeProviderConfigProvider{},
		nil,
		nil,
		nil,
		nil,
		nil,
		map[string]string{},
	)

	additionalWorkerNodePools := `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`

	// when
	_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
		ServiceID:     serviceID,
		PlanID:        broker.AWSPlanID,
		RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "additionalWorkerNodePools": %s }`, clusterName, "us-east-1", additionalWorkerNodePools)),
		RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
	}, true)
	t.Logf("%+v\n", *provisionEndpoint)

	// then
	assert.EqualError(t, err, "In the us-east-1, the machine types: g6.xlarge (used in worker node pools: name-1) are not available in 3 zones. If you want to use this machine types, set HA to false.")
}

func TestAdditionalProperties(t *testing.T) {
	t.Run("file should contain request with additional properties", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		expectedFile := filepath.Join(tempDir, additionalproperties.ProvisioningRequestsFileName)

		log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:                 []string{"aws"},
				URL:                         brokerURL,
				OnlySingleTrialPerGA:        true,
				MonitorAdditionalProperties: true,
				AdditionalPropertiesPath:    tempDir,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.AWSPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "test": "test"}`, clusterName, "eu-central-1")),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.NoError(t, err)

		contents, err := os.ReadFile(expectedFile)
		assert.NoError(t, err)

		lines := bytes.Split(contents, []byte("\n"))
		assert.Greater(t, len(lines), 0)
		var entry map[string]interface{}
		err = json.Unmarshal(lines[0], &entry)
		assert.NoError(t, err)

		assert.Equal(t, "any-global-account-id", entry["globalAccountID"])
		assert.Equal(t, subAccountID, entry["subAccountID"])
		assert.Equal(t, instanceID, entry["instanceID"])
	})

	t.Run("file should contain two requests with additional properties", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		expectedFile := filepath.Join(tempDir, additionalproperties.ProvisioningRequestsFileName)

		log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:                 []string{"aws"},
				URL:                         brokerURL,
				OnlySingleTrialPerGA:        true,
				MonitorAdditionalProperties: true,
				AdditionalPropertiesPath:    tempDir,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.AWSPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "test": "test"}`, clusterName, "eu-central-1")),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)
		assert.NoError(t, err)

		_, err = provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.AWSPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "test": "test"}`, clusterName, "eu-central-1")),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)
		assert.NoError(t, err)

		// then
		contents, err := os.ReadFile(expectedFile)
		assert.NoError(t, err)

		lines := bytes.Split(contents, []byte("\n"))
		assert.Equal(t, len(lines), 3)
		var entry map[string]interface{}

		err = json.Unmarshal(lines[0], &entry)
		assert.NoError(t, err)
		assert.Equal(t, "any-global-account-id", entry["globalAccountID"])
		assert.Equal(t, subAccountID, entry["subAccountID"])
		assert.Equal(t, instanceID, entry["instanceID"])

		err = json.Unmarshal(lines[1], &entry)
		assert.NoError(t, err)
		assert.Equal(t, "any-global-account-id", entry["globalAccountID"])
		assert.Equal(t, subAccountID, entry["subAccountID"])
		assert.Equal(t, instanceID, entry["instanceID"])
	})

	t.Run("file should not contain request without additional properties", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		expectedFile := filepath.Join(tempDir, additionalproperties.ProvisioningRequestsFileName)

		log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:                 []string{"aws"},
				URL:                         brokerURL,
				OnlySingleTrialPerGA:        true,
				MonitorAdditionalProperties: true,
				AdditionalPropertiesPath:    tempDir,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "cf-eu10"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.AWSPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, "eu-central-1")),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.NoError(t, err)

		contents, err := os.ReadFile(expectedFile)
		assert.Nil(t, contents)
		assert.Error(t, err)
	})
}

func TestSameRegionForSeedAndShoot(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	existingAWSSeedRegions := []string{"eu-central-1", "us-east-1"}

	t.Run("should succeed if configuration contains region from provisioning parameters", func(t *testing.T) {
		// given
		const expectedRegion = "eu-central-1"
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans: []string{broker.AWSPlanName},
				URL:         brokerURL,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, expectedRegion), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.AWSPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "colocateControlPlane": true, "oidc":{ %s }}`, clusterName, expectedRegion, oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "broker.tester@local.dev")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.NoError(t, err)
	})

	t.Run("should fail if configuration does not contain region from provisioning parameters", func(t *testing.T) {
		// given
		const missingRegion = "eu-west-2"
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans: []string{broker.AWSPlanName},
				URL:         brokerURL,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`
		expectedErr := fmt.Errorf("[instanceID: %s] cannot colocate the control plane in the %s region. Provider aws can have control planes in the following regions: %s",
			instanceID, missingRegion, existingAWSSeedRegions)
		expectedAPIResponse := apiresponses.NewFailureResponse(
			expectedErr,
			http.StatusBadRequest,
			expectedErr.Error())

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, missingRegion), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.AWSPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "colocateControlPlane": true, "oidc":{ %s }}`, clusterName, missingRegion, oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "broker.tester@local.dev")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedAPIResponse.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedAPIResponse.(*apiresponses.FailureResponse).LoggerAction(), apierr.LoggerAction())
	})

	t.Run("should fail for unsupported region", func(t *testing.T) {
		// given
		const unsupportedRegion = "eu-west-1"
		memoryStorage := storage.NewMemoryStorage()

		queue := &automock.Queue{}
		queue.On("Add", mock.AnythingOfType("string"))

		factoryBuilder := &automock.PlanValidator{}
		factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

		kcBuilder := &kcMock.KcBuilder{}
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans: []string{broker.AWSPlanName},
				URL:         brokerURL,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256"]`
		expectedErr := fmt.Errorf("[instanceID: %s] cannot colocate the control plane in the %s region. Provider aws can have control planes in the following regions: %s",
			instanceID, unsupportedRegion, existingAWSSeedRegions)
		expectedAPIResponse := apiresponses.NewFailureResponse(
			expectedErr,
			http.StatusBadRequest,
			expectedErr.Error())

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, unsupportedRegion), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.AWSPlanID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "colocateControlPlane": true, "oidc":{ %s }}`, clusterName, unsupportedRegion, oidcParams)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "broker.tester@local.dev")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedAPIResponse.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedAPIResponse.(*apiresponses.FailureResponse).LoggerAction(), apierr.LoggerAction())
	})
}

func TestQuotaLimitCheck(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", planID).Return(true)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))

	t.Run("should create new operation if there is no instances", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()

		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
				CheckQuotaLimit:      true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			nil,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		_, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.NoError(t, err)
	})

	t.Run("should fail if there is no unassigned quota", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()
		instance := fixInstance()
		instance.SubAccountID = subAccountID
		err := memoryStorage.Instances().Insert(instance)
		assert.NoError(t, err)

		quotaClient := &automock.QuotaClient{}
		quotaClient.On("GetQuota", subAccountID, broker.AzurePlanName).Return(1, nil)

		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
				CheckQuotaLimit:      true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			quotaClient,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.EqualError(t, err, "Kyma instances quota exceeded for plan azure. assignedQuota: 1, remainingQuota: 0. Contact your administrator.")
	})

	t.Run("should create new operation if there is unassigned quota", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()
		instance := fixInstance()
		instance.SubAccountID = subAccountID
		err := memoryStorage.Instances().Insert(instance)
		assert.NoError(t, err)

		quotaClient := &automock.QuotaClient{}
		quotaClient.On("GetQuota", subAccountID, broker.AzurePlanName).Return(2, nil)

		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
				CheckQuotaLimit:      true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			quotaClient,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.NoError(t, err)
	})

	t.Run("should fail if quota client returns error", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()
		instance := fixInstance()
		instance.SubAccountID = subAccountID
		err := memoryStorage.Instances().Insert(instance)
		assert.NoError(t, err)

		quotaClient := &automock.QuotaClient{}
		quotaClient.On("GetQuota", subAccountID, broker.AzurePlanName).Return(0, fmt.Errorf("error message"))

		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
				CheckQuotaLimit:      true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			quotaClient,
			nil,
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.EqualError(t, err, "Failed to get assigned quota for plan azure: error message.")
	})

	t.Run("should create new operation if there is no unassigned quota but whitelisted subaccount", func(t *testing.T) {
		// given
		// #setup memory storage
		memoryStorage := storage.NewMemoryStorage()
		instance := fixInstance()
		instance.SubAccountID = subAccountID
		err := memoryStorage.Instances().Insert(instance)
		assert.NoError(t, err)

		quotaClient := &automock.QuotaClient{}
		quotaClient.On("GetQuota", subAccountID, broker.AzurePlanName).Return(1, nil)

		// #create provisioner endpoint
		provisionEndpoint := broker.NewProvision(
			broker.Config{
				EnablePlans:          []string{"gcp", "azure"},
				URL:                  brokerURL,
				OnlySingleTrialPerGA: true,
				CheckQuotaLimit:      true,
			},
			gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
			imConfigFixture,
			memoryStorage,
			queue,
			broker.PlansConfig{},
			log,
			dashboardConfig,
			kcBuilder,
			whitelist.Set{},
			newSchemaService(t),
			newProviderSpec(t),
			fixValueProvider(t),
			false,
			config.FakeProviderConfigProvider{},
			quotaClient,
			whitelist.Set{subAccountID: struct{}{}},
			nil,
			nil,
			nil,
			map[string]string{},
		)

		// when
		_, err = provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        planID,
			RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
		}, true)
		t.Logf("%+v\n", *provisionEndpoint)

		// then
		assert.NoError(t, err)
	})
}

func TestDiscoveryZones(t *testing.T) {
	// given
	memoryStorage := storage.NewMemoryStorage()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", planID).Return(true)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))

	rulesService, err := rules.NewRulesServiceFromSlice([]string{"aws"}, sets.New("aws"), sets.New("aws"))
	require.NoError(t, err)

	testCases := []struct {
		name                      string
		zones                     map[string][]string
		awsError                  error
		additionalWorkerNodePools string
		expectedError             string
	}{
		{
			name:                      "Should fail if AWS returns error for Kyma worker node pool",
			awsError:                  fmt.Errorf("AWS error"),
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "Failed to validate the number of available zones. Please try again later.",
		},
		{
			name: "Should fail if not enough zones for Kyma worker node pool",
			zones: map[string][]string{
				"m6i.large": {"eu-west-2a", "eu-west-2b"},
			},
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the eu-west-2, the m6i.large machine type is not available in 3 zones.",
		},
		{
			name: "Should fail if machine type in additional worker node pool is not available",
			zones: map[string][]string{
				"m6i.large": {"eu-west-2a", "eu-west-2b", "eu-west-2c", "eu-west-2d"},
				"g6.xlarge": {},
			},
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the eu-west-2, the g6.xlarge machine type is not available.",
		},
		{
			name: "Should fail if machine type in high availability additional worker node pool is not available in at least 3 zones",
			zones: map[string][]string{
				"m6i.large": {"eu-west-2a", "eu-west-2b", "eu-west-2c", "eu-west-2d"},
				"g6.xlarge": {"eu-west-2a", "eu-west-2b"},
			},
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the eu-west-2, the machine types: g6.xlarge (used in worker node pools: name-1) are not available in 3 zones. If you want to use this machine types, set HA to false.",
		},
		{
			name: "Should fail if machine types in high availability additional worker node pools are not available in at least 3 zones",
			zones: map[string][]string{
				"m6i.large":   {"eu-west-2a", "eu-west-2b", "eu-west-2c", "eu-west-2d"},
				"g6.xlarge":   {"eu-west-2a", "eu-west-2b"},
				"g4dn.xlarge": {"eu-west-2a", "eu-west-2b"},
			},
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-3", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the eu-west-2, the machine types: g6.xlarge (used in worker node pools: name-1, name-2), g4dn.xlarge (used in worker node pools: name-3) are not available in 3 zones. If you want to use this machine types, set HA to false.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provisionEndpoint := broker.NewProvision(
				broker.Config{
					EnablePlans:          []string{"aws"},
					URL:                  brokerURL,
					OnlySingleTrialPerGA: true,
					CheckQuotaLimit:      true,
				},
				gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
				imConfigFixture,
				memoryStorage,
				queue,
				broker.PlansConfig{},
				log,
				dashboardConfig,
				kcBuilder,
				whitelist.Set{},
				newSchemaService(t),
				fixture.NewProviderSpecWithZonesDiscovery(t, true),
				fixValueProvider(t),
				false,
				config.FakeProviderConfigProvider{},
				nil,
				nil,
				rulesService,
				fixture.CreateGardenerClient(),
				fixture.NewFakeAWSClientFactory(tc.zones, tc.awsError),
				map[string]string{},
			)

			// when
			_, err := provisionEndpoint.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
				ServiceID:     serviceID,
				PlanID:        broker.AWSPlanID,
				RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s", "machineType": "%s", "additionalWorkerNodePools": %s}`, clusterName, "eu-west-2", "m6i.large", tc.additionalWorkerNodePools)),
				RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, "Test@Test.pl")),
			}, true)

			// then
			assert.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestClusterName(t *testing.T) {
	for tn, tc := range map[string]struct {
		parameters    string
		expectedError error
	}{
		"Missing name": {
			parameters:    `{"region": "eu-central-1"}`,
			expectedError: fmt.Errorf("while validating input parameters: at '': missing property 'name'"),
		},
		"Null name": {
			parameters:    `{"name": null, "region": "eu-central-1"}`,
			expectedError: fmt.Errorf("while validating input parameters: at '/name': got null, want string"),
		},
		"Empty name": {
			parameters:    `{"name": "", "region": "eu-central-1"}`,
			expectedError: fmt.Errorf("while validating input parameters: at '/name': minLength: got 0, want 1"),
		},
		"Long name": {
			parameters:    fmt.Sprintf(`{"name": "%s", "region": "eu-central-1"}`, strings.Repeat("A", 65)),
			expectedError: fmt.Errorf("while validating input parameters: at '/name': maxLength: got 65, want 64"),
		},
		"Valid name": {
			parameters:    `{"name": "cluster-testing", "region": "eu-central-1"}`,
			expectedError: nil,
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			memoryStorage := storage.NewMemoryStorage()

			queue := &automock.Queue{}
			queue.On("Add", mock.AnythingOfType("string"))

			factoryBuilder := &automock.PlanValidator{}
			factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

			kcBuilder := &kcMock.KcBuilder{}
			kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))

			provisionEndpoint := broker.NewProvision(
				broker.Config{
					EnablePlans:          []string{"aws"},
					URL:                  brokerURL,
					OnlySingleTrialPerGA: true,
				},
				gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
				imConfigFixture,
				memoryStorage,
				queue,
				broker.PlansConfig{},
				log,
				dashboardConfig,
				kcBuilder,
				whitelist.Set{},
				newSchemaService(t),
				newProviderSpec(t),
				fixValueProvider(t),
				false,
				config.FakeProviderConfigProvider{},
				nil,
				nil,
				nil,
				nil,
				nil,
				map[string]string{},
			)

			// when
			_, err := provisionEndpoint.Provision(
				fixRequestContext(t, "cf-eu10"),
				instanceID,
				domain.ProvisionDetails{
					ServiceID:     serviceID,
					PlanID:        broker.AWSPlanID,
					RawParameters: json.RawMessage(tc.parameters),
					RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
				},
				true,
			)

			// then
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestBtpRegionsMigrationSapConvergedCloud_DeprecatedRegion(t *testing.T) {
	// given
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	memoryStorage := storage.NewMemoryStorage()

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))

	provisionEndpoint := broker.NewProvision(
		broker.Config{
			EnablePlans:          []string{"sap-converged-cloud"},
			URL:                  brokerURL,
			OnlySingleTrialPerGA: true,
		},
		gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
		imConfigFixture,
		memoryStorage,
		queue,
		broker.PlansConfig{},
		log,
		dashboardConfig,
		kcBuilder,
		whitelist.Set{},
		newSchemaService(t),
		newProviderSpec(t),
		fixValueProvider(t),
		false,
		config.FakeProviderConfigProvider{},
		nil,
		nil,
		nil,
		nil,
		nil,
		map[string]string{"cf-eu10": "cf-eu01"},
	)

	// when
	_, err := provisionEndpoint.Provision(
		fixRequestContext(t, "cf-eu10"),
		instanceID,
		domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.SapConvergedCloudPlanID,
			RawParameters: json.RawMessage(`{"name": "testing-name", "region": "eu-de-2"}`),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		},
		true,
	)

	assert.EqualError(t, err, "Cluster provisioning in the eu-de-2 region is no longer supported under cf-eu10. Please use the cf-eu01 BTP region.")
}

func TestBtpRegionsMigrationSapConvergedCloud_NewRegion(t *testing.T) {
	// given
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	memoryStorage := storage.NewMemoryStorage()

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", broker.AWSPlanID).Return(true)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))

	provisionEndpoint := broker.NewProvision(
		broker.Config{
			EnablePlans:          []string{"sap-converged-cloud"},
			URL:                  brokerURL,
			OnlySingleTrialPerGA: true,
		},
		gardener.Config{Project: "test", ShootDomain: "example.com", DNSProviders: fixDNSProviders()},
		imConfigFixture,
		memoryStorage,
		queue,
		broker.PlansConfig{},
		log,
		dashboardConfig,
		kcBuilder,
		whitelist.Set{},
		newSchemaService(t),
		newProviderSpec(t),
		fixValueProvider(t),
		false,
		config.FakeProviderConfigProvider{},
		nil,
		nil,
		nil,
		nil,
		nil,
		map[string]string{"cf-eu10": "cf-eu01"},
	)

	// when
	_, err := provisionEndpoint.Provision(
		fixRequestContext(t, "cf-eu01"),
		instanceID,
		domain.ProvisionDetails{
			ServiceID:     serviceID,
			PlanID:        broker.SapConvergedCloudPlanID,
			RawParameters: json.RawMessage(`{"name": "testing-name", "region": "eu-de-2"}`),
			RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, "any-global-account-id", subAccountID, "Test@Test.pl")),
		},
		true,
	)

	assert.NoError(t, err)
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

func fixRegionsSupportingMachine() configuration.RegionsSupportingMachine {
	return configuration.RegionsSupportingMachine{
		"m8g": {
			"ap-northeast-1": nil,
			"ap-southeast-1": nil,
			"ca-central-1":   nil,
		},
		"m7g": {
			"us-west-2": nil,
		},
		"c2d-highmem": {
			"us-central1":        nil,
			"southamerica-east1": nil,
		},
		"Standard_D": {
			"uksouth":     nil,
			"brazilsouth": nil,
		},
		"g6": {
			"us-east-1":    []string{"a"},
			"westeurope":   []string{"b"},
			"eu-central-1": nil,
		},
	}
}

func newSchemaService(t *testing.T) *broker.SchemaService {
	plans := newPlanSpec(t)
	provider := newProviderSpec(t)

	channelResolver := &fixture.FakeChannelResolver{}
	schemaService := broker.NewSchemaService(provider, plans, nil, broker.Config{},
		broker.EnablePlans{broker.TrialPlanName, broker.AzurePlanName, broker.AzureLitePlanName, broker.AWSPlanName,
			broker.GCPPlanName, broker.SapConvergedCloudPlanName, broker.FreemiumPlanName}, channelResolver)
	return schemaService
}

func newProviderSpec(t *testing.T) *configuration.ProviderSpec {
	spec, err := configuration.NewProviderSpecFromFile("testdata/providers.yaml")
	require.NoError(t, err)
	return spec
}

func newPlanSpec(t *testing.T) *configuration.PlanSpecifications {
	spec, err := configuration.NewPlanSpecificationsFromFile("testdata/plans.yaml")
	require.NoError(t, err)
	return spec
}

func newEmptyProviderSpec() *configuration.ProviderSpec {
	spec, _ := configuration.NewProviderSpec(strings.NewReader(""))
	return spec
}
