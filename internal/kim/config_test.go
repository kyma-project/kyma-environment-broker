package kim

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEnabled_KimDisabled(t *testing.T) {
	config := &Config{
		Enabled:  false,
		Plans:    []string{"gcp", "preview"},
		ViewOnly: false,
	}

	assert.False(t, config.IsEnabledForPlan("gcp"))
	assert.False(t, config.IsEnabledForPlan("preview"))
}

func TestIsEnabled_KimEnabledForPreview(t *testing.T) {
	config := &Config{
		Enabled:  true,
		Plans:    []string{"preview"},
		ViewOnly: false,
	}

	assert.False(t, config.IsEnabledForPlan("gcp"))
	assert.True(t, config.IsEnabledForPlan("preview"))
}
