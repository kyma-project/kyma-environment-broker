package version

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	// given
	rr := httptest.NewRecorder()
	router := httputil.NewRouter()
	runtimeHandler := NewHandler("1.25.18")
	runtimeHandler.AttachRoutes(router)

	req, err := http.NewRequest("GET", "/version", nil)
	require.NoError(t, err)

	// when
	router.ServeHTTP(rr, req)

	// then
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "{\"version\":\"1.25.18\"}", string(rr.Body.Bytes()))
}
