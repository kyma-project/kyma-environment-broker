package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnablePlans_Contains(t *testing.T) {
	// given
	planList := "gcp,azure,aws,sap-converged-cloud,free,alicloud"
	enablePlans := StringList{}
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
