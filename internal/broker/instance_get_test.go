package broker_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/config"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"

	"github.com/kyma-project/kyma-environment-broker/common/gardener"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/broker/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	kcMock "github.com/kyma-project/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetEndpoint_GetNonExistingInstance(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	kcBuilder := &kcMock.KcBuilder{}
	svc := broker.NewGetInstance(broker.Config{}, st.Instances(), st.Operations(), kcBuilder, fixLogger())

	// when
	_, err := svc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})

	// then
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Get returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, http.StatusNotFound, apierr.ValidatedStatusCode(nil), "Get status code not matching")
}

func TestGetEndpoint_GetProvisioningInstance(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	factoryBuilder := &automock.PlanValidator{}
	factoryBuilder.On("IsPlanSupport", planID).Return(true)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", "").Return("", fmt.Errorf("error"))
	createSvc := broker.NewProvision(
		broker.Config{EnablePlans: []string{"gcp", "azure"}, OnlySingleTrialPerGA: true},
		gardener.Config{Project: "test", ShootDomain: "example.com"},
		broker.InfrastructureManager{},
		st,
		queue,
		broker.PlansConfig{},
		fixLogger(),
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
	getSvc := broker.NewGetInstance(broker.Config{}, st.Instances(), st.Operations(), kcBuilder, fixLogger())

	// when
	_, err := createSvc.Provision(fixRequestContext(t, "req-region"), instanceID, domain.ProvisionDetails{
		ServiceID:     serviceID,
		PlanID:        planID,
		RawParameters: json.RawMessage(fmt.Sprintf(`{"name": "%s", "region": "%s"}`, clusterName, clusterRegion)),
		RawContext:    json.RawMessage(fmt.Sprintf(`{"globalaccount_id": "%s", "subaccount_id": "%s", "user_id": "%s"}`, globalAccountID, subAccountID, userID)),
	}, true)
	assert.NoError(t, err)

	// then
	_, err = getSvc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Get returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, http.StatusNotFound, apierr.ValidatedStatusCode(nil), "Get status code not matching")
	assert.Equal(t, "provisioning of instanceID d3d5dca4-5dc8-44ee-a825-755c2a3fb839 in progress", apierr.Error())

	// when
	op, _ := st.Operations().GetProvisioningOperationByInstanceID(instanceID)
	op.State = domain.Succeeded
	_, err = st.Operations().UpdateProvisioningOperation(*op)
	assert.NoError(t, err)

	// then
	response, err := getSvc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})
	assert.Equal(t, nil, err, "Get returned error when expected to pass")
	assert.Len(t, response.Metadata.Labels, 1)
	assert.Regexp(t, dashboardUrlRegex, response.DashboardURL)
}

func TestGetEndpoint_DoNotReturnInstanceWhereDeletedAtIsNotZero(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	cfg := broker.Config{
		URL: "https://test-broker.local",
	}

	const (
		instanceID  = "cluster-test"
		operationID = "operationID"
	)
	op := fixture.FixProvisioningOperation(operationID, instanceID)

	instance := fixture.FixInstance(instanceID)
	instance.DeletedAt = time.Now()
	kcBuilder := &kcMock.KcBuilder{}

	err := st.Operations().InsertOperation(op)
	require.NoError(t, err)

	err = st.Instances().Insert(instance)
	require.NoError(t, err)

	svc := broker.NewGetInstance(cfg, st.Instances(), st.Operations(), kcBuilder, fixLogger())

	// when
	_, err = svc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})

	// then
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Get request returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, http.StatusNotFound, apierr.ValidatedStatusCode(nil), "Get request status code not matching")
}

func TestGetEndpoint_GetExpiredInstanceWithExpirationDetails(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	cfg := broker.Config{
		URL: "https://test-broker.local",
	}

	const (
		instanceID  = "cluster-test"
		operationID = "operationID"
	)
	op := fixture.FixProvisioningOperation(operationID, instanceID)

	instance := fixture.FixInstance(instanceID)
	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", instance.RuntimeID).Return("https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com", nil)
	instance.ServicePlanID = broker.TrialPlanID
	instance.CreatedAt = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	expireTime := instance.CreatedAt.Add(time.Hour * 24 * 14)
	instance.ExpiredAt = &expireTime

	err := st.Operations().InsertOperation(op)
	require.NoError(t, err)

	err = st.Instances().Insert(instance)
	require.NoError(t, err)

	svc := broker.NewGetInstance(cfg, st.Instances(), st.Operations(), kcBuilder, fixLogger())
	// when
	response, err := svc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})

	// then
	require.NoError(t, err)
	assert.True(t, instance.IsExpired())
	assert.Equal(t, instance.ServiceID, response.ServiceID)
	assert.NotContains(t, response.Metadata.Labels, "KubeconfigURL")
	assert.NotContains(t, response.Metadata.Labels, "APIServerURL")
	assert.Contains(t, response.Metadata.Labels, "Trial account expiration details")
	assert.Contains(t, response.Metadata.Labels, "Trial account documentation")
}

func TestGetEndpoint_GetExpiredInstanceWithExpirationDetailsAllSubaccountsIDs(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	cfg := broker.Config{
		URL: "https://test-broker.local",
	}

	const (
		instanceID  = "cluster-test"
		operationID = "operationID"
	)
	op := fixture.FixProvisioningOperation(operationID, instanceID)

	instance := fixture.FixInstance(instanceID)
	instance.SubAccountID = "test-subaccount-id"
	instance.ServicePlanID = broker.TrialPlanID
	instance.CreatedAt = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	expireTime := instance.CreatedAt.Add(time.Hour * 24 * 14)
	instance.ExpiredAt = &expireTime
	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", instance.RuntimeID).Return("https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com", nil)

	err := st.Operations().InsertOperation(op)
	require.NoError(t, err)

	err = st.Instances().Insert(instance)
	require.NoError(t, err)

	svc := broker.NewGetInstance(cfg, st.Instances(), st.Operations(), kcBuilder, fixLogger())

	// when
	response, err := svc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})

	// then
	require.NoError(t, err)
	assert.True(t, instance.IsExpired())
	assert.Equal(t, instance.ServiceID, response.ServiceID)
	assert.NotContains(t, response.Metadata.Labels, "KubeconfigURL")
	assert.NotContains(t, response.Metadata.Labels, "APIServerURL")
	assert.Contains(t, response.Metadata.Labels, "Trial account expiration details")
	assert.Contains(t, response.Metadata.Labels, "Trial account documentation")
}

func TestGetEndpoint_GetExpiredFreeInstanceWithExpirationDetails(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	cfg := broker.Config{
		URL: "https://test-broker.local",
	}

	const (
		instanceID  = "cluster-test"
		operationID = "operationID"
	)
	op := fixture.FixProvisioningOperation(operationID, instanceID)

	instance := fixture.FixInstance(instanceID)
	instance.ServicePlanID = broker.FreemiumPlanID
	instance.CreatedAt = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	expireTime := instance.CreatedAt.Add(time.Hour * 24 * 30)
	instance.ExpiredAt = &expireTime
	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", instance.RuntimeID).Return("https://api.ac0d8d9.kyma-dev.shoot.canary.k8s-hana.ondemand.com", nil)

	err := st.Operations().InsertOperation(op)
	require.NoError(t, err)

	err = st.Instances().Insert(instance)
	require.NoError(t, err)

	svc := broker.NewGetInstance(cfg, st.Instances(), st.Operations(), kcBuilder, fixLogger())

	// when
	response, err := svc.GetInstance(context.Background(), instanceID, domain.FetchInstanceDetails{})

	// then
	require.NoError(t, err)
	assert.True(t, instance.IsExpired())
	assert.Equal(t, instance.ServiceID, response.ServiceID)
	assert.NotContains(t, response.Metadata.Labels, "KubeconfigURL")
	assert.NotContains(t, response.Metadata.Labels, "APIServerURL")
	assert.Contains(t, response.Metadata.Labels, "Free plan expiration details")
	assert.Contains(t, response.Metadata.Labels, "Available plans documentation")
}

func fixLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
