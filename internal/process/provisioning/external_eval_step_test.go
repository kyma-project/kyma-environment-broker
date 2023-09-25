package provisioning

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/avs"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalEvalStep_Run(t *testing.T) {
	memoryStorage := storage.NewMemoryStorage()
	externalEvalCreator, mockOauthServer, mockAvsSvc := setupAvs(t, memoryStorage.Operations())
	defer mockAvsSvc.server.Close()
	defer mockOauthServer.Close()

	operation := fixOperationRuntimeStatus(broker.GCPPlanID, internal.GCP)
	operation.Avs.AvsEvaluationInternalId = fixAvsEvaluationInternalId
	err := memoryStorage.Operations().InsertOperation(operation)
	assert.NoError(t, err)
	step := ExternalEvalStep{
		externalEvalCreator: externalEvalCreator,
	}

	// when
	_, retry, err := step.Run(operation, logrus.New())

	// then
	assert.Zero(t, retry)
	assert.NoError(t, err)
	inDB, _ := memoryStorage.Operations().GetOperationByID(operation.ID)
	assert.Contains(t, mockAvsSvc.evals, inDB.Avs.AVSEvaluationExternalId)
}

func setupAvs(t *testing.T, operations storage.Operations) (*ExternalEvalCreator, *httptest.Server, *mockAvsService) {
	mockOauthServer := newMockAvsOauthServer()
	mockAvsSvc := newMockAvsService(t, false)
	mockAvsSvc.startServer()
	mockAvsSvc.evals[fixAvsEvaluationInternalId] = fixAvsEvaluation()
	avsConfig := avsConfig(mockOauthServer, mockAvsSvc.server)
	avsClient, err := avs.NewClient(context.TODO(), avsConfig, logrus.New())
	require.NoError(t, err)
	avsDel := avs.NewDelegator(avsClient, avsConfig, operations)
	externalEvalAssistant := avs.NewExternalEvalAssistant(avsConfig)
	externalEvalCreator := NewExternalEvalCreator(avsDel, false, externalEvalAssistant)

	return externalEvalCreator, mockOauthServer, mockAvsSvc
}
