package internal

import (
	"encoding/json"
	"errors"
	"time"
)

// StatusReport represents the incoming status report from webhook
type StatusReport struct {
	AgentID      string    `json:"agent_id"`
	AgentName    string    `json:"agent_name,omitempty"`
	AgentSource  string    `json:"agent_source,omitempty"`
	SessionTopic string    `json:"session_topic"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
	Message      string    `json:"message,omitempty"`
	Content      string    `json:"content,omitempty"`
	TTLMinutes   int       `json:"ttl_minutes,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for StatusReport
func (sr *StatusReport) UnmarshalJSON(data []byte) error {
	type Alias StatusReport
	aux := &struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(sr),
	}
	
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	// Parse timestamp
	if aux.Timestamp != "" {
		t, err := time.Parse(time.RFC3339, aux.Timestamp)
		if err != nil {
			return err
		}
		sr.Timestamp = t
	}
	
	return nil
}

// Validate validates StatusReport input
func (sr *StatusReport) Validate() error {
	if sr.AgentID == "" {
		return errors.New("agent_id is required")
	}
	if len(sr.AgentID) > 100 {
		return errors.New("agent_id must be 1-100 characters")
	}
	if len(sr.AgentName) > 200 {
		return errors.New("agent_name must be 0-200 characters")
	}
	if len(sr.AgentSource) > 200 {
		return errors.New("agent_source must be 0-200 characters")
	}
	if sr.SessionTopic == "" {
		return errors.New("session_topic is required")
	}
	if len(sr.SessionTopic) > 500 {
		return errors.New("session_topic must be 1-500 characters")
	}
	
	validStatuses := map[string]bool{
		"running": true,
		"success": true,
		"failed":  true,
		"pending": true,
	}
	if !validStatuses[sr.Status] {
		return errors.New("status must be one of: running, success, failed, pending")
	}
	
	if sr.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	
	if len(sr.Message) > 1000 {
		return errors.New("message must be 0-1000 characters")
	}
	if len(sr.Content) > 10000 {
		return errors.New("content must be 0-10000 characters")
	}
	
	if sr.TTLMinutes < 0 || (sr.TTLMinutes > 0 && (sr.TTLMinutes < 1 || sr.TTLMinutes > 1440)) {
		return errors.New("ttl_minutes must be 0 or 1-1440")
	}
	
	return nil
}

// ParseStatusReport parses JSON into StatusReport
func ParseStatusReport(data []byte) (*StatusReport, error) {
	var sr StatusReport
	if err := json.Unmarshal(data, &sr); err != nil {
		return nil, err
	}
	if err := sr.Validate(); err != nil {
		return nil, err
	}
	return &sr, nil
}
