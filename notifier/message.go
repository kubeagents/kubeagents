package notifier

import (
	"encoding/json"
	"fmt"
	"time"
)

// WebhookPayload represents the notification payload format
type WebhookPayload struct {
	MsgType string         `json:"msg_type"`
	Content WebhookContent `json:"content"`
}

// WebhookContent represents the content field of the webhook payload
type WebhookContent struct {
	Text string `json:"text"`
}

// NotificationData contains all information needed for notification
type NotificationData struct {
	AgentID      string
	AgentName    string
	SessionTopic string
	FromStatus   string
	ToStatus     string
	Timestamp    time.Time
	Message      string
	Content      string
	Duration     time.Duration
}

// FormatMessage creates a human-readable notification message
func FormatMessage(data *NotificationData) string {
	msg := fmt.Sprintf(
		"🔔 Session Status Change\n\n"+
			"Agent ID: %s\n"+
			"Agent Name: %s\n"+
			"Session: %s\n"+
			"Status: %s → %s\n"+
			"Timestamp: %s\n"+
			"Duration: %s",
		data.AgentID,
		data.AgentName,
		data.SessionTopic,
		data.FromStatus,
		data.ToStatus,
		data.Timestamp.Format(time.RFC3339),
		data.Duration.String(),
	)

	if data.Message != "" {
		msg += fmt.Sprintf("\nMessage: %s", data.Message)
	}

	if data.Content != "" {
		msg += fmt.Sprintf("\nContent: %s", data.Content)
	}

	return msg
}

// BuildPayload creates the webhook payload in JSON format
func BuildPayload(data *NotificationData) ([]byte, error) {
	payload := WebhookPayload{
		MsgType: "text",
		Content: WebhookContent{
			Text: FormatMessage(data),
		},
	}
	return json.Marshal(payload)
}
