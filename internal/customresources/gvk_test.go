package customresources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGvkByName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "kyma",
			expected: "operator.kyma-project.io/v1beta2, Kind=Kyma",
		},
		{
			name:     "gardenercluster",
			expected: "infrastructuremanager.kyma-project.io/v1, Kind=GardenerCluster",
		},
		{
			name:     "runtime",
			expected: "infrastructuremanager.kyma-project.io/v1, Kind=Runtime",
		},
		{
			name:     "ruNtiMe",
			expected: "infrastructuremanager.kyma-project.io/v1, Kind=Runtime",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvk, err := GvkByName(tt.name)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if gvk.String() != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, gvk.String())
			}
		})
	}
}

func TestGvkByNameEmptyName(t *testing.T) {
	_, err := GvkByName("")
	assert.ErrorContains(t, err, "unknown name")
}

func TestUnknownName(t *testing.T) {
	_, err := GvkByName("unknown")
	assert.ErrorContains(t, err, "unknown name")
}

func TestGvrByName(t *testing.T) {
	tests := []struct {
		name             string
		expectedResource string
		expectedGroup    string
		expectedVersion  string
	}{
		{
			name:             "runtime",
			expectedResource: "runtimes",
			expectedGroup:    "infrastructuremanager.kyma-project.io",
			expectedVersion:  "v1",
		},
		{
			name:             "kyma",
			expectedResource: "kymas",
			expectedGroup:    "operator.kyma-project.io",
			expectedVersion:  "v1beta2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvr, err := GvrByName(tt.name)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResource, gvr.Resource)
			assert.Equal(t, tt.expectedGroup, gvr.Group)
			assert.Equal(t, tt.expectedVersion, gvr.Version)
		})
	}
}
