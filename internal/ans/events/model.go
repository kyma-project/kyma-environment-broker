package events

import (
	"fmt"
)

type (
	ResourceEvent struct {
		ID                  string              `json:"id,omitempty"`
		Body                string              `json:"body"`
		Subject             string              `json:"subject"`
		EventType           string              `json:"eventType"`
		Priority            int64               `json:"priority,omitempty"`
		Resource            Resource            `json:"resource,omitempty"`
		EventTimeStamp      *int64              `json:"eventTimeStamp,omitempty"`
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

	ResourceEventOption func(*ResourceEvent)

	Resource struct {
		Type          string            `json:"resourceType"`
		Name          string            `json:"resourceName"`
		Instance      string            `json:"resourceInstance,omitempty"`
		Subaccount    string            `json:"subAccount"`
		GlobalAccount string            `json:"globalAccount,omitempty"`
		ResourceGroup string            `json:"resourceGroup"`
		Tags          map[string]string `json:"tags,omitempty"`
	}

	ResourceOption func(*Resource)

	NotificationMapping struct {
		DeduplicationID         string      `json:"deduplicationId,omitempty"`
		NotificationTypeKey     string      `json:"notificationTypeKey"`
		NotificationTemplateKey string      `json:"notificationTemplateKey,omitempty"`
		Recipients              Recipients  `json:"recipients"`
		Navigation              *Navigation `json:"navigation,omitempty"`
	}

	NotificationMappingOption func(*NotificationMapping)

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

func NewUserRecipient(email string, iasHost string) *UserRecipient {
	return &UserRecipient{
		Email:   email,
		IasHost: iasHost,
	}
}

func (r *UserRecipient) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email must not be empty")
	}
	if r.IasHost == "" {
		return fmt.Errorf("IAS host must not be empty")
	}
	return nil
}

func NewXsuaaRecipient(level Level, tenantID string, roleNames []RoleName) *XsuaaRecipient {
	return &XsuaaRecipient{
		Level:     level,
		TenantID:  tenantID,
		RoleNames: roleNames,
	}
}

func (r *XsuaaRecipient) Validate() error {
	if err := r.Level.Validate(); err != nil {
		return fmt.Errorf("invalid XSUAA level: %w", err)
	}
	if r.TenantID == "" {
		return fmt.Errorf("tenant ID must not be empty")
	}
	for _, roleName := range r.RoleNames {
		if roleName == "" {
			return fmt.Errorf("role names must not be empty")
		}
	}
	return nil
}

func WithID(id string) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.ID = id
	}
}

func WithBody(body string) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Body = body
	}
}

func WithSubject(subject string) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Subject = subject
	}
}

func WithEventType(eventType string) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.EventType = eventType
	}
}
func WithPriority(priority int64) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Priority = priority
	}
}

func WithResource(resource Resource) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Resource = resource
	}
}

func (r Resource) Validate() error {
	if r.Type == "" {
		return fmt.Errorf("resource type must not be empty")
	}
	if r.Name == "" {
		return fmt.Errorf("resource name must not be empty")
	}
	if r.Subaccount == "" {
		return fmt.Errorf("resource subaccount must not be empty")
	}
	if r.ResourceGroup == "" {
		return fmt.Errorf("resource resource group must not be empty")
	}
	return nil
}

func WithEventTimeStamp(eventTimeStamp int64) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.EventTimeStamp = &eventTimeStamp
	}
}

func WithRegion(region string) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Region = region
	}
}

func WithRegionType(regionType string) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.RegionType = regionType
	}
}

func WithSeverity(severity Severity) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Severity = severity
	}
}

func WithCategory(category Category) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Category = category
	}
}

func WithVisibility(visibility Visibility) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Visibility = visibility
	}
}

func WithNotificationMapping(notificationMapping NotificationMapping) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.NotificationMapping = notificationMapping
	}
}

func WithSource(source string) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Source = source
	}
}

func WithSourceType(sourceType string) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.SourceType = sourceType
	}
}

func WithTags(tags map[string]string) ResourceEventOption {
	return func(r *ResourceEvent) {
		r.Tags = tags
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
	if r.EventTimeStamp != nil && *r.EventTimeStamp <= 0 {
		return fmt.Errorf("event timestamp is invalid: %d", r.EventTimeStamp)
	}
	if r.NotificationMapping.Validate() != nil {
		return fmt.Errorf("invalid notification mapping: %w", r.NotificationMapping.Validate())
	}
	if err := r.Resource.Validate(); err != nil {
		return fmt.Errorf("invalid resource: %w", err)
	}
	return nil
}

func NewResourceEvent(eventType string, body string, subject string, resource Resource,
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

	for _, option := range options {
		option(resourceEvent)
	}

	if err := resourceEvent.Validate(); err != nil {
		return nil, fmt.Errorf("invalid resource event: %w", err)
	}
	return resourceEvent, nil
}

func NewResource(resourceType, resourceName, subAccount, resourceGroup string, options ...ResourceOption) Resource {
	resource := &Resource{
		Type:          resourceType,
		Name:          resourceName,
		Subaccount:    subAccount,
		ResourceGroup: resourceGroup,
	}
	for _, option := range options {
		option(resource)
	}
	return *resource
}

func WithInstance(instance string) ResourceOption {
	return func(r *Resource) {
		r.Instance = instance
	}
}

func WithResourceGlobalAccount(globalAccount string) ResourceOption {
	return func(r *Resource) {
		r.GlobalAccount = globalAccount
	}
}

func WithResourceTags(tags map[string]string) ResourceOption {
	return func(r *Resource) {
		r.Tags = tags
	}
}

func NewNotificationMapping(notificationTypeKey string, recipients Recipients, options ...NotificationMappingOption) *NotificationMapping {
	notificationMapping := &NotificationMapping{
		NotificationTypeKey: notificationTypeKey,
		Recipients:          recipients,
	}
	for _, option := range options {
		option(notificationMapping)
	}
	return notificationMapping
}

func (n *NotificationMapping) Validate() error {
	if n.NotificationTypeKey == "" {
		return fmt.Errorf("notification type key must not be empty")
	}
	if err := n.Recipients.Validate(); err != nil {
		return fmt.Errorf("invalid recipients: %w", err)
	}
	if n.Navigation != nil {
		if err := n.Navigation.Validate(); err != nil {
			return fmt.Errorf("invalid navigation: %w", err)
		}
	}
	return nil
}

func WithDeduplicationID(deduplicationID string) NotificationMappingOption {
	return func(n *NotificationMapping) {
		n.DeduplicationID = deduplicationID
	}
}

func WithNotificationTemplateKey(notificationTemplateKey string) NotificationMappingOption {
	return func(n *NotificationMapping) {
		n.NotificationTemplateKey = notificationTemplateKey
	}
}

func NewNavigation(action, object string, parameters map[string]string) *Navigation {
	return &Navigation{
		Action:     action,
		Object:     object,
		Parameters: parameters,
	}
}

func (n Navigation) Validate() error {
	if n.Action == "" {
		return fmt.Errorf("navigation action must not be empty")
	}
	if n.Object == "" {
		return fmt.Errorf("navigation object must not be empty")
	}
	return nil
}

func (r *Recipients) Validate() error {
	if len(r.XsuaaUsers) == 0 && len(r.Users) == 0 {
		return fmt.Errorf("recipients must have at least one XSUAA user or user")
	}

	for _, xsuaaUser := range r.XsuaaUsers {
		if err := xsuaaUser.Validate(); err != nil {
			return fmt.Errorf("invalid XSUAA user in recipient: %w", err)
		}
	}

	for _, user := range r.Users {
		if err := user.Validate(); err != nil {
			return fmt.Errorf("invalid user in recipient: %w", err)
		}
	}
	return nil
}

func NewRecipients(xsuaaUsers []XsuaaRecipient, users []UserRecipient) *Recipients {
	recipients := &Recipients{
		XsuaaUsers: xsuaaUsers,
		Users:      users,
	}
	return recipients
}
