package rules

import "github.com/google/uuid"

type MatchingResult struct {
	ParsingResultID uuid.UUID

	OriginalRule string

	Rule *Rule

	Matched bool

	FinalMatch bool
}
