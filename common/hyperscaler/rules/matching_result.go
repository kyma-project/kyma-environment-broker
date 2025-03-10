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

func (m *MatchingResult) Labels() map[string]string {
	return m.Rule.Labels(m.ProvisioningAttributes)
}
