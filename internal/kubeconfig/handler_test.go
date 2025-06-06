package kubeconfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig/automock"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	instanceID        = "93241a34-8ab5-4f10-978e-eaa6f8ad551c"
	operationID       = "306f2406-e972-4fae-8edd-50fc50e56817"
	instanceRuntimeID = "e04813ba-244a-4150-8670-506c37959388"
	ownClusterPlanID  = "03e3cb66-a4c6-4c6a-b4b0-5d42224debea"
)

func TestHandler_GetKubeconfig(t *testing.T) {
	cases := map[string]struct {
		pass                 bool
		missingSecret        bool
		instanceID           string
		runtimeID            string
		operationStatus      domain.LastOperationState
		expectedStatusCode   int
		expectedErrorMessage string
	}{
		"new kubeconfig was returned": {
			pass:               true,
			instanceID:         instanceID,
			runtimeID:          instanceRuntimeID,
			expectedStatusCode: http.StatusOK,
		},
		"instance ID is empty": {
			pass:                 false,
			instanceID:           "",
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: "instanceID is required",
		},
		"runtimeID not exist": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            "",
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: fmt.Sprintf("kubeconfig for instance %s does not exist. Provisioning could be in progress, please try again later", instanceID),
		},
		"provisioning operation is not ready": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            instanceRuntimeID,
			operationStatus:      domain.InProgress,
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: fmt.Sprintf("provisioning operation for instance %s is in progress state, kubeconfig not exist yet, please try again later", instanceID),
		},
		"unsuspension operation is not ready": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            instanceRuntimeID,
			operationStatus:      internal.OperationStatePending,
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: fmt.Sprintf("provisioning operation for instance %s is in progress state, kubeconfig not exist yet, please try again later", instanceID),
		},
		"provisioning operation failed": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            instanceRuntimeID,
			operationStatus:      domain.Failed,
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: fmt.Sprintf("provisioning operation for instance %s failed, kubeconfig does not exist", instanceID),
		},
		"kubeconfig builder failed": {
			pass:                 false,
			instanceID:           instanceID,
			runtimeID:            instanceRuntimeID,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedErrorMessage: "cannot fetch SKR kubeconfig: builder error",
		},
		"kubeconfig secret is missing": {
			pass:                 false,
			missingSecret:        true,
			instanceID:           instanceID,
			runtimeID:            instanceRuntimeID,
			expectedStatusCode:   http.StatusNotFound,
			expectedErrorMessage: "kubeconfig does not exist",
		},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			// given
			instance := internal.Instance{
				InstanceID: d.instanceID,
				RuntimeID:  d.runtimeID,
			}

			operation := internal.ProvisioningOperation{
				Operation: internal.Operation{
					ID:         operationID,
					InstanceID: instance.InstanceID,
					State:      d.operationStatus,
					Type:       internal.OperationTypeProvision,
				},
			}

			db := storage.NewMemoryStorage()
			err := db.Instances().Insert(instance)
			require.NoError(t, err)
			err = db.Operations().InsertProvisioningOperation(operation)
			require.NoError(t, err)

			builder := &automock.KcBuilder{}
			if d.pass {
				builder.On("Build", &instance).Return("--kubeconfig file", nil)
				defer builder.AssertExpectations(t)
			} else if d.missingSecret {
				builder.On("Build", &instance).Return("", NewNotFoundError("secret is missing"))
			} else {
				builder.On("Build", &instance).Return("", fmt.Errorf("builder error"))
			}

			log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))

			router := httputil.NewRouter()

			handler := NewHandler(db, builder, "", ownClusterPlanID, log)
			handler.AttachRoutes(router)

			server := httptest.NewServer(router)

			// when
			response, err := http.Get(fmt.Sprintf("%s/kubeconfig/%s", server.URL, d.instanceID))
			require.NoError(t, err)

			// then
			require.Equal(t, d.expectedStatusCode, response.StatusCode)

			if d.pass {
				require.Equal(t, "application/x-yaml", response.Header.Get("Content-Type"))
			} else {
				require.Equal(t, "application/json", response.Header.Get("Content-Type"))
			}

			body, err := ioutil.ReadAll(response.Body)
			require.NoError(t, err)

			if d.pass {
				require.Equal(t, "--kubeconfig file", string(body))
			} else {
				var errorResponse ErrorResponse
				err := json.Unmarshal(body, &errorResponse)
				require.NoError(t, err)
				require.Equal(t, d.expectedErrorMessage, errorResponse.Error)
			}
		})
	}
}

func TestHandler_GetKubeconfigForOwnCluster(t *testing.T) {
	// given
	instance := internal.Instance{
		Parameters: internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{
				Kubeconfig: "custom-kubeconfig",
			},
		},
		InstanceDetails: internal.InstanceDetails{
			Kubeconfig: "custom-kubeconfig",
		},
		InstanceID:    instanceID,
		RuntimeID:     runtimeID,
		ServicePlanID: ownClusterPlanID,
	}

	operation := internal.ProvisioningOperation{
		Operation: internal.Operation{
			ID:         operationID,
			InstanceID: instance.InstanceID,
			State:      domain.Succeeded,
			InstanceDetails: internal.InstanceDetails{
				Kubeconfig: "custom-kubeconfig",
			},
			Type: internal.OperationTypeProvision,
		},
	}

	db := storage.NewMemoryStorage()
	err := db.Instances().Insert(instance)
	require.NoError(t, err)
	err = db.Operations().InsertProvisioningOperation(operation)
	require.NoError(t, err)

	// we do not expect usage of KcBuilder
	builder := &automock.KcBuilder{}
	defer builder.AssertExpectations(t)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	router := httputil.NewRouter()

	handler := NewHandler(db, builder, "", ownClusterPlanID, log)
	handler.AttachRoutes(router)

	server := httptest.NewServer(router)

	// when
	response, err := http.Get(fmt.Sprintf("%s/kubeconfig/%s", server.URL, instanceID))
	require.NoError(t, err)

	// then
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestHandler_specifyAllowOriginHeader(t *testing.T) {
	cases := map[string]struct {
		requestHeader      http.Header
		origins            string
		corsHeaderExist    bool
		corsExpectedHeader string
	}{
		"one origin which exist": {
			requestHeader:      map[string][]string{"Origin": {"https://example.com"}},
			origins:            "https://example.com",
			corsHeaderExist:    true,
			corsExpectedHeader: "https://example.com",
		},
		"many origins which one exist": {
			requestHeader:      map[string][]string{"Origin": {"https://example.com"}},
			origins:            "https://acme.com,https://example.com,https://eggplant.io",
			corsHeaderExist:    true,
			corsExpectedHeader: "https://example.com",
		},
		"many origins non one exist": {
			requestHeader:   map[string][]string{"Origin": {"https://example.com"}},
			origins:         "https://acme.com,https://gopher.com,https://eggplant.io",
			corsHeaderExist: false,
		},
		"accept all origins": {
			requestHeader:      map[string][]string{"Origin": {"https://example.com"}},
			origins:            "*",
			corsHeaderExist:    true,
			corsExpectedHeader: "*",
		},
		"no origin header in request": {
			requestHeader:   map[string][]string{},
			origins:         "https://acme.com,https://example.com,https://eggplant.io",
			corsHeaderExist: false,
		},
		"wrong origin configuration": {
			requestHeader:   map[string][]string{"Origin": {"https://example.com"}},
			origins:         "https://acme.com;https://example.com;https://eggplant.io",
			corsHeaderExist: false,
		},
	}

	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			// given
			request := &http.Request{Header: d.requestHeader}
			response := &httptest.ResponseRecorder{}

			handler := NewHandler(storage.NewMemoryStorage(), nil, d.origins, ownClusterPlanID, nil)

			// when
			handler.specifyAllowOriginHeader(request, response)

			// then
			if d.corsHeaderExist {
				t.Log(response.Header())
				require.Equal(t, d.corsExpectedHeader, response.Header().Get("Access-Control-Allow-Origin"))
			} else {
				require.NotContains(t, "Access-Control-Allow-Origin", response.Header())
			}
		})
	}
}
