package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/common/orchestration"
	"github.com/kyma-project/kyma-environment-broker/internal/process"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestClusterHandler_AttachRoutes(t *testing.T) {
	t.Run("upgrade", func(t *testing.T) {
		// given
		handler := fixClusterHandler(t)

		params := orchestration.Parameters{
			Targets: orchestration.TargetSpec{
				Include: []orchestration.RuntimeTarget{
					{
						RuntimeID: "test",
					},
				},
			},
			Strategy: orchestration.StrategySpec{
				Schedule: "now",
			},
		}
		p, err := json.Marshal(&params)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "/upgrade/cluster", bytes.NewBuffer(p))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := chi.NewRouter()
		handler.AttachRoutes(router)

		// when
		router.ServeHTTP(rr, req)

		// then
		require.Equal(t, http.StatusAccepted, rr.Code)

		var out orchestration.UpgradeResponse

		err = json.Unmarshal(rr.Body.Bytes(), &out)
		require.NoError(t, err)
		assert.NotEmpty(t, out.OrchestrationID)
	})
}

func fixClusterHandler(t *testing.T) *clusterHandler {
	db := storage.NewMemoryStorage()
	logs := logrus.New()
	q := process.NewQueue(&testExecutor{}, logs)
	handler := NewClusterHandler(db.Orchestrations(), q, logs)

	return handler
}

type testExecutor struct{}

func (t *testExecutor) Execute(opID string) (time.Duration, error) {
	return 0, nil
}
