package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnablePlans_Unmarshal(t *testing.T) {
	// given
	planList := "gcp,azure,aws,sap-converged-cloud,free,alicloud"
	expectedPlanList := []string{"gcp", "azure", "aws", "sap-converged-cloud", "free", "alicloud"}
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

func TestEnablePlans_Contains(t *testing.T) {
	// given
	planList := "gcp,azure,aws,sap-converged-cloud,free,alicloud"
	enablePlans := EnablePlans{}
	err := enablePlans.Unmarshal(planList)
	assert.NoError(t, err)

	// when
	tests := []struct {
		name     string
		planName string
		expected bool
	}{
		{"Valid Plan", "gcp", true},
		{"Invalid Plan", "invalid", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := enablePlans.Contains(test.planName)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestEnablePlans_ContainsPlanID(t *testing.T) {
	// given
	planList := "gcp,azure,aws,sap-converged-cloud,free,alicloud"
	enablePlans := EnablePlans{}
	err := enablePlans.Unmarshal(planList)
	assert.NoError(t, err)

	// when
	tests := []struct {
		name     string
		planID   string
		expected bool
	}{
		{"Valid Plan ID", GCPPlanID, true},
		{"Invalid Plan ID", "invalid", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := enablePlans.ContainsPlanID(test.planID)
			assert.Equal(t, test.expected, result)
		})
	}
}
