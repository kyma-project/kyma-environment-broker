package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEnabled_KimDisabled(t *testing.T) {
	config := &KimConfig{
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

func TestIsEnabled_KimEnabled_PlansWithWhitespaces(t *testing.T) {
	config := &KimConfig{
		Enabled:      true,
		Plans:        []string{"gcp ", " preview"},
		KimOnlyPlans: []string{"gcp ", " preview"},
	}

	assert.True(t, config.IsEnabledForPlan("gcp"))
	assert.True(t, config.IsEnabledForPlan("preview"))
	assert.True(t, config.IsDrivenByKim("gcp"))
	assert.True(t, config.IsDrivenByKim("preview"))
	assert.True(t, config.IsDrivenByKimOnly("gcp"))
	assert.True(t, config.IsDrivenByKimOnly("preview"))
}

func TestIsEnabled_KimEnabledForPreview(t *testing.T) {
	config := &KimConfig{
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
	config := &KimConfig{
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
	config := &KimConfig{
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
	assert.False(t, config.IsPlanIdDrivenByKimOnly("ca6e5357-707f-4565-bbbd-b3ab732597c6"))
	assert.False(t, config.IsPlanIdDrivenByKimOnly("5cb3d976-b85c-42ea-a636-79cadda109a9"))
	assert.False(t, config.IsPlanIdDrivenByKim("ca6e5357-707f-4565-bbbd-b3ab732597c6"))
	assert.False(t, config.IsPlanIdDrivenByKim("5cb3d976-b85c-42ea-a636-79cadda109a9"))
	assert.False(t, config.IsPlanIdDrivenByKimOnly("ca6e5357-707f-4565-bbbd-b3ab732597c6"))
	assert.False(t, config.IsPlanIdDrivenByKimOnly("5cb3d976-b85c-42ea-a636-79cadda109a9"))
}

func TestDrivenByKimOnly_PreviewByKimOnly(t *testing.T) {
	config := &KimConfig{
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
	config := &KimConfig{
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
	config := &KimConfig{
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
