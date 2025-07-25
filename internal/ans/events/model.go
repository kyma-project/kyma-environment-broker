package events

import (
	"fmt"
	"time"
)

type (
	ResourceEvent struct {
		ID                  string              `json:"id,omitempty"`
		Body                string              `json:"body"`
		Subject             string              `json:"subject"`
		EventType           string              `json:"eventType"`
		Priority            int64               `json:"priority,omitempty"`
		Resource            Resource            `json:"resource,omitempty"`
		EventTimeStamp      int64               `json:"eventTimeStamp"`
		Region              string              `json:"region,omitempty"`
		RegionType          string              `json:"regionType,omitempty"`
		Severity            Severity            `json:"severity"`
		Category            Category            `json:"category"`
		Visibility          Visibility          `json:"visibility"`
		NotificationMapping NotificationMapping `json:"notificationMapping,omitempty"`
		Source              string              `json:"source,omitempty"`
		SourceType          string              `json:"sourceType,omitempty"`
		Tags                map[string]string   `json:"tags,omitempty"`
	}

	ResourceEventOption func(*ResourceEvent) error

	Resource struct {
		Type          string            `json:"resourceType"`
		Name          string            `json:"resourceName"`
		Instance      string            `json:"resourceInstance,omitempty"`
		Subaccount    string            `json:"subAccount"`
		GlobalAccount string            `json:"globalAccount,omitempty"`
		ResourceGroup string            `json:"resourceGroup"`
		Tags          map[string]string `json:"tags,omitempty"`
	}

	ResourceOption func(*Resource) error

	NotificationMapping struct {
		DeduplicationID         string     `json:"deduplicationId,omitempty"`
		NotificationTypeKey     string     `json:"notificationTypeKey"`
		NotificationTemplateKey string     `json:"notificationTemplateKey,omitempty"`
		Recipients              Recipients `json:"recipients"`
		Navigation              Navigation `json:"navigation,omitempty"`
	}

	NotificationMappingOption func(*NotificationMapping) error

	Navigation struct {
		Action     string            `json:"action"`
		Object     string            `json:"object"`
		Parameters map[string]string `json:"parameters,omitempty"`
	}

	Recipients struct {
		XsuaaUsers []XsuaaRecipient `json:"xsuaa,omitempty"`
		Users      []UserRecipient  `json:"users,omitempty"`
	}

	XsuaaRecipient struct {
		Level     Level      `json:"level"`
		TenantID  string     `json:"tenantId"`
		RoleNames []RoleName `json:"roleNames,omitempty"`
	}
	UserRecipient struct {
		Email   string `json:"email"`
		IasHost string `json:"iasHost"`
	}

	Level      string
	Severity   string
	Category   string
	Visibility string
	RoleName   string
)

const (
	SeverityInfo         Severity   = "INFO"
	SeverityNotice       Severity   = "NOTICE"
	SeverityWarning      Severity   = "WARNING"
	SeverityError        Severity   = "ERROR"
	SeverityFatal        Severity   = "FATAL"
	CategoryException    Category   = "EXCEPTION"
	CategoryNotification Category   = "NOTIFICATION"
	CategoryAlert        Category   = "ALERT"
	VisibilitySource     Visibility = "SOURCE"
	//VisibilityInternal        Visibility = "INTERNAL" // failed during manual testing, not supported by the service
	VisibilityOwner           Visibility = "OWNER"
	VisibilityOwnerSubAccount Visibility = "OWNER_SUBACCOUNT"
	VisibilityGlobalAccount   Visibility = "GLOBAL_ACCOUNT"
	LevelGlobalAccount        Level      = "GLOBAL_ACCOUNT"
	LevelSubaccount           Level      = "SUBACCOUNT"
)

func (c Category) Validate() error {
	switch c {
	case CategoryException, CategoryNotification, CategoryAlert:
		return nil
	default:
		return fmt.Errorf("invalid category: %s", c)
	}
}

func (v Visibility) Validate() error {
	switch v {
	case VisibilitySource, VisibilityOwner, VisibilityOwnerSubAccount, VisibilityGlobalAccount:
		return nil
	default:
		return fmt.Errorf("invalid visibility: %s", v)
	}
}

func (l Level) Validate() error {
	switch l {
	case LevelGlobalAccount, LevelSubaccount:
		return nil
	default:
		return fmt.Errorf("invalid level: %s", l)
	}
}

func (s Severity) Validate() error {
	switch s {
	case SeverityInfo, SeverityNotice, SeverityWarning, SeverityError, SeverityFatal:
		return nil
	default:
		return fmt.Errorf("invalid severity: %s", s)
	}
}

func NewUserRecipient(email string, iasHost string) (*UserRecipient, error) {
	if email == "" {
		return nil, fmt.Errorf("email must not be empty")
	}
	if iasHost == "" {
		return nil, fmt.Errorf("IAS host must not be empty")
	}
	return &UserRecipient{
		Email:   email,
		IasHost: iasHost,
	}, nil
}

func NewXsuaaRecipient(level Level, tenantID string, roleNames []RoleName) (*XsuaaRecipient, error) {
	if err := level.Validate(); err != nil {
		return nil, fmt.Errorf("invalid XSUAA level: %w", err)
	}
	if tenantID == "" {
		return nil, fmt.Errorf("tenant ID must not be empty")
	}
	for _, roleName := range roleNames {
		if roleName == "" {
			return nil, fmt.Errorf("role name must not be empty")
		}
	}
	return &XsuaaRecipient{
		Level:     level,
		TenantID:  tenantID,
		RoleNames: roleNames,
	}, nil
}

func WithID(id string) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if len(id) == 0 {
			return fmt.Errorf("resource event ID must not be empty")
		}
		r.ID = id
		return nil
	}
}

func WithBody(body string) ResourceEventOption {
	return func(r *ResourceEvent) error {
		r.Body = body
		return nil
	}
}

func WithSubject(subject string) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if len(subject) == 0 {
			return fmt.Errorf("resource event subject must not be empty")
		}
		r.Subject = subject
		return nil
	}
}

func WithEventType(eventType string) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if len(eventType) == 0 {
			return fmt.Errorf("resource event type must not be empty")
		}
		r.EventType = eventType
		return nil
	}
}
func WithPriority(priority int64) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if priority < 0 {
			return fmt.Errorf("resource event priority must be non-negative")
		}
		r.Priority = priority
		return nil
	}
}

func WithResource(resource Resource) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if resource.Type == "" {
			return fmt.Errorf("resource type must not be empty")
		}
		if resource.Name == "" {
			return fmt.Errorf("resource name must not be empty")
		}
		if resource.Subaccount == "" {
			return fmt.Errorf("resource subaccount must not be empty")
		}
		if resource.ResourceGroup == "" {
			return fmt.Errorf("resource resource group must not be empty")
		}
		r.Resource = resource
		return nil
	}
}

func WithEventTimeStamp(eventTimeStamp int64) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if eventTimeStamp <= 0 {
			return fmt.Errorf("event timestamp must be a positive integer")
		}
		r.EventTimeStamp = eventTimeStamp
		return nil
	}
}

func WithRegion(region string) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if len(region) == 0 {
			return fmt.Errorf("region must not be empty")
		}
		r.Region = region
		return nil
	}
}

func WithRegionType(regionType string) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if len(regionType) == 0 {
			return fmt.Errorf("region type must not be empty")
		}
		r.RegionType = regionType
		return nil
	}
}

func WithSeverity(severity Severity) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if err := severity.Validate(); err != nil {
			return fmt.Errorf("invalid severity: %w", err)
		}
		r.Severity = severity
		return nil
	}
}

func WithCategory(category Category) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if err := category.Validate(); err != nil {
			return fmt.Errorf("invalid category: %w", err)
		}
		r.Category = category
		return nil
	}
}

func WithVisibility(visibility Visibility) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if err := visibility.Validate(); err != nil {
			return fmt.Errorf("invalid visibility: %w", err)
		}
		r.Visibility = visibility
		return nil
	}
}

func WithNotificationMapping(notificationMapping NotificationMapping) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if notificationMapping.NotificationTypeKey == "" {
			return fmt.Errorf("notification type key must not be empty")
		}
		r.NotificationMapping = notificationMapping
		return nil
	}
}

func WithSource(source string) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if len(source) == 0 {
			return fmt.Errorf("source must not be empty")
		}
		r.Source = source
		return nil
	}
}

func WithSourceType(sourceType string) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if len(sourceType) == 0 {
			return fmt.Errorf("source type must not be empty")
		}
		r.SourceType = sourceType
		return nil
	}
}

func WithTags(tags map[string]string) ResourceEventOption {
	return func(r *ResourceEvent) error {
		if tags == nil {
			return fmt.Errorf("tags must not be nil")
		}
		r.Tags = tags
		return nil
	}
}

func (r *ResourceEvent) Validate() error {
	if err := r.Severity.Validate(); err != nil {
		return fmt.Errorf("invalid severity: %w", err)
	}
	if err := r.Category.Validate(); err != nil {
		return fmt.Errorf("invalid category: %w", err)
	}
	if err := r.Visibility.Validate(); err != nil {
		return fmt.Errorf("invalid visibility: %w", err)

	}
	if r.EventType == "" {
		return fmt.Errorf("event type is empty")
	}
	if r.Subject == "" {
		return fmt.Errorf("subject is empty")
	}
	if r.Body == "" {
		return fmt.Errorf("body is empty")
	}
	if r.EventTimeStamp <= 0 {
		return fmt.Errorf("event timestamp is invalid: %d", r.EventTimeStamp)
	}
	return nil
}

func NewResourceEvent(eventType string, body string, subject string, resource Resource, eventTimeStamp *int64,
	severity Severity, category Category, visibility Visibility, notificationMapping NotificationMapping,
	options ...ResourceEventOption) (*ResourceEvent, error) {
	resourceEvent := &ResourceEvent{
		EventType:           eventType,
		Body:                body,
		Subject:             subject,
		Resource:            resource,
		Severity:            severity,
		Category:            category,
		Visibility:          visibility,
		NotificationMapping: notificationMapping,
	}

	if eventTimeStamp != nil {
		resourceEvent.EventTimeStamp = *eventTimeStamp
	} else {
		resourceEvent.EventTimeStamp = time.Now().UnixMilli()
	}

	for _, option := range options {
		if err := option(resourceEvent); err != nil {
			return nil, fmt.Errorf("error applying resource event option: %w", err)
		}
	}

	if err := resourceEvent.Validate(); err != nil {
		return nil, fmt.Errorf("invalid resource event: %w", err)
	}
	return resourceEvent, nil
}

func NewResource(resourceType, resourceName, subAccount, resourceGroup string, options ...ResourceOption) (*Resource, error) {
	if resourceType == "" {
		return nil, fmt.Errorf("resource type must not be empty")

	}
	if resourceName == "" {
		return nil, fmt.Errorf("resource name must not be empty")

	}
	if subAccount == "" {

		return nil, fmt.Errorf("subaccount must not be empty")
	}
	if resourceGroup == "" {
		return nil, fmt.Errorf("resource group must not be empty")
	}
	resource := &Resource{
		Type:          resourceType,
		Name:          resourceName,
		Subaccount:    subAccount,
		ResourceGroup: resourceGroup,
	}
	for _, option := range options {
		if err := option(resource); err != nil {
			return nil, fmt.Errorf("error applying resource option: %w", err)
		}

	}
	return resource, nil
}

func WithInstance(instance string) ResourceOption {
	return func(r *Resource) error {
		if instance == "" {
			return fmt.Errorf("resource instance must not be empty")
		}
		r.Instance = instance
		return nil
	}
}

func WithResourceGlobalAccount(globalAccount string) ResourceOption {
	return func(r *Resource) error {
		if globalAccount == "" {
			return fmt.Errorf("resource global account must not be empty")
		}
		r.GlobalAccount = globalAccount
		return nil
	}
}

func WithResourceTags(tags map[string]string) ResourceOption {
	return func(r *Resource) error {
		if tags == nil {
			return fmt.Errorf("resource tags must not be nil")
		}
		r.Tags = tags
		return nil
	}
}

func NewNotificationMapping(notificationTypeKey string, recipients Recipients, options ...NotificationMappingOption) (*NotificationMapping, error) {
	if notificationTypeKey == "" {
		return nil, fmt.Errorf("notification type key must not be empty")
	}
	if err := recipients.Validate(); err != nil {
		return nil, fmt.Errorf("invalid recipients: %w", err)
	}
	notificationMapping := &NotificationMapping{
		NotificationTypeKey: notificationTypeKey,
		Recipients:          recipients,
	}
	for _, option := range options {
		if err := option(notificationMapping); err != nil {
			return nil, fmt.Errorf("error applying notification mapping option: %w", err)
		}
	}
	return notificationMapping, nil
}

func WithDeduplicationID(deduplicationID string) NotificationMappingOption {
	return func(n *NotificationMapping) error {
		if deduplicationID == "" {
			return fmt.Errorf("deduplication ID must not be empty")
		}
		n.DeduplicationID = deduplicationID
		return nil
	}
}

func WithNotificationTemplateKey(notificationTemplateKey string) NotificationMappingOption {
	return func(n *NotificationMapping) error {
		if notificationTemplateKey == "" {
			return fmt.Errorf("notification template key must not be empty")
		}
		n.NotificationTemplateKey = notificationTemplateKey
		return nil
	}
}

func NewNavigation(action, object string, parameters map[string]string) (*Navigation, error) {
	return &Navigation{
		Action:     action,
		Object:     object,
		Parameters: parameters,
	}, nil
}

func (r *Recipients) Validate() error {
	if len(r.XsuaaUsers) == 0 && len(r.Users) == 0 {
		return fmt.Errorf("recipients must have at least one XSUAA user or user")
	}

	for _, xsuaaUser := range r.XsuaaUsers {
		if err := xsuaaUser.Level.Validate(); err != nil {
			return fmt.Errorf("invalid XSUAA level in recipient: %w", err)
		}
		if xsuaaUser.TenantID == "" {
			return fmt.Errorf("XSUAA recipient tenant ID is empty")
		}
	}

	for _, user := range r.Users {
		if user.Email == "" {
			return fmt.Errorf("user recipient email is empty")
		}
		if user.IasHost == "" {
			return fmt.Errorf("user recipient IAS host is empty")
		}
	}

	return nil
}

func NewRecipients(xsuaaUsers []XsuaaRecipient, users []UserRecipient) (*Recipients, error) {
	recipients := &Recipients{
		XsuaaUsers: xsuaaUsers,
		Users:      users,
	}
	if err := recipients.Validate(); err != nil {
		return nil, fmt.Errorf("invalid recipients: %w", err)
	}
	return recipients, nil
}
