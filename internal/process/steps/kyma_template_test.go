package steps

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitKymaTemplate_Run(t *testing.T) {
	// given
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	err := db.Operations().InsertOperation(operation)
	require.NoError(t, err)
	svc := NewInitKymaTemplate(db.Operations())
	ic := fixture.FixInputCreator("aws")
	ic.Config = &internal.ConfigForPlan{
		KymaTemplate: `
apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
    name: my-kyma
    namespace: kyma-system
spec:
    sync:
        strategy: secret
    channel: stable
    modules: []
`,
	}
	operation.InputCreator = ic

	// when
	op, backoff, err := svc.Run(operation, fixLogger())
	require.NoError(t, err)

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, "kyma-system", op.KymaResourceNamespace)
	assert.Equal(t, ic.Config.KymaTemplate, op.KymaTemplate)
}
