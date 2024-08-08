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
	assert.False(t, config.IsDrivenByKim("gcp"))
	assert.False(t, config.IsDrivenByKim("preview"))
	assert.False(t, config.IsDrivenByKimOnly("gcp"))
	assert.False(t, config.IsDrivenByKimOnly("preview"))
}

func TestIsEnabled_KimEnabledForPreview(t *testing.T) {
	config := &Config{
		Enabled:  true,
		Plans:    []string{"preview"},
		ViewOnly: false,
		DryRun:   false,
	}

	assert.False(t, config.IsEnabledForPlan("gcp"))
	assert.True(t, config.IsEnabledForPlan("preview"))
	assert.False(t, config.IsDrivenByKim("gcp"))
	assert.True(t, config.IsDrivenByKim("preview"))
	assert.False(t, config.IsDrivenByKimOnly("gcp"))
	assert.False(t, config.IsDrivenByKimOnly("preview"))
}

func TestIsEnabled_KimEnabledForPreview_DryRun(t *testing.T) {
	config := &Config{
		Enabled:  true,
		Plans:    []string{"preview"},
		ViewOnly: false,
		DryRun:   true,
	}

	assert.False(t, config.IsEnabledForPlan("gcp"))
	assert.True(t, config.IsEnabledForPlan("preview"))
	assert.False(t, config.IsDrivenByKim("gcp"))
	assert.False(t, config.IsDrivenByKim("preview"))
	assert.False(t, config.IsDrivenByKimOnly("gcp"))
	assert.False(t, config.IsDrivenByKimOnly("preview"))
}

func TestDrivenByKimOnly_KimDisabled(t *testing.T) {
	config := &Config{
		Enabled:      false,
		Plans:        []string{"gcp", "preview"},
		KimOnlyPlans: []string{"preview"},
		ViewOnly:     false,
	}

	assert.False(t, config.IsDrivenByKimOnly("gcp"))
	assert.False(t, config.IsDrivenByKimOnly("preview"))
	assert.False(t, config.IsDrivenByKim("gcp"))
	assert.False(t, config.IsDrivenByKim("preview"))
	assert.False(t, config.IsDrivenByKimOnly("gcp"))
	assert.False(t, config.IsDrivenByKimOnly("preview"))
}

func TestDrivenByKimOnly_PreviewByKimOnly(t *testing.T) {
	config := &Config{
		Enabled:      true,
		Plans:        []string{"preview"},
		KimOnlyPlans: []string{"preview"},
		ViewOnly:     false,
	}

	assert.False(t, config.IsEnabledForPlan("gcp"))
	assert.True(t, config.IsDrivenByKimOnly("preview"))
	assert.False(t, config.IsDrivenByKim("gcp"))
	assert.True(t, config.IsDrivenByKim("preview"))
	assert.False(t, config.IsDrivenByKimOnly("gcp"))
	assert.True(t, config.IsDrivenByKimOnly("preview"))
}

func TestDrivenByKimOnly_PreviewByKimOnlyButNotEnabled(t *testing.T) {
	config := &Config{
		Enabled:      true,
		KimOnlyPlans: []string{"preview"},
		ViewOnly:     false,
	}

	assert.False(t, config.IsEnabledForPlan("gcp"))
	assert.False(t, config.IsDrivenByKimOnly("preview"))
	assert.False(t, config.IsDrivenByKim("gcp"))
	assert.False(t, config.IsDrivenByKim("preview"))
	assert.False(t, config.IsDrivenByKimOnly("gcp"))
	assert.False(t, config.IsDrivenByKimOnly("preview"))
}

func TestDrivenByKim_ButNotByKimOnly(t *testing.T) {
	config := &Config{
		Enabled:      true,
		KimOnlyPlans: []string{"no-plan"},
		Plans:        []string{"preview"},
		ViewOnly:     false,
		DryRun:       false,
	}

	assert.False(t, config.IsEnabledForPlan("gcp"))
	assert.False(t, config.IsDrivenByKimOnly("preview"))
	assert.False(t, config.IsDrivenByKim("gcp"))
	assert.True(t, config.IsDrivenByKim("preview"))
	assert.False(t, config.IsDrivenByKimOnly("gcp"))
	assert.False(t, config.IsDrivenByKimOnly("preview"))
}
