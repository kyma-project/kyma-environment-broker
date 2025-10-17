package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFullZoneName_AlicloudProvider(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		zone     string
		expected string
	}{
		{
			name:     "Region does not end with digit",
			region:   "cn-beijing",
			zone:     "a",
			expected: "cn-beijing-a",
		},
		{
			name:     "Region ends with digit",
			region:   "eu-central-1",
			zone:     "a",
			expected: "eu-central-1a",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := FullZoneName(AlicloudProviderType, test.region, test.zone)
			assert.Equal(t, test.expected, result)
		})
	}
}
