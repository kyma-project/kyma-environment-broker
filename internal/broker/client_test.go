package broker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/stretchr/testify/assert"
)

const (
	fixInstanceID      = "72b83910-ac12-4dcb-b91d-960cca2b36abx"
	fixTrialInstanceID = "46955f0b-9d81-4eb0-935a-52013c96f7bf"
	fixRuntimeID       = "24da44ea-0295-4b1c-b5c1-6fd26efa4f24"
	fixOpID            = "04f91bff-9e17-45cb-a246-84d511274ef1"

	gcpPlanID   = "ca6e5357-707f-4565-bbbd-b3ab732597c6"
	azurePlanID = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
)

type ClientTest struct {
	t *testing.T
}

func TestClient_Deprovision(t *testing.T) {
	t.Run("should return deprovisioning operation ID on success", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(t, nil)
		defer testServer.Close()

		config := NewClientConfig(testServer.URL)
		client := NewClientWithPoller(context.Background(), *config, NewPassthroughPoller())
		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: azurePlanID,
		}

		// when
		opID, err := client.Deprovision(instance)

		// then
		assert.NoError(t, err)
		assert.Equal(t, fixOpID, opID)
	})

	t.Run("should return error on failed request execution", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(t, requestFailureServerError)
		defer testServer.Close()

		config := NewClientConfig(testServer.URL)

		client := NewClientWithPoller(context.Background(), *config, NewPassthroughPoller())

		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: gcpPlanID,
		}

		// when
		opID, err := client.Deprovision(instance)

		// then
		assert.Error(t, err)
		assert.Len(t, opID, 0)
	})
}

func TestClient_ExpirationRequest(t *testing.T) {

	t.Run("should return true on successfully commenced suspension", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(t, nil)
		defer testServer.Close()

		config := ClientConfig{
			URL: testServer.URL,
		}
		client := NewClientWithPoller(context.Background(), config, NewPassthroughPoller())
		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixTrialInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: TrialPlanID,
		}

		// when
		suspensionUnderWay, err := client.SendExpirationRequest(instance)

		// then
		assert.NoError(t, err)
		assert.True(t, suspensionUnderWay)
	})

	t.Run("should return error when trying to make other plan than trial expired", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(t, nil)
		defer testServer.Close()

		config := ClientConfig{
			URL: testServer.URL,
		}
		client := NewClientWithPoller(context.Background(), config, NewPassthroughPoller())
		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: azurePlanID,
		}

		// when
		suspensionUnderWay, err := client.SendExpirationRequest(instance)

		// then
		assert.Error(t, err)
		assert.False(t, suspensionUnderWay)
	})

	t.Run("should return error when expiration request fails", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(t, requestFailureServerError)
		defer testServer.Close()

		config := ClientConfig{
			URL: testServer.URL,
		}
		client := NewClientWithPoller(context.Background(), config, NewPassthroughPoller())

		client.setHttpClient(testServer.Client())

		instance := internal.Instance{
			InstanceID:    fixInstanceID,
			RuntimeID:     fixRuntimeID,
			ServicePlanID: TrialPlanID,
		}

		// when
		suspensionUnderWay, err := client.SendExpirationRequest(instance)

		// then
		assert.Error(t, err)
		assert.False(t, suspensionUnderWay)
	})

	t.Run("should return true for non-existent instanceId and false for existing", func(t *testing.T) {
		// given
		testServer := fixHTTPServer(t, nil)
		defer testServer.Close()

		config := ClientConfig{
			URL: testServer.URL,
		}
		client := NewClientWithPoller(context.Background(), config, NewPassthroughPoller())
		client.setHttpClient(testServer.Client())

		// when
		response, err := client.GetInstanceRequest("non-existent")

		// then
		assert.NoError(t, err)
		defer func() { _ = response.Body.Close() }()
		assert.Equal(t, response.StatusCode, http.StatusNotFound)

		// when
		responseOtherThanNotFound, err := client.GetInstanceRequest("real")

		// then
		assert.NoError(t, err)
		defer func() { _ = responseOtherThanNotFound.Body.Close() }()
		assert.NotEqual(t, responseOtherThanNotFound.StatusCode, http.StatusNotFound)
	})
}

func fixHTTPServer(t *testing.T, requestFailureFunc func(http.ResponseWriter, *http.Request)) *httptest.Server {
	if requestFailureFunc != nil {
		r := httputil.NewRouter()
		r.HandleFunc("DELETE /oauth/v2/service_instances/{instance_id}", requestFailureFunc)
		r.HandleFunc("PATCH /oauth/v2/service_instances/{instance_id}", requestFailureFunc)
		r.HandleFunc("PUT /expire/service_instance/{instance_id}", requestFailureFunc)
		return httptest.NewServer(r)
	}

	clientTest := &ClientTest{t: t}
	r := httputil.NewRouter()
	r.HandleFunc("DELETE /oauth/v2/service_instances/{instance_id}", clientTest.deprovision)
	r.HandleFunc("PUT /expire/service_instance/{instance_id}", clientTest.expiration)
	r.HandleFunc("GET /oauth/v2/service_instances/{instance_id}", clientTest.getInstance)
	return httptest.NewServer(r)
}

func (c *ClientTest) expiration(w http.ResponseWriter, r *http.Request) {
	instanceID := strings.Split(r.URL.Path, "/")[3]
	if instanceID == fixInstanceID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if instanceID != fixTrialInstanceID {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_, err := w.Write([]byte(fmt.Sprintf(`{"operation": "%s"}`, fixOpID)))
	assert.NoError(c.t, err)
}

func (c *ClientTest) deprovision(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	_, okServiceID := params["service_id"]
	if !okServiceID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, okPlanID := params["plan_id"]
	if !okPlanID {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_, err := w.Write([]byte(fmt.Sprintf(`{"operation": "%s"}`, fixOpID)))
	assert.NoError(c.t, err)
}

func requestFailureServerError(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func requestFailureUnprocessableEntity(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusUnprocessableEntity)
}

func (c *ClientTest) getInstance(w http.ResponseWriter, r *http.Request) {
	instance := path.Base(r.URL.Path)
	if instance == "non-existent" {
		w.WriteHeader(http.StatusNotFound)
	} else {
		_, err := w.Write([]byte(fmt.Sprintf(`{"instanceID": "%s"}`, instance)))
		assert.NoError(c.t, err)
		w.WriteHeader(http.StatusOK)
	}
}
