package broker

func (b *BindingContext) CreatedBy() string {
	if b.Email != nil && *b.Email != "" {
		return *b.Email
	} else if b.Origin != nil && *b.Origin != "" {
		return *b.Origin
	}
	return ""
}
