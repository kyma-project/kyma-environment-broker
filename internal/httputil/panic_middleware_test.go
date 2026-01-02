package httputil_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanicRecoveryMiddleware_CatchesPanic(t *testing.T) {
	// given
	var logOutput strings.Builder
	logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{Level: slog.LevelError}))

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic in middleware")
	})

	middleware := httputil.PanicRecoveryMiddleware(logger)
	wrappedHandler := middleware(panicHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// when
	wrappedHandler.ServeHTTP(rr, req)

	// then
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "Internal Server Error")

	logs := logOutput.String()
	assert.Contains(t, logs, "panic recovered in HTTP handler: test panic in middleware")
	assert.Contains(t, logs, "path=/test")
	assert.Contains(t, logs, "method=GET")
	assert.Contains(t, logs, "stack=")
}

func TestPanicRecoveryMiddleware_NormalExecution(t *testing.T) {
	// given
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	middleware := httputil.PanicRecoveryMiddleware(logger)
	wrappedHandler := middleware(normalHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// when
	wrappedHandler.ServeHTTP(rr, req)

	// then
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())
}
