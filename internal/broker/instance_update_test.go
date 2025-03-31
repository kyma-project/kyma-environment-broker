package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

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

	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
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
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(
		Config{},
		st.Instances(),
		st.RuntimeStates(),
		st.Operations(),
		handler,
		true,
		false,
		true,
		q,
		PlansConfig{},
		planDefaults,
		fixLogger(),
		dashboardConfig,
		kcBuilder,
		&OneForAllConvergedCloudRegionsProvider{},
		fakeKcpK8sClient,
		nil)

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
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
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
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
		ServicePlanID: AWSPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: AWSPlanID,
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	t.Run("Should fail on invalid (too low) autoScalerMin and autoScalerMax", func(t *testing.T) {

		// when
		response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AWSPlanID,
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
			PlanID:          AWSPlanID,
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
			PlanID:          AWSPlanID,
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
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
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
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	// when
	_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
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
		ServicePlanID:   TrialPlanID,
		GlobalAccountID: "origin-account-id",
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}

	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
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

func TestUpdateEndpoint_UpdateParameters(t *testing.T) {
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	t.Run("Should fail on invalid OIDC params", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"{clientID}","groupsClaim":"groups","issuerURL":"{issuerURL}","signingAlgs":["RS256"],"usernameClaim":"email","usernamePrefix":"-"`
		errMsg := fmt.Errorf("issuerURL must be a valid URL, issuerURL must have https scheme")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on invalid OIDC issuerURL param", func(t *testing.T) {
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
				expectedError: "issuerURL must be a valid URL",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				errMsg := fmt.Errorf(tc.expectedError)
				expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

				// when
				_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
					ServiceID:       "",
					PlanID:          AzurePlanID,
					RawParameters:   json.RawMessage("{\"oidc\":{" + tc.oidcParams + "}}"),
					PreviousValues:  domain.PreviousValues{},
					RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
					MaintenanceInfo: nil,
				}, true)

				// then
				require.Error(t, err)
				assert.IsType(t, &apiresponses.FailureResponse{}, err)
				apierr := err.(*apiresponses.FailureResponse)
				assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
				assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).LoggerAction(), apierr.LoggerAction())
			})
		}
	})

	t.Run("Should fail on insufficient OIDC params (missing clientID)", func(t *testing.T) {
		// given
		oidcParams := `"issuerURL":"https://test.local"`
		errMsg := fmt.Errorf("clientID must not be empty")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).LoggerAction(), apierr.LoggerAction())
	})

	t.Run("Should fail on invalid OIDC signingAlgs param", func(t *testing.T) {
		// given
		oidcParams := `"clientID":"client-id","issuerURL":"https://test.local","signingAlgs":["RS256","notValid"]`
		errMsg := fmt.Errorf("signingAlgs must contain valid signing algorithm(s)")
		expectedErr := apiresponses.NewFailureResponse(errMsg, http.StatusUnprocessableEntity, errMsg.Error())

		// when
		_, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AzurePlanID,
			RawParameters:   json.RawMessage("{\"oidc\":{" + oidcParams + "}}"),
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
			MaintenanceInfo: nil,
		}, true)

		// then
		require.Error(t, err)
		assert.IsType(t, &apiresponses.FailureResponse{}, err)
		apierr := err.(*apiresponses.FailureResponse)
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).ValidatedStatusCode(nil), apierr.ValidatedStatusCode(nil))
		assert.Equal(t, expectedErr.(*apiresponses.FailureResponse).LoggerAction(), apierr.LoggerAction())
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
			instance.ServicePlanID = AWSPlanID
			st := storage.NewMemoryStorage()
			err := st.Instances().Insert(instance)
			require.NoError(t, err)
			err = st.Operations().InsertProvisioningOperation(fixProvisioningOperation("provisioning01"))
			require.NoError(t, err)

			handler := &handler{}
			q := &automock.Queue{}
			q.On("Add", mock.AnythingOfType("string"))
			planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
				return &gqlschema.ClusterConfigInput{}, nil
			}
			kcBuilder := &kcMock.KcBuilder{}
			svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
				planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

			// when
			_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
				ServiceID:       "",
				PlanID:          AWSPlanID,
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
		instance.ServicePlanID = AWSPlanID
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
		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
			planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

		// when
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AWSPlanID,
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
		instance.ServicePlanID = AWSPlanID
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
		planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
			return &gqlschema.ClusterConfigInput{}, nil
		}
		kcBuilder := &kcMock.KcBuilder{}
		svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
			planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

		// when
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       "",
			PlanID:          AWSPlanID,
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
			planID: TrialPlanID,
		},
		"Free": {
			planID: FreemiumPlanID,
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
			planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
				return &gqlschema.ClusterConfigInput{}, nil
			}
			kcBuilder := &kcMock.KcBuilder{}
			svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
				planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

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
		ServicePlanID: TrialPlanID,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{AllowUpdateExpiredInstanceWithContext: true}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, false, true, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)
	createFakeCRs(t)
	// when
	response, err := svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          TrialPlanID,
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
		ServicePlanID:   TrialPlanID,
		GlobalAccountID: "globalaccount_id_init",
		Parameters: internal.ProvisioningParameters{
			PlanID:     TrialPlanID,
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

	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}

	svc := NewUpdate(Config{AllowUpdateExpiredInstanceWithContext: true}, storage.Instances(), storage.RuntimeStates(), storage.Operations(), handler, true, false, true, queue, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	t.Run("should accept if it is same as previous", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			RawParameters:   nil,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_init\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
	})

	t.Run("should accept change GA", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			RawParameters:   nil,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_new\"}"),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
	})

	t.Run("should accept change GA, with params", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
			PreviousValues:  domain.PreviousValues{},
			RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_new_2\", \"active\":true}"),
			RawParameters:   json.RawMessage(`{"autoScalerMin": 4, "autoScalerMax": 3}`),
			MaintenanceInfo: nil,
		}, true)
		require.NoError(t, err)
	})

	t.Run("should fail as not global account passed", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
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
		ServicePlanID:   TrialPlanID,
		GlobalAccountID: "InitialGlobalAccountID",
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
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

	handler := &handler{}

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}

	svc := NewUpdate(Config{SubaccountMovementEnabled: true}, storage.Instances(), storage.RuntimeStates(), storage.Operations(), handler, true, true, true, queue, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	t.Run("no move performed so subscription should be empty", func(t *testing.T) {
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
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
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
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
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
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
		ServicePlanID:   TrialPlanID,
		GlobalAccountID: oldGlobalAccountId,
		RuntimeID:       runtimeId,
		Parameters: internal.ProvisioningParameters{
			PlanID: TrialPlanID,
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

	handler := &handler{}

	queue := &automock.Queue{}
	queue.On("Add", mock.AnythingOfType("string"))

	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}

	svc := NewUpdate(Config{SubaccountMovementEnabled: true}, storage.Instances(), storage.RuntimeStates(), storage.Operations(), handler, true, true, true, queue, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	t.Run("simulate flow of moving account with labels on CRs", func(t *testing.T) {
		// initial state of instance - moving account was never donex
		i, e := storage.Instances().GetByID(instanceID)
		require.NoError(t, e)
		assert.Equal(t, oldGlobalAccountId, i.GlobalAccountID)
		assert.Empty(t, i.SubscriptionGlobalAccountID)
		assert.Equal(t, runtimeId, i.RuntimeID)

		// simulate moving account with new global account id - it means that we should update labels in CR
		_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
			ServiceID:       KymaServiceID,
			PlanID:          TrialPlanID,
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
		err = fakeKcpK8sClient.Get(context.Background(), client.ObjectKey{Name: i.RuntimeID, Namespace: KcpNamespace}, cr)
		require.NoError(t, err)
		labels := cr.GetLabels()
		assert.Len(t, labels, 1)
		assert.Equal(t, newGlobalAccountId, labels[customresources.GlobalAccountIdLabel])

		gvk, err = customresources.GvkByName(customresources.RuntimeCr)
		require.NoError(t, err)
		cr = &unstructured.Unstructured{}
		cr.SetGroupVersionKind(gvk)
		err = fakeKcpK8sClient.Get(context.Background(), client.ObjectKey{Name: i.RuntimeID, Namespace: KcpNamespace}, cr)
		require.NoError(t, err)
		labels = cr.GetLabels()
		assert.Len(t, labels, 1)
		assert.Equal(t, newGlobalAccountId, labels[customresources.GlobalAccountIdLabel])

		gvk, err = customresources.GvkByName(customresources.GardenerClusterCr)
		require.NoError(t, err)
		cr = &unstructured.Unstructured{}
		cr.SetGroupVersionKind(gvk)
		err = fakeKcpK8sClient.Get(context.Background(), client.ObjectKey{Name: i.RuntimeID, Namespace: KcpNamespace}, cr)
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, fixRegionsSupportingMachine())

	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          AzurePlanID,
		RawParameters:   json.RawMessage("{\"machineType\":" + "\"Standard_D16s_v5\"" + "}"),
		PreviousValues:  domain.PreviousValues{},
		RawContext:      json.RawMessage("{\"globalaccount_id\":\"globalaccount_id_1\", \"active\":true}"),
		MaintenanceInfo: nil,
	}, true)

	// then
	assert.EqualError(t, err, "In the region westeurope, the machine type Standard_D16s_v5 is not available, it is supported in the uksouth, brazilsouth")
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, fixRegionsSupportingMachine())

	testCases := []struct {
		name                      string
		additionalWorkerNodePools string
		expectedError             string
	}{
		{
			name:                      "Single unsupported machine type",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_D8s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the region westeurope, the following machine types are not available: Standard_D8s_v5 (used in: name-1), it is supported in the uksouth, brazilsouth",
		},
		{
			name:                      "Multiple unsupported machine types",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_D8s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "Standard_D16s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the region westeurope, the following machine types are not available: Standard_D8s_v5 (used in: name-1), it is supported in the uksouth, brazilsouth; Standard_D16s_v5 (used in: name-2), it is supported in the uksouth, brazilsouth",
		},
		{
			name:                      "Duplicate unsupported machine type",
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_D8s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "Standard_D8s_v5", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "In the region westeurope, the following machine types are not available: Standard_D8s_v5 (used in: name-1, name-2), it is supported in the uksouth, brazilsouth",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
				ServiceID:       "",
				PlanID:          AzurePlanID,
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
	planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
		return &gqlschema.ClusterConfigInput{}, nil
	}
	kcBuilder := &kcMock.KcBuilder{}
	svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
		planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

	additionalWorkerNodePools := `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`
	// when
	_, err = svc.Update(context.Background(), instanceID, domain.UpdateDetails{
		ServiceID:       "",
		PlanID:          AzurePlanID,
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
			planID:                    AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g6.xlarge (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		"Multiple AWS G6 GPU machine types": {
			planID:                    AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g6.2xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g6.xlarge (used in worker node pools: name-1), g6.2xlarge (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Duplicate AWS G6 GPU machine type": {
			planID:                    AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g6.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g6.xlarge (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Single AWS G4dn GPU machine type": {
			planID:                    AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g4dn.xlarge (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		"Multiple AWS G4dn GPU machine types": {
			planID:                    AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g4dn.2xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g4dn.xlarge (used in worker node pools: name-1), g4dn.2xlarge (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Duplicate AWS G4dn GPU machine type": {
			planID:                    AWSPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g4dn.xlarge", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g4dn.xlarge (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Single GCP GPU machine type": {
			planID:                    GCPPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g2-standard-4 (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		"Multiple GCP GPU machine types": {
			planID:                    GCPPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g2-standard-8", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g2-standard-4 (used in worker node pools: name-1), g2-standard-8 (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Duplicate GCP GPU machine type": {
			planID:                    GCPPlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "g2-standard-4", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: g2-standard-4 (used in worker node pools: name-1, name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Single Azure GPU machine type": {
			planID:                    AzurePlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: Standard_NC4as_T4_v3 (used in worker node pools: name-1) are not available for your account. For details, please contact your sales representative.",
		},
		"Multiple Azure GPU machine types": {
			planID:                    AzurePlanID,
			additionalWorkerNodePools: `[{"name": "name-1", "machineType": "Standard_NC4as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}, {"name": "name-2", "machineType": "Standard_NC8as_T4_v3", "haZones": true, "autoScalerMin": 3, "autoScalerMax": 20}]`,
			expectedError:             "The following GPU machine types: Standard_NC4as_T4_v3 (used in worker node pools: name-1), Standard_NC8as_T4_v3 (used in worker node pools: name-2) are not available for your account. For details, please contact your sales representative.",
		},
		"Duplicate Azure GPU machine type": {
			planID:                    AzurePlanID,
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
			planDefaults := func(planID string, platformProvider pkg.CloudProvider, provider *pkg.CloudProvider) (*gqlschema.ClusterConfigInput, error) {
				return &gqlschema.ClusterConfigInput{}, nil
			}
			kcBuilder := &kcMock.KcBuilder{}
			svc := NewUpdate(Config{}, st.Instances(), st.RuntimeStates(), st.Operations(), handler, true, true, false, q, PlansConfig{},
				planDefaults, fixLogger(), dashboardConfig, kcBuilder, &OneForAllConvergedCloudRegionsProvider{}, fakeKcpK8sClient, nil)

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
		us.SetNamespace(KcpNamespace)
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
		us.SetNamespace(KcpNamespace)
		err = fakeKcpK8sClient.Delete(context.Background(), &us)
		require.NoError(t, err)
	}

	createCustomResource(t, runtimeID, customresources.KymaCr)
	createCustomResource(t, runtimeID, customresources.GardenerClusterCr)
	createCustomResource(t, runtimeID, customresources.RuntimeCr)
}

func fixRegionsSupportingMachine() map[string][]string {
	return map[string][]string{
		"Standard_D": {"uksouth", "brazilsouth"},
	}
}
