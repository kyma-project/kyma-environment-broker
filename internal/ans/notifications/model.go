package notifications

import "fmt"

type (
	Notification struct {
		ID                        string            `json:"id,omitempty"`
		OriginID                  string            `json:"originId,omitempty"`
		NotificationTypeKey       string            `json:"NotificationTypeKey"`
		NotificationTypeID        *string           `json:"NotificationTypeId,omitempty"`
		NotificationTypeVersion   *string           `json:"NotificationTypeVersion,omitempty"`
		NotificationTypeTimestamp *string           `json:"NotificationTypeTimestamp,omitempty"`
		NotificationTemplateKey   *string           `json:"NotificationTemplateKey,omitempty"`
		Priority                  *Priority         `json:"Priority,omitempty"`
		ProviderID                *string           `json:"ProviderId,omitempty"`
		Recipients                []Recipient       `json:"Recipients"`
		Properties                []Property        `json:"Properties,omitempty"`
		TargetParameters          []TargetParameter `json:"TargetParameters,omitempty"`
		Attachments               []Attachment      `json:"Attachments,omitempty"`
		NavigationTargetObject    *string           `json:"NavigationTargetObject,omitempty"`
		NavigationTargetAction    *string           `json:"NavigationTargetAction,omitempty"`
		ActorID                   *string           `json:"ActorId,omitempty"`
		ActorDisplayText          *string           `json:"ActorDisplayText,omitempty"`
		ActorImageURL             *string           `json:"ActorImageUrl,omitempty"`
	}

	NotificationOption func(notification *Notification) error
	Recipient          struct {
		GlobalUserId        string     `json:"GlobalUserId,omitempty"`
		RecipientId         string     `json:"RecipientId"`
		IasHost             string     `json:"IasHost,omitempty"`
		ProviderRecipientId string     `json:"ProviderRecipientId,omitempty"`
		IasGroupId          string     `json:"IasGroupId,omitempty"`
		XsuaaLevel          XsuaaLevel `json:"XsuaaLevel,omitempty"`
		TenantId            string     `json:"TenantId,omitempty"`
		RoleName            string     `json:"RoleName,omitempty"`
		Language            string     `json:"Language,omitempty"`
	}

	RecipientOption func(recipient *Recipient)

	Property struct {
		Language     *string       `json:"Language,omitempty"`
		Key          string        `json:"Key"`
		Value        string        `json:"Value"`
		PropertyType *PropertyType `json:"Type,omitempty"`
		IsSensitive  *bool         `json:"IsSensitive,omitempty"`
	}

	PropertyOption func(property *Property) error
	Attachment     struct {
		Headers Headers `json:"Headers"`
		Content Content `json:"Content"`
	}

	AttachmentOption func(attachment *Attachment) error
	Headers          struct {
		ContentType        string `json:"ContentType"`
		ContentDisposition string `json:"ContentDisposition"`
		ContentID          string `json:"ContentId"`
	}

	HeadersOption func(headers *Headers) error
	Content       struct {
		External External `json:"External"`
	}

	ContentOption  func(content *Content) error
	ExternalOption func(external *External) error
	External       struct {
		Path string `json:"Path"`
	}
	TargetParameter struct {
		Key   string `json:"Key"`
		Value string `json:"Value"`
	}
	TargetParameterOption func(targetParameter *TargetParameter) error
	PropertyType          string
	Priority              string
	XsuaaLevel            string
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

func (p Priority) Validate() error {
	switch p {
	case PriorityLow, PriorityNeutral, PriorityMedium, PriorityHigh:
		return nil
	default:
		return fmt.Errorf("invalid priority: %s", p)
	}
}

func (pt PropertyType) Validate() error {
	switch pt {
	case PropertyTypeString, PropertyTypeJSON:
		return nil
	default:
		return fmt.Errorf("invalid property type: %s", pt)
	}
}

func (xl *XsuaaLevel) Validate() error {
	if xl != nil {
		switch *xl {
		case XsuaaLevelGlobalAccount, XsuaaLevelSubaccount:
			return nil
		default:
			return fmt.Errorf("invalid XSUAA level: %s", *xl)
		}
	}
	return nil
}

func (n *Notification) Validate() error {
	if len(n.NotificationTypeKey) == 0 {
		return fmt.Errorf("notification type key must not be empty")
	}
	if len(n.Recipients) == 0 {
		return fmt.Errorf("recipients must not be empty")
	}
	for _, recipient := range n.Recipients {
		if err := recipient.XsuaaLevel.Validate(); err != nil {
			return fmt.Errorf("invalid XSUAA level in recipient: %w", err)
		}
	}
	return nil
}

func NewNotification(typeKey string, recipients []Recipient, options ...NotificationOption) (*Notification, error) {
	notification := &Notification{
		NotificationTypeKey: typeKey,
		Recipients:          recipients,
	}
	for _, option := range options {
		if err := option(notification); err != nil {
			return nil, err
		}
	}
	if err := notification.Validate(); err != nil {
		return nil, fmt.Errorf("invalid notification:  %w", err)
	}
	return notification, nil
}

func WithID(id string) NotificationOption {
	return func(n *Notification) error {
		if len(id) == 0 {
			return fmt.Errorf("notification ID must not be empty")
		}
		n.ID = id
		return nil
	}
}

func WithOriginID(originID string) NotificationOption {
	return func(n *Notification) error {
		if len(originID) == 0 {
			return fmt.Errorf("origin ID must not be empty")
		}
		n.OriginID = originID
		return nil
	}
}

func WithNotificationTypeID(notificationTypeID string) NotificationOption {
	return func(n *Notification) error {
		if len(notificationTypeID) == 0 {
			return fmt.Errorf("notification type ID must not be empty")
		}
		n.NotificationTypeID = &notificationTypeID
		return nil
	}
}

func WithNotificationTypeVersion(notificationTypeVersion string) NotificationOption {
	return func(n *Notification) error {
		if len(notificationTypeVersion) == 0 {
			return fmt.Errorf("notification type version must not be empty")
		}
		n.NotificationTypeVersion = &notificationTypeVersion
		return nil
	}
}

func WithNotificationTypeTimestamp(notificationTypeTimestamp string) NotificationOption {
	//TODO default to current time if empty
	return func(n *Notification) error {
		if len(notificationTypeTimestamp) == 0 {
			return fmt.Errorf("notification type timestamp must not be empty")
		}
		n.NotificationTypeTimestamp = &notificationTypeTimestamp
		return nil
	}
}

func WithNotificationTemplateKey(notificationTemplateKey string) NotificationOption {
	return func(n *Notification) error {
		if len(notificationTemplateKey) == 0 {
			return fmt.Errorf("notification template key must not be empty")
		}
		n.NotificationTemplateKey = &notificationTemplateKey
		return nil
	}
}

func WithPriority(priority Priority) NotificationOption {
	return func(n *Notification) error {
		if err := priority.Validate(); err != nil {
			return fmt.Errorf("invalid priority: %w", err)
		}
		n.Priority = &priority
		return nil
	}
}
func WithProviderID(providerID string) NotificationOption {
	return func(n *Notification) error {
		if len(providerID) == 0 {
			return fmt.Errorf("provider ID must not be empty")
		}
		n.ProviderID = &providerID
		return nil
	}
}

func WithProperties(properties []Property) NotificationOption {
	return func(n *Notification) error {
		if len(properties) == 0 {
			return fmt.Errorf("properties must not be empty")
		}
		for _, property := range properties {
			if property.PropertyType != nil {
				if err := property.PropertyType.Validate(); err != nil {
					return fmt.Errorf("invalid property type: %w", err)
				}
			}
		}
		n.Properties = properties
		return nil
	}
}

func WithTargetParameters(targetParameters []TargetParameter) NotificationOption {
	return func(n *Notification) error {
		if len(targetParameters) == 0 {
			return fmt.Errorf("target parameters must not be empty")
		}
		n.TargetParameters = targetParameters
		return nil
	}
}

func WithAttachments(attachments []Attachment) NotificationOption {
	return func(n *Notification) error {
		if len(attachments) == 0 {
			return fmt.Errorf("attachments must not be empty")
		}
		n.Attachments = attachments
		return nil
	}
}

func NewTargetParameter(key, value string) TargetParameter {
	return TargetParameter{
		Key:   key,
		Value: value,
	}
}

func NewAttachment(headers Headers, content Content) Attachment {
	return Attachment{
		Headers: headers,
		Content: content,
	}
}

func NewHeaders(contentType, contentDisposition, contentID string) Headers {
	return Headers{
		ContentType:        contentType,
		ContentDisposition: contentDisposition,
		ContentID:          contentID,
	}
}

func NewContent(external External) Content {
	return Content{
		External: external,
	}
}

func NewExternal(path string) External {
	return External{
		Path: path,
	}
}

func NewProperty(key, value string, options ...PropertyOption) (*Property, error) {
	property := &Property{
		Key:   key,
		Value: value,
	}
	for _, option := range options {
		if err := option(property); err != nil {
			return nil, err
		}
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("property key must not be empty")
	}
	return property, nil
}

func WithType(propertyType PropertyType) PropertyOption {
	return func(p *Property) error {
		if err := propertyType.Validate(); err != nil {
			return fmt.Errorf("invalid property type: %w", err)
		}
		p.PropertyType = &propertyType
		return nil
	}
}

func WithIsSensitive(isSensitive bool) PropertyOption {
	return func(p *Property) error {
		p.IsSensitive = &isSensitive
		return nil
	}
}

func NewRecipient(recipientID string, options ...RecipientOption) *Recipient {
	recipient := &Recipient{
		RecipientId: recipientID,
	}
	for _, option := range options {
		option(recipient)
	}
	return recipient
}

func WithGlobalUserID(globalUserID string) RecipientOption {
	return func(r *Recipient) {
		//if len(globalUserID) == 0 {
		//	return fmt.Errorf("global user ID must not be empty")
		//}
		r.GlobalUserId = globalUserID
	}
}
func WithIasHost(iasHost string) RecipientOption {
	return func(r *Recipient) {
		//if len(iasHost) == 0 {
		//	return fmt.Errorf("IAS host must not be empty")
		//}
		r.IasHost = iasHost
	}
}

func WithXsuaaLevel(xsuaaLevel XsuaaLevel) RecipientOption {
	return func(r *Recipient) {
		//if err := xsuaaLevel.Validate(); err != nil {
		//	return fmt.Errorf("invalid XSUAA level: %w", err)
		//}
		r.XsuaaLevel = xsuaaLevel
	}
}

func WithTenantID(tenantID string) RecipientOption {
	return func(r *Recipient) {
		//if len(tenantID) == 0 {
		//	return fmt.Errorf("tenant ID must not be empty")
		//}
		r.TenantId = tenantID
	}
}

func WithRoleName(roleName string) RecipientOption {
	return func(r *Recipient) {
		//if len(roleName) == 0 {
		//	return fmt.Errorf("role name must not be empty")
		//}
		r.RoleName = roleName
	}
}

func WithLanguage(language string) RecipientOption {
	return func(r *Recipient) {
		r.Language = language
	}
}

func WithPropertyLanguage(language string) PropertyOption {
	return func(r *Property) error {
		if len(language) == 0 {
			return fmt.Errorf("language must not be empty")
		}
		r.Language = &language
		return nil
	}
}

func WithIasGroupId(iasGroupId string) RecipientOption {
	return func(r *Recipient) {
		r.IasGroupId = iasGroupId
	}
}

func WithProviderRecipientID(providerRecipientId string) RecipientOption {
	return func(r *Recipient) {
		r.ProviderRecipientId = providerRecipientId
	}
}

func (r *Recipient) Validate() error {
	if len(r.RecipientId) == 0 {
		return fmt.Errorf("recipient ID must not be empty")
	}
	if err := r.XsuaaLevel.Validate(); err != nil {
		return fmt.Errorf("invalid XSUAA level: %w", err)
	}
	if len(r.ProviderRecipientId) == 0 {
		return fmt.Errorf("provider recipient ID must not be empty")
	}
	if len(r.IasGroupId) == 0 {
		return fmt.Errorf("IAS group ID must not be empty")
	}
	if len(r.Language) == 0 {
		return fmt.Errorf("language must not be empty")
	}

	return nil
}
