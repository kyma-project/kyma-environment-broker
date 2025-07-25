package ans

type (
	DataTenantPayload struct {
		Name        string `json:"name"`
		Environment string `json:"environment"`
		Secret      string `json:"secret"`
	}
	Notification struct {
		ID                      string           `json:"id"`
		NotificationTypeKey     string           `json:"notificationTypeKey"`
		NotificationTemplateKey string           `json:"notificationTemplateKey"`
		Priority                Priority         `json:"priority"`
		Severity                Severity         `json:"severity"`
		Recipients              []Recipient      `json:"recipients"`
		Properties              []Property       `json:"properties"`
		Attachments             []Attachment     `json:"attachments"`
		NavigationTarget        NavigationTarget `json:"navigationTarget"`
		Actor                   Actor            `json:"actor"`
	}
	Recipient struct {
		GlobalUserId string `json:"globalUserId"`
		RecipientId  string `json:"recipientId"`
		IasHost      string `json:"iasHost"`
		IasGroupId   string `json:"iasGroupId"`
		Language     string `json:"language"`
	}
	Property struct {
		Language     string       `json:"language"`
		Key          string       `json:"key"`
		Value        string       `json:"value"`
		PropertyType PropertyType `json:"type"`
	}
	Attachment struct {
		Headers Attachment.Headers `json:"headers"`
		Content Attachment.Content `json:"content"`
	}
	Header struct {
		ContentType        string `json:"contentType"`
		ContentDisposition string `json:"contentDisposition"`
	}

	NavigationTarget struct {
		Object     string                       `json:"object"`
		Action     string                       `json:"action"`
		Parameters []NavigationTarget.Parameter `json:"parameters"`
	}
	Parameter struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	Actor struct {
		ID          string `json:"id"`
		DisplayText string `json:"displayText"`
		ImageURL    string `json:"imageUrl"`
	}

	PropertyType string
	Priority     string
	Severity     string
)

const (
	PriorityLow        Priority     = "low"
	PriorityNeutral    Priority     = "neutral"
	PriorityMedium     Priority     = "medium"
	PriorityHigh       Priority     = "high"
	SeverityInfo       Severity     = "info"
	SeveritySuccess    Severity     = "success"
	SeverityWarning    Severity     = "warning"
	SeverityError      Severity     = "error"
	PropertyTypeString PropertyType = "string"
	PropertyTypeJSON   PropertyType = "jsonobject"
)
