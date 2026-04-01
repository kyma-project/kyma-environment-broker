package broker

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
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

func Test_gvisorToBool(t *testing.T) {
	type args struct {
		gvisor *runtime.GvisorDTO
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "returns false for nil",
			args: args{
				gvisor: nil,
			},
			want: false,
		},
		{
			name: "returns false for disabled gvisor",
			args: args{
				gvisor: &runtime.GvisorDTO{Enabled: false},
			},
			want: false,
		},
		{
			name: "returns true for enabled gvisor",
			args: args{
				gvisor: &runtime.GvisorDTO{Enabled: true},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, gvisorToBool(tt.args.gvisor), "gvisorToBool(%v)", tt.args.gvisor)
		})
	}
}
