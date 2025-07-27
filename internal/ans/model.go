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
