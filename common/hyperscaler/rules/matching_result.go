package rules

import "github.com/google/uuid"

type MatchingResult struct {
	ParsingResultID uuid.UUID

	OriginalRule string

	Rule Rule

	Matched bool

	FinalMatch bool

	ProvisioningAttributes *ProvisioningAttributes
}

type Result map[string]string

func (r Result) IsShared() bool {
	value, found := r[SHARED_LABEL]
	return found && value == "true"
}

func (m *MatchingResult) Labels() map[string]string {
	return m.Rule.Labels(m.ProvisioningAttributes)
}
