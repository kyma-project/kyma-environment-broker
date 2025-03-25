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

type Result struct {
	HyperscalerType string
	EUAccess        bool
	Shared          bool
	RawData         RawData
}

func (r Result) Hyperscaler() string {
	return r.HyperscalerType
}

func (r Result) IsShared() bool {
	return r.Shared
}

func (r Result) IsEUAccess() bool {
	return r.EUAccess
}

func (r Result) Rule() string {
	return r.RawData.Rule
}

func (r Result) NumberedRule() string {
	return r.RawData.NumberedRule()
}
