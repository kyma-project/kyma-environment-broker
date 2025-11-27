package broker

// MockChannelResolver is a test implementation that returns "fast" for all plans
// It can be used across all broker package tests to avoid duplication
type MockChannelResolver struct{}

func (m *MockChannelResolver) GetChannelForPlan(planID string) (string, error) {
	return "fast", nil
}

func (m *MockChannelResolver) GetAllPlanChannels() (map[string]string, error) {
	return map[string]string{
		"azure":             "fast",
		"azure_lite":        "fast",
		"trial":             "fast",
		"aws":               "fast",
		"gcp":               "fast",
		"freemium":          "fast",
		"sapconvergedcloud": "fast",
	}, nil
}
