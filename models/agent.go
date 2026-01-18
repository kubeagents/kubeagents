package models

import (
	"errors"
	"time"
)

// Agent represents an external AI Agent system
type Agent struct {
	AgentID    string    `json:"agent_id"`
	UserID     string    `json:"user_id,omitempty"` // Owner user ID for data isolation
	Name       string    `json:"name,omitempty"`
	Source     string    `json:"source,omitempty"`
	Registered time.Time `json:"registered"`
	LastSeen   time.Time `json:"last_seen"`
}

// Validate validates Agent fields
func (a *Agent) Validate() error {
	if a.AgentID == "" {
		return errors.New("agent_id is required")
	}
	if len(a.AgentID) > 100 {
		return errors.New("agent_id must be 1-100 characters")
	}
	if len(a.Name) > 200 {
		return errors.New("name must be 0-200 characters")
	}
	if len(a.Source) > 200 {
		return errors.New("source must be 0-200 characters")
	}
	if a.Registered.IsZero() {
		return errors.New("registered time is required")
	}
	if a.LastSeen.IsZero() {
		return errors.New("last_seen time is required")
	}
	return nil
}

// Session represents a task (task equals Session)
type Session struct {
	AgentID      string     `json:"agent_id"`
	SessionTopic string     `json:"session_topic"`
	Created      time.Time  `json:"created"`
	LastUpdated  time.Time  `json:"last_updated"`
	Expired      bool       `json:"expired"`
	ExpiredAt    *time.Time `json:"expired_at,omitempty"`
	TTLMinutes   int        `json:"ttl_minutes,omitempty"`
}

// Validate validates Session fields
func (s *Session) Validate() error {
	if s.AgentID == "" {
		return errors.New("agent_id is required")
	}
	if s.SessionTopic == "" {
		return errors.New("session_topic is required")
	}
	if len(s.SessionTopic) > 500 {
		return errors.New("session_topic must be 1-500 characters")
	}
	if s.Created.IsZero() {
		return errors.New("created time is required")
	}
	if s.LastUpdated.IsZero() {
		return errors.New("last_updated time is required")
	}
	if s.LastUpdated.Before(s.Created) {
		return errors.New("last_updated must be >= created")
	}
	if s.TTLMinutes < 0 || s.TTLMinutes > 1440 {
		return errors.New("ttl_minutes must be 0 or 1-1440")
	}
	return nil
}

// AgentStatus represents Agent status entity, recording Session status history
type AgentStatus struct {
	AgentID      string    `json:"agent_id"`
	SessionTopic string    `json:"session_topic"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
	Message      string    `json:"message,omitempty"`
	Content      string    `json:"content,omitempty"`
}

// Validate validates AgentStatus fields
func (as *AgentStatus) Validate() error {
	if as.AgentID == "" {
		return errors.New("agent_id is required")
	}
	if as.SessionTopic == "" {
		return errors.New("session_topic is required")
	}
	validStatuses := map[string]bool{
		"running": true,
		"success": true,
		"failed":  true,
		"pending": true,
	}
	if !validStatuses[as.Status] {
		return errors.New("status must be one of: running, success, failed, pending")
	}
	if as.Timestamp.IsZero() {
		return errors.New("timestamp is required")
	}
	if len(as.Message) > 1000 {
		return errors.New("message must be 0-1000 characters")
	}
	if len(as.Content) > 10000 {
		return errors.New("content must be 0-10000 characters")
	}
	return nil
}
