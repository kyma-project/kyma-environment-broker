package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/middleware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ctxKey string

const (
	ctxParentKeyA   ctxKey = "key-A"
	ctxParentValueA string = "value-A"
)

func TestRequestRegionKey(t *testing.T) {
	// given
	const (
		fixRequestRegion = "request-region-A"
	)

	req, err := http.NewRequest(http.MethodGet, "http://url.dev/endpoint/"+fixRequestRegion, nil)
	require.NoError(t, err)

	parentCtx := context.WithValue(context.Background(), ctxParentKeyA, ctxParentValueA)
	req = req.WithContext(parentCtx)

	var gotCtx context.Context
	spyHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotCtx = req.Context()
	})

	router := httputil.NewRouter()
	regionMiddleware := middleware.AddRegionToContext("default-region")
	router.Use(regionMiddleware)
	router.HandleFunc("/endpoint/{region}", spyHandler)

	// when
	router.ServeHTTP(httptest.NewRecorder(), req)
	gotRegion, foundRegion := middleware.RegionFromContext(gotCtx)

	// then
	assert.Equal(t, ctxParentValueA, gotCtx.Value(ctxParentKeyA))

	assert.True(t, foundRegion)
	assert.Equal(t, fixRequestRegion, gotRegion)
}

func TestRequestRegionKeyDefault(t *testing.T) {
	// given
	const (
		fixDefaultRegion = "default-region"
	)

	clientReq, err := http.NewRequest(http.MethodGet, "http://url.dev/endpoint-without-region", nil)
	require.NoError(t, err)

	parentCtx := context.WithValue(context.Background(), ctxParentKeyA, ctxParentValueA)
	clientReq = clientReq.WithContext(parentCtx)

	var gotCtx context.Context
	spyHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		gotCtx = req.Context()
	})

	regionMiddleware := middleware.AddRegionToContext(fixDefaultRegion)
	router := httputil.NewRouter()
	router.Use(regionMiddleware)
	router.HandleFunc("/endpoint-without-region", spyHandler)

	// when
	router.ServeHTTP(httptest.NewRecorder(), clientReq)
	gotRegion, foundRegion := middleware.RegionFromContext(gotCtx)

	// then
	assert.Equal(t, ctxParentValueA, gotCtx.Value(ctxParentKeyA))

	assert.True(t, foundRegion)
	assert.Equal(t, fixDefaultRegion, gotRegion)
}

func TestRequestRegionKeyNotFound(t *testing.T) {
	// when
	gotValue, found := middleware.RegionFromContext(context.Background())

	// then
	assert.Empty(t, gotValue)
	assert.False(t, found)
}
