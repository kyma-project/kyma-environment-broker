package provider

const (
	PurposeEvaluation = "evaluation"
	PurposeProduction = "production"
)

func updateString(toUpdate *string, value *string) {
	if value != nil {
		*toUpdate = *value
	}
}

func updateSlice(toUpdate *[]string, value []string) {
	if value != nil {
		*toUpdate = value
	}
}
