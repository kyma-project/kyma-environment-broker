package rules

type SignatureSet struct {
	items map[string][]*ParsingResult
}

func NewSignatureSet(results []*ParsingResult) *SignatureSet {
	signatureSet := &SignatureSet{
		items: make(map[string][]*ParsingResult),
	}

	for _, result := range results {
		if result.HasErrors() {
			continue
		}

		signature := result.Rule.SignatureWithSymbols(ASTERISK, ATTRIBUTE_WITH_VALUE)
		signatureSet.items[signature] = append(signatureSet.items[signature], result)
	}

	return signatureSet
}

func (s *SignatureSet) Mirrored(rule *Rule) []*ParsingResult {
	mirroredSignatureKey := rule.MirroredSignature()

	mirroredSignatureItems, _ := s.items[mirroredSignatureKey]

	return mirroredSignatureItems
}
