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

	NotificationOption func(notification *Notification)
	Recipient          struct {
		GlobalUserId        string      `json:"GlobalUserId,omitempty"`
		RecipientId         string      `json:"RecipientId"`
		IasHost             string      `json:"IasHost"`
		ProviderRecipientId string      `json:"ProviderRecipientId,omitempty"`
		IasGroupId          string      `json:"IasGroupId,omitempty"`
		XsuaaLevel          *XsuaaLevel `json:"XsuaaLevel,omitempty"`
		TenantId            string      `json:"TenantId,omitempty"`
		RoleName            string      `json:"RoleName,omitempty"`
		Language            string      `json:"Language,omitempty"`
	}

	RecipientOption func(recipient *Recipient)

	Property struct {
		Language     string       `json:"Language,omitempty"`
		Key          string       `json:"Key"`
		Value        string       `json:"Value"`
		PropertyType PropertyType `json:"Type,omitempty"`
		IsSensitive  *bool        `json:"IsSensitive,omitempty"`
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
		if err := recipient.Validate(); err != nil {
			return fmt.Errorf("invalid recipient: %w", err)
		}
	}
	return nil
}

func NewNotification(typeKey string, recipients []Recipient, options ...NotificationOption) *Notification {
	notification := &Notification{
		NotificationTypeKey: typeKey,
		Recipients:          recipients,
	}
	for _, option := range options {
		option(notification)
	}
	return notification
}

func WithID(id string) NotificationOption {
	return func(n *Notification) {
		n.ID = id
	}
}

func WithOriginID(originID string) NotificationOption {
	return func(n *Notification) {
		n.OriginID = originID
	}
}

func WithNotificationTypeID(notificationTypeID string) NotificationOption {
	return func(n *Notification) {
		n.NotificationTypeID = &notificationTypeID
	}
}

func WithNotificationTypeVersion(notificationTypeVersion string) NotificationOption {
	return func(n *Notification) {
		n.NotificationTypeVersion = &notificationTypeVersion
	}
}

func WithNotificationTypeTimestamp(notificationTypeTimestamp string) NotificationOption {
	return func(n *Notification) {
		n.NotificationTypeTimestamp = &notificationTypeTimestamp
	}
}

func WithNotificationTemplateKey(notificationTemplateKey string) NotificationOption {
	return func(n *Notification) {
		n.NotificationTemplateKey = &notificationTemplateKey
	}
}

func WithPriority(priority Priority) NotificationOption {
	return func(n *Notification) {
		n.Priority = &priority
	}
}
func WithProviderID(providerID string) NotificationOption {
	return func(n *Notification) {
		n.ProviderID = &providerID
	}
}

func WithProperties(properties []Property) NotificationOption {
	return func(n *Notification) {
		n.Properties = properties
	}
}

func WithTargetParameters(targetParameters []TargetParameter) NotificationOption {
	return func(n *Notification) {
		n.TargetParameters = targetParameters
	}
}

func WithAttachments(attachments []Attachment) NotificationOption {
	return func(n *Notification) {
		n.Attachments = attachments
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
		p.PropertyType = propertyType
		return nil
	}
}

func WithIsSensitive(isSensitive bool) PropertyOption {
	return func(p *Property) error {
		p.IsSensitive = &isSensitive
		return nil
	}
}

func NewRecipient(recipientID string, iasHost string, options ...RecipientOption) *Recipient {
	recipient := &Recipient{
		RecipientId: recipientID,
		IasHost:     iasHost,
	}
	for _, option := range options {
		option(recipient)
	}
	return recipient
}

func WithGlobalUserId(globalUserID string) RecipientOption {
	return func(r *Recipient) {
		r.GlobalUserId = globalUserID
	}
}

func WithXsuaaLevel(xsuaaLevel XsuaaLevel) RecipientOption {
	return func(r *Recipient) {
		r.XsuaaLevel = &xsuaaLevel
	}
}

func WithTenantId(tenantID string) RecipientOption {
	return func(r *Recipient) {
		r.TenantId = tenantID
	}
}

func WithRoleName(roleName string) RecipientOption {
	return func(r *Recipient) {
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
		r.Language = language
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
	if r.XsuaaLevel != nil {
		if err := r.XsuaaLevel.Validate(); err != nil {
			return fmt.Errorf("invalid XSUAA level: %w", err)
		}
	}
	if len(r.IasHost) == 0 {
		return fmt.Errorf("IAS host must not be empty")
	}
	return nil
}
