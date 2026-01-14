package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/kubeagents/kubeagents/internal"
	"github.com/kubeagents/kubeagents/models"
	"github.com/kubeagents/kubeagents/notifier"
	"github.com/kubeagents/kubeagents/store"
)

// WebhookHandler handles webhook status reports
type WebhookHandler struct {
	store    *store.Store
	notifier *notifier.NotificationManager
}

// NewWebhookHandler creates a new webhook handler without notifications
func NewWebhookHandler(s *store.Store) *WebhookHandler {
	return &WebhookHandler{
		store:    s,
		notifier: nil,
	}
}

// NewWebhookHandlerWithNotifier creates a new webhook handler with notifications
func NewWebhookHandlerWithNotifier(s *store.Store, n *notifier.NotificationManager) *WebhookHandler {
	return &WebhookHandler{
		store:    s,
		notifier: n,
	}
}

// SuccessResponse represents a successful response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ServeHTTP handles POST /webhook/status requests
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var statusReport internal.StatusReport
	if err := json.NewDecoder(r.Body).Decode(&statusReport); err != nil {
		h.respondError(w, http.StatusBadRequest, "bad_request", "Invalid JSON: "+err.Error())
		return
	}

	// Validate input
	if err := statusReport.Validate(); err != nil {
		h.respondError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	// Process status report
	if err := h.processStatusReport(&statusReport); err != nil {
		log.Printf("Error processing status report: %v", err)
		h.respondError(w, http.StatusInternalServerError, "internal_error", "Failed to process status report")
		return
	}

	// Respond with success
	h.respondSuccess(w, "Status reported successfully")
}

// processStatusReport processes a status report and updates the store
func (h *WebhookHandler) processStatusReport(sr *internal.StatusReport) error {
	now := time.Now()

	// Get previous status for transition detection
	var previousStatus string
	var startTimestamp time.Time
	history, _ := h.store.GetStatusHistory(sr.AgentID, sr.SessionTopic)
	if len(history) > 0 {
		// Find latest status
		latest := history[0]
		for _, s := range history {
			if s.Timestamp.After(latest.Timestamp) {
				latest = s
			}
		}
		previousStatus = latest.Status

		// Find the "running" status timestamp for duration calculation
		for _, s := range history {
			if s.Status == "running" {
				if startTimestamp.IsZero() || s.Timestamp.Before(startTimestamp) {
					startTimestamp = s.Timestamp
				}
			}
		}
	}

	// Create or update agent
	agent, err := h.store.GetAgent(sr.AgentID)
	if err != nil {
		// Agent doesn't exist, create new one
		agent = &models.Agent{
			AgentID:    sr.AgentID,
			Name:       sr.AgentName,
			Source:     sr.AgentSource,
			Registered: now,
			LastSeen:   now,
		}
	} else {
		// Agent exists, update it
		if sr.AgentName != "" {
			agent.Name = sr.AgentName
		}
		if sr.AgentSource != "" {
			agent.Source = sr.AgentSource
		}
		agent.LastSeen = now
	}

	if err := h.store.CreateOrUpdateAgent(agent); err != nil {
		return err
	}

	// Create or update session
	session, err := h.store.GetSession(sr.AgentID, sr.SessionTopic)
	if err != nil {
		// Session doesn't exist, create new one
		ttl := sr.TTLMinutes
		if ttl == 0 {
			ttl = 30 // default 30 minutes
		}

		session = &models.Session{
			AgentID:      sr.AgentID,
			SessionTopic: sr.SessionTopic,
			Created:      now,
			LastUpdated:  now,
			Expired:      false,
			TTLMinutes:   ttl,
		}
	} else {
		// Session exists, update it
		session.LastUpdated = now
		if sr.TTLMinutes > 0 {
			session.TTLMinutes = sr.TTLMinutes
		}
	}

	if err := h.store.CreateOrUpdateSession(session); err != nil {
		return err
	}

	// Add status to history
	agentStatus := &models.AgentStatus{
		AgentID:      sr.AgentID,
		SessionTopic: sr.SessionTopic,
		Status:       sr.Status,
		Timestamp:    sr.Timestamp,
		Message:      sr.Message,
		Content:      sr.Content,
	}

	if err := h.store.AddStatus(agentStatus); err != nil {
		return err
	}

	// Check for status transition and send notification
	if h.notifier != nil && previousStatus == "running" &&
		(sr.Status == "success" || sr.Status == "failed") {

		duration := time.Duration(0)
		if !startTimestamp.IsZero() {
			duration = sr.Timestamp.Sub(startTimestamp)
		}

		notificationData := &notifier.NotificationData{
			AgentID:      sr.AgentID,
			AgentName:    agent.Name,
			SessionTopic: sr.SessionTopic,
			FromStatus:   previousStatus,
			ToStatus:     sr.Status,
			Timestamp:    sr.Timestamp,
			Message:      sr.Message,
			Content:      sr.Content,
			Duration:     duration,
		}

		// Send notification asynchronously (non-blocking)
		if err := h.notifier.Notify(context.Background(), notificationData); err != nil {
			// Log error but don't fail the request
			log.Printf("Failed to queue notification: %v", err)
		}
	}

	return nil
}

// respondSuccess sends a success response
func (h *WebhookHandler) respondSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: message,
	})
}

// respondError sends an error response
func (h *WebhookHandler) respondError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}
