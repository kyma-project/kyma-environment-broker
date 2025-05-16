package broker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/additionalproperties"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/provider"
	"github.com/kyma-project/kyma-environment-broker/internal/regionssupportingmachine"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/customresources"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/google/uuid"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker/automock"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/stretchr/testify/mock"

	"github.com/kyma-project/kyma-environment-broker/internal/dashboard"

	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	kcMock "github.com/kyma-project/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/pivotal-cf/brokerapi/v12/domain/apiresponses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var dashboardConfig = dashboard.Config{LandscapeURL: "https://dashboard.example.com"}
var fakeKcpK8sClient = fake.NewClientBuilder().Build()
var imConfigFixture = broker.InfrastructureManager{
	UseSmallerMachineTypes: false,
	EnableIngressFiltering: false,
	IngressFilteringPlans:  []string{"gcp", "azure", "aws"},
}

type handler struct {
	Instance   internal.Instance
	ersContext internal.ERSContext
}

func (h *handler) Handle(inst *internal.Instance, ers internal.ERSContext) (bool, error) {
	h.Instance = *inst
	h.ersContext = ers
	return false, nil
}

func TestUpdateEndpoint_UpdateSuspension(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: broker.TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          nil,
			},
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	err = st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("02"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	kcBuilder := &kcMock.KcBuilder{}
	svc := broker.NewUpdate(
		broker.Config{},
		st,
		handler,
		true,
		false,
		true,
		q,
		broker.PlansConfig{},
		nil,
		fixLogger(),
		dashboardConfig,
		kcBuilder,
		fakeKcpK8sClient,
		nil, imConfigFixture, newSchemaService(t))

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then

	assert.Equal(t, internal.ERSContext{
		Active: ptr.Bool(false),
	}, handler.ersContext)

	require.NotNil(t, handler.Instance.Parameters.ErsContext.Active)
	assert.True(t, *handler.Instance.Parameters.ErsContext.Active)
	assert.Len(t, response.Metadata.Labels, 1)

	inst, err := st.Instances().GetByID(instanceID)
	assert.False(t, *inst.Parameters.ErsContext.Active)
}

func TestUpdateEndpoint_UpdateOfExpiredTrial(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: broker.TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          ptr.Bool(false),
			},
		},
		ExpiredAt: ptr.Time(time.Now()),
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	kcBuilder := &kcMock.KcBuilder{}
	svc := broker.NewUpdate(broker.Config{}, st, handler, true, false, true, q, broker.PlansConfig{},
		nil, fixLogger(),
		dashboardConfig, kcBuilder, fakeKcpK8sClient, nil, imConfigFixture, newSchemaService(t))

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.TrialPlanID,
		RawParameters:   json.RawMessage(`{"autoScalerMin": 3}`),
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.Error(t, err)
	assert.ErrorContains(t, err, "cannot update an expired instance")
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusBadRequest, "Updating status code not matching")
	assert.False(t, response.IsAsync)
}

func TestUpdateEndpoint_UpdateAutoscalerParams(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: broker.AWSPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: broker.AWSPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          ptr.Bool(false),
			},
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))

	kcBuilder := &kcMock.KcBuilder{}
	svc := broker.NewUpdate(broker.Config{}, st, handler, true, false, true, q, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder,
		fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

	t.Run("Should fail on invalid (too low) autoScalerMin and autoScalerMax", func(t *testing.T) {

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AWSPlanID,
			RawParameters:   json.RawMessage(`{"autoScalerMin": 1, "autoScalerMax": 1}`),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"active\":false}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.ErrorContains(t, err, "while validating update parameters:")
		assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusBadRequest, "Updating status code not matching")
		assert.False(t, response.IsAsync)
	})

	t.Run("Should fail on invalid autoScalerMin and autoScalerMax", func(t *testing.T) {

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AWSPlanID,
			RawParameters:   json.RawMessage(`{"autoScalerMin": 4, "autoScalerMax": 3}`),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"active\":false}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.ErrorContains(t, err, "AutoScalerMax 3 should be larger than AutoScalerMin 4")
		assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusBadRequest, "Updating status code not matching")
		assert.False(t, response.IsAsync)
	})

	t.Run("Should fail on invalid autoScalerMin and autoScalerMax and JSON validation should precede", func(t *testing.T) {

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AWSPlanID,
			RawParameters:   json.RawMessage(`{"autoScalerMin": 2, "autoScalerMax": 1}`),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"active\":false}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.ErrorContains(t, err, "while validating update parameters:")
		assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusBadRequest, "Updating status code not matching")
		assert.False(t, response.IsAsync)
	})
}

func TestUpdateEndpoint_UpdateUnsuspension(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: broker.TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          nil,
			},
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	err = st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	kcBuilder := &kcMock.KcBuilder{}
	svc := broker.NewUpdate(broker.Config{}, st, handler, true, false, true, q, broker.PlansConfig{},
		nil, fixLogger(), dashboardConfig, kcBuilder,
		fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":true}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then

	assert.Equal(t, internal.ERSContext{
		Active: ptr.Bool(true),
	}, handler.ersContext)

	require.NotNil(t, handler.Instance.Parameters.ErsContext.Active)
	assert.False(t, *handler.Instance.Parameters.ErsContext.Active)
}

func TestUpdateEndpoint_UpdateInstanceWithWrongActiveValue(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: broker.TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          ptr.Bool(false),
			},
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	kcBuilder := &kcMock.KcBuilder{}
	svc := broker.NewUpdate(broker.Config{}, st, handler, true, false, true, q, broker.PlansConfig{},
		nil, fixLogger(), dashboardConfig, kcBuilder,
		fakeKcpK8sClient, nil, imConfigFixture, newSchemaService(t))

	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	assert.Equal(t, internal.ERSContext{
		Active: ptr.Bool(false),
	}, handler.ersContext)

	assert.True(t, *handler.Instance.Parameters.ErsContext.Active)
}

func TestUpdateEndpoint_UpdateNonExistingInstance(t *testing.T) {
	// given
	st := storage.NewMemoryStorage()
	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	kcBuilder := &kcMock.KcBuilder{}

	svc := broker.NewUpdate(broker.Config{}, st, handler, true, false, true, q, broker.PlansConfig{},
		nil, fixLogger(), dashboardConfig, kcBuilder,
		fakeKcpK8sClient, nil, imConfigFixture, newSchemaService(t))

	// when
	_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.IsType(t, err, &apiresponses.FailureResponse{}, "Updating returned error of unexpected type")
	apierr := err.(*apiresponses.FailureResponse)
	assert.Equal(t, apierr.ValidatedStatusCode(nil), http.StatusNotFound, "Updating status code not matching")
}

func fixProvisioningOperation(id string) internal.ProvisioningOperation {
	provisioningOperation := fixture.FixProvisioningOperation(id, instanceID)

	return internal.ProvisioningOperation{Operation: provisioningOperation}
}

func fixSuspensionOperation() internal.DeprovisioningOperation {
	deprovisioningOperation := fixture.FixDeprovisioningOperation("id", instanceID)
	deprovisioningOperation.Temporary = true

	return deprovisioningOperation
}

func TestUpdateEndpoint_UpdateGlobalAccountID(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:      instanceID,
		ServicePlanID:   broker.TrialPlanID,
		GlobalAccountID: "origin-account-id",
		Parameters: internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          nil,
			},
		},
	}
	newGlobalAccountID := "updated-account-id"
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	err = st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("02"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))

	kcBuilder := &kcMock.KcBuilder{}

	svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
		nil, fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, nil, imConfigFixture, newSchemaService(t))

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"globalaccount_id\":\"" + newGlobalAccountID + "\", \"active\":true}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	inst, err := st.Instances().GetByID(instanceID)
	require.NoError(t, err)
	// Check if SubscriptionGlobalAccountID is not empty
	assert.NotEmpty(t, inst.SubscriptionGlobalAccountID)

	// Check if SubscriptionGlobalAccountID is now the same as GlobalAccountID
	assert.Equal(t, inst.GlobalAccountID, newGlobalAccountID)

	require.NotNil(t, handler.Instance.Parameters.ErsContext.Active)
	assert.True(t, *handler.Instance.Parameters.ErsContext.Active)
	assert.Len(t, response.Metadata.Labels, 1)
}

func TestUpdateEndpoint_UpdateFromOIDCObject(t *testing.T) {
	// given
	instance := fixture.FixInstance(instanceID)
	instance.Parameters.Parameters.OIDC = &pkg.OIDCConnectDTO{
		OIDCConfigDTO: &pkg.OIDCConfigDTO{
			ClientID:       "client-id",
			GroupsClaim:    "groups",
			IssuerURL:      "https://test.local",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "email",
			UsernamePrefix: "-",
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.com", nil)

	svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

	t.Run("Should accept update to OIDC object", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"updated-client","groupsClaim":"groups","issuerURL":"https://test.com","signingAlgs":["RS256"],"usernameClaim":"email","usernamePrefix":"-"`

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)
		operation, err := st.Operations().GetProvisioningOperationByID(response.OperationData)

		// then
		require.NoError(t, err)
		assert.Equal(t, &pkg.OIDCConfigDTO{
			ClientID:       "updated-client",
			GroupsClaim:    "groups",
			IssuerURL:      "https://test.com",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "email",
			UsernamePrefix: "-",
		}, operation.ProvisioningParameters.Parameters.OIDC.OIDCConfigDTO)
	})

	t.Run("Should accept update to OIDC list", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"updated-client","groupsClaim":"groups","issuerURL":"https://test.com","signingAlgs":["RS256"],"usernameClaim":"email","usernamePrefix":"-","groupsPrefix":"-","requiredClaims":[]`

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{ \"list\":[{" + oidcParams + "}]}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)
		operation, err := st.Operations().GetProvisioningOperationByID(response.OperationData)

		// then
		require.NoError(t, err)
		assert.Equal(t, pkg.OIDCConfigDTO{
			ClientID:       "updated-client",
			GroupsClaim:    "groups",
			IssuerURL:      "https://test.com",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "email",
			UsernamePrefix: "-",
			GroupsPrefix:   "-",
			RequiredClaims: []string{},
		}, operation.ProvisioningParameters.Parameters.OIDC.List[0])
	})
	t.Run("Should accept update to empty OIDC list", func(t *testing.T) {
		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{ \"list\":[]}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)
		operation, err := st.Operations().GetProvisioningOperationByID(response.OperationData)

		// then
		require.NoError(t, err)
		assert.Len(t, operation.ProvisioningParameters.Parameters.OIDC.List, 0)
	})
}

func TestUpdateEndpoint_UpdateFromOIDCList(t *testing.T) {
	// given
	instance := fixture.FixInstance(instanceID)
	instance.Parameters.Parameters.OIDC = &pkg.OIDCConnectDTO{
		List: []pkg.OIDCConfigDTO{
			{
				ClientID:       "client-id",
				GroupsClaim:    "groups",
				IssuerURL:      "https://test.local",
				SigningAlgs:    []string{"RS256"},
				UsernameClaim:  "email",
				UsernamePrefix: "-",
			},
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.com", nil)

	svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

	t.Run("Should reject update to OIDC object", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"updated-client","groupsClaim":"groups","issuerURL":"https://test.com","signingAlgs":["RS256"],"usernameClaim":"email","usernamePrefix":"-"`

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.EqualError(t, err, "an object OIDC cannot be used because the instance OIDC configuration uses a list")
	})
	t.Run("Should accept update to OIDC list", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"updated-client","groupsClaim":"groups","issuerURL":"https://test.com","signingAlgs":["RS256"],"usernameClaim":"email","usernamePrefix":"-","groupsPrefix":"-","requiredClaims":[]`

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{ \"list\":[{" + oidcParams + "}]}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
		operation, err := st.Operations().GetProvisioningOperationByID(response.OperationData)

		// then
		require.NoError(t, err)
		assert.Equal(t, pkg.OIDCConfigDTO{
			ClientID:       "updated-client",
			GroupsClaim:    "groups",
			IssuerURL:      "https://test.com",
			SigningAlgs:    []string{"RS256"},
			UsernameClaim:  "email",
			UsernamePrefix: "-",
			GroupsPrefix:   "-",
			RequiredClaims: []string{},
		}, operation.ProvisioningParameters.Parameters.OIDC.List[0])
	})
	t.Run("Should accept update to empty OIDC list", func(t *testing.T) {
		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{ \"list\":[]}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)
		operation, err := st.Operations().GetProvisioningOperationByID(response.OperationData)

		// then
		require.NoError(t, err)
		assert.Len(t, operation.ProvisioningParameters.Parameters.OIDC.List, 0)
	})
}

func TestUpdateAdditionalWorkerNodePools(t *testing.T) {
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
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			instance := fixture.FixInstance(instanceID)
			instance.ServicePlanID = broker.AWSPlanID
			st := storage.NewMemoryStorage()
			err := st.Instances().Insert(instance)
			require.NoError(t, err)
			err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
			require.NoError(t, err)

			handler := &handler{}
			q := &automock.Queue{}
			q.On("Add", mock.AnythingOfType("string"))

			kcBuilder := &kcMock.KcBuilder{}
			kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.com", nil)

			svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
				fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

			// when
			_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
				ServiceID:       "",
				PlanID:          broker.AWSPlanID,
				RawParameters:   json.RawMessage("{\"additionalWorkerNodePools\":" + tc.additionalWorkerNodePools + "}"),
				PreviousValues:  domain.PreviousValues{},
				RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
				MaintenanceInfo: nil,
			}, true)

			// then
			assert.Equal(t, tc.expectedError, err != nil)
		})
	}
}

func TestHAZones(t *testing.T) {
	t.Run("should fail when attempting to disable HA zones for existing additional worker node pool", func(t *testing.T) {
		// given
		instance := fixture.FixInstance(instanceID)
		instance.ServicePlanID = broker.AWSPlanID
		instance.Parameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
			{
				Name:          "name-1",
				MachineType:   "m6i.large",
				HAZones:       true,
				AutoScalerMin: 3,
				AutoScalerMax: 20,
			},
		}
		st := storage.NewMemoryStorage()
		err := st.Instances().Insert(instance)
		require.NoError(t, err)
		err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
		require.NoError(t, err)

		handler := &handler{}
		q := &automock.Queue{}
		q.On("Add", mock.AnythingOfType("string"))

		kcBuilder := &kcMock.KcBuilder{}

		svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
			fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

		// when
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AWSPlanID,
			RawParameters:   json.RawMessage(`{"additionalWorkerNodePools": [{"name": "name-1", "machineType": "m6i.large", "haZones": false, "autoScalerMin": 3, "autoScalerMax": 20}]}`),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.EqualError(t, err, "HA zones setting is permanent and cannot be changed for name-1 additional worker node pool")
	})

	t.Run("should fail when attempting to enable HA zones for existing additional worker node pool", func(t *testing.T) {
		// given
		instance := fixture.FixInstance(instanceID)
		instance.ServicePlanID = broker.AWSPlanID
		instance.Parameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
			{
				Name:          "name-1",
				MachineType:   "m6i.large",
				HAZones:       false,
				AutoScalerMin: 3,
				AutoScalerMax: 20,
			},
		}
		st := storage.NewMemoryStorage()
		err := st.Instances().Insert(instance)
		require.NoError(t, err)
		err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
		require.NoError(t, err)

		handler := &handler{}
		q := &automock.Queue{}
		q.On("Add", mock.AnythingOfType("string"))

		kcBuilder := &kcMock.KcBuilder{}

		svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
			fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

		// when
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AWSPlanID,
			RawParameters:   json.RawMessage(`{"additionalWorkerNodePools": [{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]}`),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.EqualError(t, err, "HA zones setting is permanent and cannot be changed for name-1 additional worker node pool")
	})
}

func TestUpdateAdditionalWorkerNodePoolsForUnsupportedPlans(t *testing.T) {
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
			instance := fixture.FixInstance(instanceID)
			instance.ServicePlanID = tc.planID
			st := storage.NewMemoryStorage()
			err := st.Instances().Insert(instance)
			require.NoError(t, err)
			err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
			require.NoError(t, err)

			handler := &handler{}
			q := &automock.Queue{}
			q.On("Add", mock.AnythingOfType("string"))

			kcBuilder := &kcMock.KcBuilder{}

			svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
				fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

			additionalWorkerNodePools := `[{"name": "name-1", "machineType": "m6i.large", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`

			// when
			_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
				ServiceID:       "",
				PlanID:          tc.planID,
				RawParameters:   json.RawMessage("{\"additionalWorkerNodePools\":" + additionalWorkerNodePools + "}"),
				PreviousValues:  domain.PreviousValues{},
				RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
				MaintenanceInfo: nil,
			}, true)

			// then
			assert.EqualError(t, err, fmt.Sprintf("additional worker node pools are not supported for plan ID: %s", tc.planID))
		})
	}
}

func TestUpdateEndpoint_UpdateWithEnabledDashboard(t *testing.T) {
	// given
	instance := internal.Instance{
		InstanceID:    instanceID,
		ServicePlanID: broker.TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
			ErsContext: internal.ERSContext{
				TenantID:        "",
				SubAccountID:    "",
				GlobalAccountID: "",
				Active:          nil,
			},
		},
		DashboardURL: "https://console.cd6e47b.example.com",
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)
	// st.Operations().InsertDeprovisioningOperation(fixSuspensionOperation())
	// st.Operations().InsertProvisioningOperation(fixProvisioningOperation("02"))

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))

	kcBuilder := &kcMock.KcBuilder{}
	svc := broker.NewUpdate(broker.Config{AllowUpdateExpiredInstanceWithContext: true}, st, handler, true, false, true, q, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))
	createFakeCRs(t)
	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.TrialPlanID,
		RawParameters:   nil,
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"active\":false}"),
		MaintenanceInfo: nil,
	}, true)
	require.NoError(t, err)

	// then
	inst, err := st.Instances().GetByID(instanceID)
	require.NoError(t, err)

	// check if the instance is updated successfully
	assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, inst.DashboardURL)
	// check if the API response is correct
	assert.Regexp(t, `^https:\/\/dashboard\.example\.com\/\?kubeconfigID=`, response.DashboardURL)
}

func TestUpdateExpiredInstance(t *testing.T) {
	instance := internal.Instance{
		InstanceID:      instanceID,
		ServicePlanID:   broker.TrialPlanID,
		GlobalAccountID: "globalaccount_id_init",
		Parameters: internal.ProvisioningParameters{
			PlanID:     broker.TrialPlanID,
			ErsContext: internal.ERSContext{},
		},
	}
	expireTime := instance.CreatedAt.Add(time.Hour * 24 * 14)
	instance.ExpiredAt = &expireTime

	storage := storage.NewMemoryStorage()
	createFakeCRs(t)
	err := storage.Instances().Insert(instance)
	require.NoError(t, err)

	err = storage.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)

	kcBuilder := &kcMock.KcBuilder{}

	handler := &handler{}

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))
	svc := broker.NewUpdate(broker.Config{AllowUpdateExpiredInstanceWithContext: true}, storage, handler, true, false, true, queue, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

	t.Run("should accept if it is same as previous", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       broker.KymaServiceID,
			PlanID:          broker.TrialPlanID,
			RawParameters:   nil,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_init\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
	})

	t.Run("should accept change GA", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       broker.KymaServiceID,
			PlanID:          broker.TrialPlanID,
			RawParameters:   nil,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_new\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
	})

	t.Run("should accept change GA, with params", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       broker.KymaServiceID,
			PlanID:          broker.TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_new_2\", \"active\":true}"),
			RawParameters:   json.RawMessage(`{"autoScalerMin": 4, "autoScalerMax": 3}`),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
	})

	t.Run("should fail as not global account passed", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       broker.KymaServiceID,
			PlanID:          broker.TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"x\":\"y\", \"active\":true}"),
			RawParameters:   json.RawMessage(`{"autoScalerMin": 4, "autoScalerMax": 3}`),
			MaintenanceInfo: nil,
		}, true)
		require.Error(t, err)
	})
}

func TestSubaccountMovement(t *testing.T) {
	registerCRD()
	runtimeId := createFakeCRs(t)
	defer cleanFakeCRs(t, runtimeId)

	instance := internal.Instance{
		InstanceID:      instanceID,
		RuntimeID:       runtimeId,
		ServicePlanID:   broker.TrialPlanID,
		GlobalAccountID: "InitialGlobalAccountID",
		Parameters: internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
			ErsContext: internal.ERSContext{
				GlobalAccountID: "InitialGlobalAccountID",
			},
		},
	}

	storage := storage.NewMemoryStorage()
	err := storage.Instances().Insert(instance)
	require.NoError(t, err)

	err = storage.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.com", nil)

	handler := &handler{}

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	svc := broker.NewUpdate(broker.Config{SubaccountMovementEnabled: true}, storage, handler, true, true, true, queue, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

	t.Run("no move performed so subscription should be empty", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       broker.KymaServiceID,
			PlanID:          broker.TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"ChangedlGlobalAccountID\"}"),
			RawParameters:   json.RawMessage("{\"name\":\"test\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
		instance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, "InitialGlobalAccountID", instance.SubscriptionGlobalAccountID)
		assert.Equal(t, "ChangedlGlobalAccountID", instance.GlobalAccountID)
	})

	t.Run("move subaccount first time", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       broker.KymaServiceID,
			PlanID:          broker.TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"newGlobalAccountID-v1\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
		instance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, "InitialGlobalAccountID", instance.SubscriptionGlobalAccountID)
		assert.Equal(t, "newGlobalAccountID-v1", instance.GlobalAccountID)
	})

	t.Run("move subaccount second time", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       broker.KymaServiceID,
			PlanID:          broker.TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"newGlobalAccountID-v2\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
		instance, err := storage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, "InitialGlobalAccountID", instance.SubscriptionGlobalAccountID)
		assert.Equal(t, "newGlobalAccountID-v2", instance.GlobalAccountID)
	})
}

func TestLabelChangeWhenMovingSubaccount(t *testing.T) {
	const (
		oldGlobalAccountId = "first-global-account-id"
		newGlobalAccountId = "changed-global-account-id"
	)
	registerCRD()
	createCRDs(t)
	runtimeId := createFakeCRs(t)
	defer cleanFakeCRs(t, runtimeId)

	instance := internal.Instance{
		InstanceID:      instanceID,
		ServicePlanID:   broker.TrialPlanID,
		GlobalAccountID: oldGlobalAccountId,
		RuntimeID:       runtimeId,
		Parameters: internal.ProvisioningParameters{
			PlanID: broker.TrialPlanID,
			ErsContext: internal.ERSContext{
				GlobalAccountID: newGlobalAccountId,
			},
		},
	}

	storage := storage.NewMemoryStorage()
	err := storage.Instances().Insert(instance)
	require.NoError(t, err)

	err = storage.Operations().InsertProvisioningOperation(fixProvisioningOperation("01"))
	require.NoError(t, err)

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.com.dummy", nil)

	handler := &handler{}

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	svc := broker.NewUpdate(broker.Config{SubaccountMovementEnabled: true}, storage, handler, true, true, true, queue, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

	t.Run("simulate flow of moving account with labels on CRs", func(t *testing.T) {
		// initial state of instance - moving account was never donex
		i, e := storage.Instances().GetByID(instanceID)
		require.NoError(t, e)
		assert.Equal(t, oldGlobalAccountId, i.GlobalAccountID)
		assert.Empty(t, i.SubscriptionGlobalAccountID)
		assert.Equal(t, runtimeId, i.RuntimeID)

		// simulate moving account with new global account id - it means that we should update labels in CR
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       broker.KymaServiceID,
			PlanID:          broker.TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"changed-global-account-id\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)

		// after update instance should have new global account id and old global account id as subscription global account id, subsciprion global id is set only once.
		i, err = storage.Instances().GetByID(instanceID)
		require.NoError(t, err)
		assert.Equal(t, newGlobalAccountId, i.GlobalAccountID)
		assert.Equal(t, oldGlobalAccountId, i.SubscriptionGlobalAccountID)
		assert.Equal(t, runtimeId, i.RuntimeID)

		// all CRs should have new global account id as label
		gvk, err := customresources.GvkByName(customresources.KymaCr)
		require.NoError(t, err)
		cr := &unstructured.Unstructured{}
		cr.SetGroupVersionKind(gvk)
		err = fakeKcpK8sClient.Get(context.Background(), client.ObjectKey{Name: i.RuntimeID, Namespace: broker.KcpNamespace}, cr)
		require.NoError(t, err)
		labels := cr.GetLabels()
		assert.Len(t, labels, 1)
		assert.Equal(t, newGlobalAccountId, labels[customresources.GlobalAccountIdLabel])

		gvk, err = customresources.GvkByName(customresources.RuntimeCr)
		require.NoError(t, err)
		cr = &unstructured.Unstructured{}
		cr.SetGroupVersionKind(gvk)
		err = fakeKcpK8sClient.Get(context.Background(), client.ObjectKey{Name: i.RuntimeID, Namespace: broker.KcpNamespace}, cr)
		require.NoError(t, err)
		labels = cr.GetLabels()
		assert.Len(t, labels, 1)
		assert.Equal(t, newGlobalAccountId, labels[customresources.GlobalAccountIdLabel])

		gvk, err = customresources.GvkByName(customresources.GardenerClusterCr)
		require.NoError(t, err)
		cr = &unstructured.Unstructured{}
		cr.SetGroupVersionKind(gvk)
		err = fakeKcpK8sClient.Get(context.Background(), client.ObjectKey{Name: i.RuntimeID, Namespace: broker.KcpNamespace}, cr)
		require.NoError(t, err)
		labels = cr.GetLabels()
		assert.Len(t, labels, 1)
		assert.Equal(t, newGlobalAccountId, labels[customresources.GlobalAccountIdLabel])
	})
}

func TestUpdateUnsupportedMachine(t *testing.T) {
	// given
	instance := fixture.FixInstance(instanceID)
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))

	kcBuilder := &kcMock.KcBuilder{}
	svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, fixRegionsSupportingMachine(), imConfigFixture, newSchemaService(t))

	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.AzurePlanID,
		RawParameters:   json.RawMessage("{\"machineType\":" + "\"Standard_D16s_v5\"" + "}"),
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.EqualError(t, err, "In the region westeurope, the machine type Standard_D16s_v5 is not available, it is supported in the brazilsouth, uksouth")
}

func TestUpdateUnsupportedMachineInAdditionalWorkerNodePools(t *testing.T) {
	// given
	instance := fixture.FixInstance(instanceID)
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))

	kcBuilder := &kcMock.KcBuilder{}
	svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, fixRegionsSupportingMachine(), imConfigFixture, newSchemaService(t))

	testCases := []struct {
		name                      string
		additionalWorkerNodePools string
		expectedError             string
	}{
		{
			name:                      "Single unsupported machine type",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_D8s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the region westeurope, the following machine types are not available: Standard_D8s_v5 (used in: name-1), it is supported in the brazilsouth, uksouth",
		},
		{
			name:                      "Multiple unsupported machine types",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_D8s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "Standard_D16s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the region westeurope, the following machine types are not available: Standard_D8s_v5 (used in: name-1), it is supported in the brazilsouth, uksouth; Standard_D16s_v5 (used in: name-2), it is supported in the brazilsouth, uksouth",
		},
		{
			name:                      "Duplicate unsupported machine type",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_D8s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "Standard_D8s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the region westeurope, the following machine types are not available: Standard_D8s_v5 (used in: name-1, name-2), it is supported in the brazilsouth, uksouth",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
				ServiceID:       "",
				PlanID:          broker.AzurePlanID,
				RawParameters:   json.RawMessage("{\"additionalWorkerNodePools\":" + tc.additionalWorkerNodePools + "}"),
				PreviousValues:  domain.PreviousValues{},
				RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
				MaintenanceInfo: nil,
			}, true)

			// then
			assert.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestUpdateGPUMachineForInternalUser(t *testing.T) {
	// given
	instance := fixture.FixInstance(instanceID)
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.dummy", nil)
	svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

	additionalWorkerNodePools := `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`
	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.AzurePlanID,
		RawParameters:   json.RawMessage("{\"machineType\":\"Standard_D16s_v5\", \"additionalWorkerNodePools\": " + additionalWorkerNodePools + "}"),
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.NoError(t, err)
}

func TestUpdateGPUMachineForExternalCustomer(t *testing.T) {
	for tn, tc := range map[string]struct {
		planID                    string
		additionalWorkerNodePools string
		expectedError             string
	}{
		"Single AWS G6 GPU machine type": {
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g6.xlarge (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		"Multiple AWS G6 GPU machine types": {
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g6.2xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g6.xlarge (used in worker node pools: name-1), g6.2xlarge (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Duplicate AWS G6 GPU machine type": {
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g6.xlarge (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Single AWS G4dn GPU machine type": {
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g4dn.xlarge (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		"Multiple AWS G4dn GPU machine types": {
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g4dn.2xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g4dn.xlarge (used in worker node pools: name-1), g4dn.2xlarge (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Duplicate AWS G4dn GPU machine type": {
			planID:                    broker.AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g4dn.xlarge (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Single GCP GPU machine type": {
			planID:                    broker.GCPPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g2-standard-4 (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		"Multiple GCP GPU machine types": {
			planID:                    broker.GCPPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g2-standard-8", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g2-standard-4 (used in worker node pools: name-1), g2-standard-8 (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Duplicate GCP GPU machine type": {
			planID:                    broker.GCPPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g2-standard-4 (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Single Azure GPU machine type": {
			planID:                    broker.AzurePlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: Standard_NC4as_T4_v3 (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		"Multiple Azure GPU machine types": {
			planID:                    broker.AzurePlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "Standard_NC8as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: Standard_NC4as_T4_v3 (used in worker node pools: name-1), Standard_NC8as_T4_v3 (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Duplicate Azure GPU machine type": {
			planID:                    broker.AzurePlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: Standard_NC4as_T4_v3 (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
	} {
		t.Run(tn, func(t *testing.T) {
			// given
			instance := fixture.FixInstance(instanceID)
			instance.ServicePlanID = tc.planID
			st := storage.NewMemoryStorage()
			err := st.Instances().Insert(instance)
			require.NoError(t, err)
			err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
			require.NoError(t, err)

			handler := &handler{}
			q := &automock.Queue{}
			q.On("Add", mock.AnythingOfType("string"))
			kcBuilder := &kcMock.KcBuilder{}

			svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
				fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

			// when
			_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
				ServiceID:       "",
				PlanID:          tc.planID,
				RawParameters:   json.RawMessage("{\"additionalWorkerNodePools\":" + tc.additionalWorkerNodePools + "}"),
				PreviousValues:  domain.PreviousValues{},
				RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true, \"license_type\": \"CUSTOMER\"}"),
				MaintenanceInfo: nil,
			}, true)

			// then
			assert.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestAvailableZonesValidationDuringUpdate(t *testing.T) {
	// given
	instance := fixture.FixInstance(instanceID)
	instance.ServicePlanID = broker.AWSPlanID
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))
	kcBuilder := &kcMock.KcBuilder{}

	imConfig := broker.InfrastructureManager{
		UseSmallerMachineTypes: false,
		EnableIngressFiltering: false,
	}

	svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, fixRegionsSupportingMachine(), imConfig, newSchemaService(t))

	additionalWorkerNodePools := `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`

	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.AWSPlanID,
		RawParameters:   json.RawMessage("{\"additionalWorkerNodePools\":" + additionalWorkerNodePools + "}"),
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.EqualError(t, err, "In the westeurope, the g6.xlarge machine type is not available in 3 zones. If you want to use this machine type, set HA to false.")
}

func TestMachineTypeUpdateInAdditionalWorkerNodePools(t *testing.T) {
	// given
	instance := fixture.FixInstance(instanceID)
	instance.Parameters.Parameters.AdditionalWorkerNodePools = []pkg.AdditionalWorkerNodePool{
		{
			Name:          "name-1",
			MachineType:   "Standard_NC8as_T4_v3",
			HAZones:       true,
			AutoScalerMin: 3,
			AutoScalerMax: 20,
		},
	}
	st := storage.NewMemoryStorage()
	err := st.Instances().Insert(instance)
	require.NoError(t, err)
	err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
	require.NoError(t, err)

	handler := &handler{}
	q := &automock.Queue{}
	q.On("Add", mock.AnythingOfType("string"))

	imConfig := broker.InfrastructureManager{
		UseSmallerMachineTypes: false,
		EnableIngressFiltering: false,
	}

	kcBuilder := &kcMock.KcBuilder{}
	kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.dummy", nil)
	svc := broker.NewUpdate(broker.Config{}, st, handler, true, true, false, q, broker.PlansConfig{},
		fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfig, newSchemaService(t))

	additionalWorkerNodePools := `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`
	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          broker.AzurePlanID,
		RawParameters:   json.RawMessage("{\"machineType\":\"Standard_D16s_v5\", \"additionalWorkerNodePools\": " + additionalWorkerNodePools + "}"),
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.EqualError(t, err, "Machine type setting is permanent, and you cannot change it for the name-1 additional worker node pool")
}

func TestUpdateAdditionalProperties(t *testing.T) {
	t.Run("file should contain request with additional properties", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		expectedFile := filepath.Join(tempDir, additionalproperties.UpdateRequestsFileName)
		instance := fixture.FixInstance(instanceID)
		st := storage.NewMemoryStorage()
		err := st.Instances().Insert(instance)
		require.NoError(t, err)
		err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
		require.NoError(t, err)

		handler := &handler{}
		q := &automock.Queue{}
		q.On("Add", mock.AnythingOfType("string"))

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.dummy", nil)
		svc := broker.NewUpdate(broker.Config{MonitorAdditionalProperties: true, AdditionalPropertiesPath: tempDir}, st, handler, true, true, false, q, broker.PlansConfig{},
			fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

		// when
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"machineType\":\"Standard_D16s_v5\",\"test\":\"test\"}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"subaccount_id\":\"subaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.NoError(t, err)

		contents, err := os.ReadFile(expectedFile)
		assert.NoError(t, err)

		lines := bytes.Split(contents, []byte("\n"))
		assert.Greater(t, len(lines), 0)
		var entry map[string]interface{}
		err = json.Unmarshal(lines[0], &entry)
		assert.NoError(t, err)

		assert.Equal(t, "globalaccount_id_1", entry["globalAccountID"])
		assert.Equal(t, "subaccount_id_1", entry["subAccountID"])
		assert.Equal(t, instanceID, entry["instanceID"])
	})

	t.Run("file should contain two requests with additional properties", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		expectedFile := filepath.Join(tempDir, additionalproperties.UpdateRequestsFileName)
		instance := fixture.FixInstance(instanceID)
		st := storage.NewMemoryStorage()
		err := st.Instances().Insert(instance)
		require.NoError(t, err)
		err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
		require.NoError(t, err)

		handler := &handler{}
		q := &automock.Queue{}
		q.On("Add", mock.AnythingOfType("string"))

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.dummy", nil)
		svc := broker.NewUpdate(broker.Config{MonitorAdditionalProperties: true, AdditionalPropertiesPath: tempDir}, st, handler, true, true, false, q, broker.PlansConfig{},
			fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

		// when
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"machineType\":\"Standard_D16s_v5\",\"test\":\"test\"}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"subaccount_id\":\"subaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)
		assert.NoError(t, err)

		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"machineType\":\"Standard_D16s_v5\",\"test\":\"test\"}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"subaccount_id\":\"subaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)
		assert.NoError(t, err)

		// then
		contents, err := os.ReadFile(expectedFile)
		assert.NoError(t, err)
		lines := bytes.Split(contents, []byte("\n"))
		assert.Equal(t, len(lines), 3)
		var entry map[string]interface{}

		err = json.Unmarshal(lines[0], &entry)
		assert.NoError(t, err)
		assert.Equal(t, "globalaccount_id_1", entry["globalAccountID"])
		assert.Equal(t, "subaccount_id_1", entry["subAccountID"])
		assert.Equal(t, instanceID, entry["instanceID"])

		err = json.Unmarshal(lines[1], &entry)
		assert.NoError(t, err)
		assert.Equal(t, "globalaccount_id_1", entry["globalAccountID"])
		assert.Equal(t, "subaccount_id_1", entry["subAccountID"])
		assert.Equal(t, instanceID, entry["instanceID"])
	})

	t.Run("file should not contain request without additional properties", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		expectedFile := filepath.Join(tempDir, additionalproperties.UpdateRequestsFileName)
		instance := fixture.FixInstance(instanceID)
		st := storage.NewMemoryStorage()
		err := st.Instances().Insert(instance)
		require.NoError(t, err)
		err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
		require.NoError(t, err)

		handler := &handler{}
		q := &automock.Queue{}
		q.On("Add", mock.AnythingOfType("string"))

		kcBuilder := &kcMock.KcBuilder{}
		kcBuilder.On("GetServerURL", mock.Anything).Return("https://kcp.example.dummy", nil)
		svc := broker.NewUpdate(broker.Config{MonitorAdditionalProperties: true, AdditionalPropertiesPath: tempDir}, st, handler, true, true, false, q, broker.PlansConfig{},
			fixValueProvider(t), fixLogger(), dashboardConfig, kcBuilder, fakeKcpK8sClient, regionssupportingmachine.RegionsSupportingMachine{}, imConfigFixture, newSchemaService(t))

		// when
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          broker.AzurePlanID,
			RawParameters:   json.RawMessage("{\"machineType\":\"Standard_D16s_v5\"}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"subaccount_id\":\"subaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		assert.NoError(t, err)

		contents, err := os.ReadFile(expectedFile)
		assert.Nil(t, contents)
		assert.Error(t, err)
	})
}

func fixValueProvider(t *testing.T) broker.ValuesProvider {
	return provider.NewPlanSpecificValuesProvider(
		broker.InfrastructureManager{
			DefaultGardenerShootPurpose:  "production",
			DefaultTrialProvider:         pkg.AWS,
			MultiZoneCluster:             true,
			ControlPlaneFailureTolerance: "",
			UseSmallerMachineTypes:       true,
		}, nil,
		newSchemaService(t))
}

func registerCRD() {
	var customResourceDefinition apiextensionsv1.CustomResourceDefinition
	customResourceDefinition.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	})
	fakeKcpK8sClient.Scheme().AddKnownTypeWithName(customResourceDefinition.GroupVersionKind(), &customResourceDefinition)
}

func createCRDs(t *testing.T) {
	createCustomResource := func(gvkName string) {
		var customResourceDefinition apiextensionsv1.CustomResourceDefinition
		gvk, err := customresources.GvkByName(gvkName)
		require.NoError(t, err)
		crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
		customResourceDefinition.SetName(crdName)
		err = fakeKcpK8sClient.Create(context.Background(), &customResourceDefinition)
		require.NoError(t, err)
	}
	createCustomResource(customresources.KymaCr)
	createCustomResource(customresources.GardenerClusterCr)
	createCustomResource(customresources.RuntimeCr)
}

func createFakeCRs(t *testing.T) string {
	runtimeID := uuid.New().String()
	createCustomResource := func(t *testing.T, runtimeID string, crName string) {
		assert.NotNil(t, fakeKcpK8sClient)
		gvk, err := customresources.GvkByName(crName)
		require.NoError(t, err)
		us := unstructured.Unstructured{}
		us.SetGroupVersionKind(gvk)
		us.SetName(runtimeID)
		us.SetNamespace(broker.KcpNamespace)
		err = fakeKcpK8sClient.Create(context.Background(), &us)
		require.NoError(t, err)
	}

	createCustomResource(t, runtimeID, customresources.KymaCr)
	createCustomResource(t, runtimeID, customresources.GardenerClusterCr)
	createCustomResource(t, runtimeID, customresources.RuntimeCr)

	return runtimeID
}

func cleanFakeCRs(t *testing.T, runtimeID string) {
	createCustomResource := func(t *testing.T, id string, crName string) {
		assert.NotNil(t, fakeKcpK8sClient)
		gvk, err := customresources.GvkByName(crName)
		require.NoError(t, err)
		us := unstructured.Unstructured{}
		us.SetGroupVersionKind(gvk)
		us.SetName(runtimeID)
		us.SetNamespace(broker.KcpNamespace)
		err = fakeKcpK8sClient.Delete(context.Background(), &us)
		require.NoError(t, err)
	}

	createCustomResource(t, runtimeID, customresources.KymaCr)
	createCustomResource(t, runtimeID, customresources.GardenerClusterCr)
	createCustomResource(t, runtimeID, customresources.RuntimeCr)
}
