package servicebindingcleanup_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/servicebindingcleanup"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dberr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/driver/memory"
	"github.com/pivotal-cf/brokerapi/v8/domain/apiresponses"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	requestTimeout = 100 * time.Millisecond
	requestRetries = 2
)

func TestServiceBindingCleanupJob(t *testing.T) {
	// setup
	bindingsStorage := memory.NewBinding()
	handler := newUnbindHandler(bindingsStorage)
	srv := newServer(handler)
	defer srv.Close()

	ctx := context.TODO()
	brokerClientConfig := broker.ClientConfig{
		URL: srv.URL,
	}
	brokerClient := broker.NewClientWithRequestTimeoutAndRetries(ctx, brokerClientConfig, requestTimeout, requestRetries)

	t.Run("should not unbind service bindings in dry run mode", func(t *testing.T) {
		// given
		expectedBinding := internal.Binding{
			ID:         "binding-id",
			InstanceID: "instance-id",
		}
		require.NoError(t, bindingsStorage.Insert(&expectedBinding))

		svc := servicebindingcleanup.NewService(true, brokerClient, bindingsStorage)

		// when
		err := svc.PerformCleanup()
		require.NoError(t, err)

		// then
		actualBinding, err := bindingsStorage.Get(expectedBinding.InstanceID, expectedBinding.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedBinding.ID, actualBinding.ID)

		// cleanup
		require.NoError(t, bindingsStorage.Delete(expectedBinding.InstanceID, expectedBinding.ID))
	})

	t.Run("should unbind service bindings", func(t *testing.T) {
		// given
		bindings := []internal.Binding{
			{
				ID:         "binding-id-1",
				InstanceID: "instance-id-1",
				ExpiresAt:  time.Now().Add(-time.Hour),
			},
			{
				ID:         "binding-id-2",
				InstanceID: "instance-id-2",
				ExpiresAt:  time.Now().Add(-time.Hour),
			},
			{
				ID:         "binding-id-3",
				InstanceID: "instance-id-3",
				ExpiresAt:  time.Now().Add(-time.Hour),
			},
		}
		for _, b := range bindings {
			require.NoError(t, bindingsStorage.Insert(&b))
		}

		svc := servicebindingcleanup.NewService(false, brokerClient, bindingsStorage)

		// when
		err := svc.PerformCleanup()
		require.NoError(t, err)

		// then
		for _, b := range bindings {
			_, err = bindingsStorage.Get(b.InstanceID, b.ID)
			assert.True(t, dberr.IsNotFound(err))
		}
	})

	t.Run("should timeout on unbind requests", func(t *testing.T) {
		// given
		expectedBinding := internal.Binding{
			ID:         "binding-id-1",
			InstanceID: "instance-id-1",
			ExpiresAt:  time.Now().Add(-time.Hour),
		}
		require.NoError(t, bindingsStorage.Insert(&expectedBinding))

		svc := servicebindingcleanup.NewService(false, brokerClient, bindingsStorage)

		// when
		handler.setHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * requestTimeout)
		})
		err := svc.PerformCleanup()
		require.NoError(t, err)

		// then
		actualBinding, err := bindingsStorage.Get(expectedBinding.InstanceID, expectedBinding.ID)
		require.NoError(t, err)
		assert.Equal(t, expectedBinding.ID, actualBinding.ID)

		// cleanup
		require.NoError(t, bindingsStorage.Delete(expectedBinding.InstanceID, expectedBinding.ID))
		handler.setHandlerFunc(handler.deleteServiceBindingFromStorage)
	})

	t.Run("should only unbind expired service binding", func(t *testing.T) {
		// given
		activeBinding := internal.Binding{
			ID:         "active-binding-id",
			InstanceID: "instance-id",
			ExpiresAt:  time.Now().Add(time.Hour),
		}
		expiredBinding := internal.Binding{
			ID:         "expired-binding-id",
			InstanceID: "instance-id",
			ExpiresAt:  time.Now().Add(-time.Hour),
		}
		require.NoError(t, bindingsStorage.Insert(&activeBinding))
		require.NoError(t, bindingsStorage.Insert(&expiredBinding))

		svc := servicebindingcleanup.NewService(false, brokerClient, bindingsStorage)

		// when
		err := svc.PerformCleanup()
		require.NoError(t, err)

		// then
		actualBinding, err := bindingsStorage.Get(activeBinding.InstanceID, activeBinding.ID)
		require.NoError(t, err)
		assert.Equal(t, activeBinding.ID, actualBinding.ID)

		_, err = bindingsStorage.Get(expiredBinding.InstanceID, expiredBinding.ID)
		assert.True(t, dberr.IsNotFound(err))
	})

	t.Run("should continue cleanup when unbind endpoint returns 410 status code", func(t *testing.T) {
		// given
		binding := internal.Binding{
			ID:         "binding-id-1",
			InstanceID: "instance-id-1",
			ExpiresAt:  time.Now().Add(-time.Hour),
		}
		require.NoError(t, bindingsStorage.Insert(&binding))

		svc := servicebindingcleanup.NewService(false, brokerClient, bindingsStorage)

		// when
		handler.setHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bindingID := r.PathValue("binding_id")
			instanceID := r.PathValue("instance_id")
			if len(bindingID) == 0 || len(instanceID) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, err := bindingsStorage.Get(instanceID, bindingID)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err = bindingsStorage.Delete(instanceID, bindingID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusGone)
			encoder := json.NewEncoder(w)
			if err := encoder.Encode(apiresponses.EmptyResponse{}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		})

		err := svc.PerformCleanup()
		require.NoError(t, err)

		// then
		_, err = bindingsStorage.Get(binding.InstanceID, binding.ID)
		assert.True(t, dberr.IsNotFound(err))

		// cleanup
		handler.setHandlerFunc(handler.deleteServiceBindingFromStorage)
	})

	t.Run("should process request only when the request contains required query params", func(t *testing.T) {
		// given
		binding := internal.Binding{
			ID:         "binding-id-1",
			InstanceID: "instance-id-1",
			ExpiresAt:  time.Now().Add(-time.Hour),
		}
		require.NoError(t, bindingsStorage.Insert(&binding))

		svc := servicebindingcleanup.NewService(false, brokerClient, bindingsStorage)

		// when
		handler.setHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, r.ParseForm())
			if len(r.Form.Get("service_id")) == 0 || len(r.Form.Get("plan_id")) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			encoder := json.NewEncoder(w)
			if err := encoder.Encode(apiresponses.EmptyResponse{}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		})

		err := svc.PerformCleanup()

		// then
		require.NoError(t, err)

		// cleanup
		require.NoError(t, bindingsStorage.Delete(binding.InstanceID, binding.ID))
		handler.setHandlerFunc(handler.deleteServiceBindingFromStorage)
	})
}

type server struct {
	*httptest.Server
}

func newServer(handler *unbindHandler) *server {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /oauth/v2/service_instances/{instance_id}/service_bindings/{binding_id}", handler.unbind)

	return &server{httptest.NewServer(mux)}
}

type unbindHandler struct {
	bindings   storage.Bindings
	handleFunc func(w http.ResponseWriter, r *http.Request)
}

func newUnbindHandler(bindings storage.Bindings) *unbindHandler {
	handler := &unbindHandler{bindings: bindings}
	handler.defaults()
	return handler
}

func (h *unbindHandler) defaults() {
	h.handleFunc = h.deleteServiceBindingFromStorage
}

func (h *unbindHandler) deleteServiceBindingFromStorage(w http.ResponseWriter, r *http.Request) {
	bindingID := r.PathValue("binding_id")
	instanceID := r.PathValue("instance_id")
	if len(bindingID) == 0 || len(instanceID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err := h.bindings.Get(instanceID, bindingID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err = h.bindings.Delete(instanceID, bindingID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(apiresponses.EmptyResponse{}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *unbindHandler) setBindingsStorage(bindings storage.Bindings) {
	h.bindings = bindings
}

func (h *unbindHandler) setHandlerFunc(handleFunc func(w http.ResponseWriter, r *http.Request)) {
	h.handleFunc = handleFunc
}

func (h *unbindHandler) unbind(w http.ResponseWriter, r *http.Request) {
	h.handleFunc(w, r)
}
