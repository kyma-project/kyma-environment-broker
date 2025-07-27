package ans

type (
	Notification struct {
		ID                        string            `json:"id"`
		OriginID                  string            `json:"originId"`
		NotificationTypeKey       string            `json:"NotificationTypeKey"`
		NotificationTypeID        string            `json:"NotificationTypeId"`
		NotificationTypeVersion   string            `json:"NotificationTypeVersion"`
		NotificationTypeTimestamp string            `json:"NotificationTypeTimestamp"`
		NotificationTemplateKey   string            `json:"NotificationTemplateKey"`
		Priority                  Priority          `json:"Priority"`
		ProviderID                string            `json:"ProviderId"`
		Recipients                []Recipient       `json:"Recipients"`
		Properties                []Property        `json:"Properties"`
		TargetParameters          []TargetParameter `json:"TargetParameters"`
		Attachments               []Attachment      `json:"Attachments"`
		NavigationTargetObject    string            `json:"NavigationTargetObject"`
		NavigationTargetAction    string            `json:"NavigationTargetAction"`
		ActorID                   string            `json:"ActorId"`
		ActorDisplayText          string            `json:"ActorDisplayText"`
		ActorImageURL             string            `json:"ActorImageUrl"`
	}
	Recipient struct {
		GlobalUserId        string     `json:"GlobalUserId"`
		RecipientId         string     `json:"RecipientId"`
		ProviderRecipientId string     `json:"ProviderRecipientId"`
		IasGroupId          string     `json:"IasGroupId"`
		XsuaaLevel          XsuaaLevel `json:"XsuaaLevel"`
		TenantId            string     `json:"TenantId"`
		RoleName            string     `json:"RoleName"`
		Language            string     `json:"Language"`
	}
	Property struct {
		Language     string       `json:"Language"`
		Key          string       `json:"Key"`
		Value        string       `json:"Value"`
		PropertyType PropertyType `json:"Type"`
		IsSensitive  bool         `json:"IsSensitive"`
	}
	Attachment struct {
		Headers Headers `json:"Headers"`
		Content Content `json:"Content"`
	}
	Headers struct {
		ContentType        string `json:"ContentType"`
		ContentDisposition string `json:"ContentDisposition"`
		ContentID          string `json:"ContentId"`
	}

	Content struct {
		External External `json:"External"`
	}

	External struct {
		Path string `json:"Path"`
	}
	TargetParameter struct {
		Key   string `json:"Key"`
		Value string `json:"Value"`
	}

	PropertyType string
	Priority     string
	Severity     string
	XsuaaLevel   string
)

func NewNotification(typeKey string, recipients []Recipient, options ...func(*Notification)) *Notification {
	return &Notification{}
}

func NewProperty(key, value string, options ...func(*Property)) *Property {
	property := &Property{
		Key:   key,
		Value: value,
	}
	for _, option := range options {
		option(property)
	}
	return property
}

func (p *Property) WithLanguage(language string) *Property {
	p.Language = language
	return p
}

func (p *Property) WithType(propertyType PropertyType) *Property {
	p.PropertyType = propertyType
	return p
}

func (p *Property) WithIsSensitive(isSensitive bool) *Property {
	p.IsSensitive = isSensitive
	return p
}

func NewRecipient(recipientID string, options ...func(*Recipient)) *Recipient {
	recipient := &Recipient{
		RecipientId: recipientID,
	}
	for _, option := range options {
		option(recipient)
	}
	return recipient
}

func (r *Recipient) WithGlobalUserID(globalUserID string) *Recipient {
	r.GlobalUserId = globalUserID
	return r
}

func (r *Recipient) WithXsuaaLevel(xsuaaLevel XsuaaLevel) *Recipient {
	r.XsuaaLevel = xsuaaLevel
	return r
}

func (r *Recipient) WithTenantID(tenantID string) *Recipient {
	r.TenantId = tenantID
	return r
}

func (r *Recipient) WithRoleName(roleName string) *Recipient {
	r.RoleName = roleName
	return r
}

func (r *Recipient) WithLanguage(language string) *Recipient {
	r.Language = language
	return r
}

func (r *Recipient) WithIasGroupID(iasGroupID string) *Recipient {
	r.IasGroupId = iasGroupID
	return r
}

func (r *Recipient) WithProviderRecipientID(providerRecipientID string) *Recipient {
	r.ProviderRecipientId = providerRecipientID
	return r
}

const (
	PriorityLow             Priority     = "LOW"
	PriorityNeutral         Priority     = "NEUTRAL"
	PriorityMedium          Priority     = "MEDIUM"
	PriorityHigh            Priority     = "HIGH"
	XsuaaLevelGlobalAccount XsuaaLevel   = "GLOBAL_ACCOUNT"
	XsuaaLevelSubaccount    XsuaaLevel   = "SUBACCOUNT"
	PropertyTypeString      PropertyType = "String"
	PropertyTypeJSON        PropertyType = "JsonObject"
)
