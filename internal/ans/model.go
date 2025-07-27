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
		NavigationTargetObject    string            `json:"navigationTargetObject"`
		NavigationTargetAction    string            `json:"navigationTargetAction"`
		ActorID                   string            `json:"ActorId"`
		ActorDisplayText          string            `json:"ActorDisplayText"`
		ActorImageURL             string            `json:"ActorImageUrl"`
	}
	Recipient struct {
		GlobalUserId        string     `json:"globalUserId"`
		RecipientId         string     `json:"recipientId"`
		ProviderRecipientId string     `json:"providerRecipientId"`
		IasGroupId          string     `json:"iasGroupId"`
		XsuaaLevel          XsuaaLevel `json:"XsuaaLevel"`
		TenantId            string     `json:"tenantId"`
		RoleName            string     `json:"roleName"`
		Language            string     `json:"language"`
	}
	Property struct {
		Language     string       `json:"language"`
		Key          string       `json:"key"`
		Value        string       `json:"value"`
		PropertyType PropertyType `json:"type"`
		IsSensitive  bool         `json:"isSensitive"`
	}
	Attachment struct {
		Headers Headers `json:"headers"`
		Content Content `json:"content"`
	}
	Headers struct {
		ContentType        string `json:"contentType"`
		ContentDisposition string `json:"contentDisposition"`
		ContentID          string `json:"contentId"`
	}

	Content struct {
		External External `json:"external"`
	}

	External struct {
		Path string `json:"path"`
	}
	TargetParameter struct {
		Key   string `json:"key"`
		Value string `json:"value"`
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
	PropertyTypeString      PropertyType = "string"
	PropertyTypeJSON        PropertyType = "jsonobject"
)
