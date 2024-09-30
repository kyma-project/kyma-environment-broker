package provider

const (
	PurposeEvaluation  = "evaluation"
	PurposeProduction  = "production"
	PurposeDevelopment = "development"
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
