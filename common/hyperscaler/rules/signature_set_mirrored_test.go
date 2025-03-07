package rules

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignatureSet_Mirrored(t *testing.T) {

	t.Run("should return nil list for non existing signature", func(t *testing.T) {
		rule1 := NewRule()

		signatureSet := NewSignatureSet([]*ParsingResult{})

		_, err := rule1.SetPlan("aws", &broker.EnablePlans{"aws"})
		require.NoError(t, err)
		mirroredResults := signatureSet.Mirrored(rule1)

		assert.Nil(t, mirroredResults)

		for range mirroredResults {
			require.Fail(t, "should not return any result")
		}
	})
}
