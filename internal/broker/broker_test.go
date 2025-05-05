package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnablePlans_Unmarshal(t *testing.T) {
	// given
	planList := "gcp,azure,aws,sap-converged-cloud,free"
	expectedPlanList := []string{"gcp", "azure", "aws", "sap-converged-cloud", "free"}
	// when
	enablePlans := &EnablePlans{}
	err := enablePlans.Unmarshal(planList)
	// then
	assert.NoError(t, err)

	for _, plan := range expectedPlanList {
		if !enablePlans.Contains(plan) {
			t.Errorf("Expected plan %s not found in the list", plan)
		}
	}
	assert.Error(t, enablePlans.Unmarshal("invalid,plan"))
}
