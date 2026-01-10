package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kubeagents/kubeagents/models"
	"github.com/kubeagents/kubeagents/store"
)

// AgentHandler handles agent-related requests
type AgentHandler struct {
	store *store.Store
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(s *store.Store) *AgentHandler {
	return &AgentHandler{
		store: s,
	}
}

// AgentWithStats represents an agent with session statistics
type AgentWithStats struct {
	*models.Agent
	SessionCount      int    `json:"session_count"`
	ActiveSessionCount int   `json:"active_session_count"`
	LatestStatus      string `json:"latest_status,omitempty"`
	LatestMessage     string `json:"latest_message,omitempty"`
}

// ListAgents handles GET /api/agents
func (h *AgentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	statusFilter := r.URL.Query().Get("status")
	searchQuery := r.URL.Query().Get("search")

	// Get all agents
	agents := h.store.ListAgents()

	// Filter and search
	var filteredAgents []*models.Agent
	for _, agent := range agents {
		// Apply search filter
		if searchQuery != "" {
			searchLower := strings.ToLower(searchQuery)
			agentIDLower := strings.ToLower(agent.AgentID)
			nameLower := strings.ToLower(agent.Name)
			if !strings.Contains(agentIDLower, searchLower) && !strings.Contains(nameLower, searchLower) {
				continue
			}
		}

		// Apply status filter
		if statusFilter != "" {
			latestStatus, _ := h.getAgentLatestStatus(agent.AgentID)
			if latestStatus != statusFilter {
				continue
			}
		}

		filteredAgents = append(filteredAgents, agent)
	}

	// Build response with statistics
	agentsWithStats := make([]*AgentWithStats, 0, len(filteredAgents))
	for _, agent := range filteredAgents {
		stats := h.calculateAgentStats(agent.AgentID)
		agentsWithStats = append(agentsWithStats, &AgentWithStats{
			Agent:             agent,
			SessionCount:      stats.SessionCount,
			ActiveSessionCount: stats.ActiveSessionCount,
			LatestStatus:      stats.LatestStatus,
			LatestMessage:     stats.LatestMessage,
		})
	}

	response := map[string]interface{}{
		"agents": agentsWithStats,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// AgentStats represents session statistics for an agent
type AgentStats struct {
	SessionCount      int
	ActiveSessionCount int
	LatestStatus      string
	LatestMessage     string
}

// calculateAgentStats calculates statistics for an agent
func (h *AgentHandler) calculateAgentStats(agentID string) AgentStats {
	sessions := h.store.ListSessions(agentID, true)
	activeSessions := h.store.ListSessions(agentID, false)

	stats := AgentStats{
		SessionCount:      len(sessions),
		ActiveSessionCount: len(activeSessions),
	}

	// Find latest status across all sessions
	var latestStatus *models.AgentStatus
	for _, session := range sessions {
		status, err := h.store.GetLatestStatus(agentID, session.SessionTopic)
		if err != nil {
			continue
		}
		if latestStatus == nil || status.Timestamp.After(latestStatus.Timestamp) {
			latestStatus = status
		}
	}

	if latestStatus != nil {
		stats.LatestStatus = latestStatus.Status
		stats.LatestMessage = latestStatus.Message
	}

	return stats
}

// getAgentLatestStatus gets the latest status for an agent
func (h *AgentHandler) getAgentLatestStatus(agentID string) (string, error) {
	sessions := h.store.ListSessions(agentID, true)
	
	var latestStatus *models.AgentStatus
	for _, session := range sessions {
		status, err := h.store.GetLatestStatus(agentID, session.SessionTopic)
		if err != nil {
			continue
		}
		if latestStatus == nil || status.Timestamp.After(latestStatus.Timestamp) {
			latestStatus = status
		}
	}

	if latestStatus == nil {
		return "", nil
	}
	return latestStatus.Status, nil
}

// GetAgent handles GET /api/agents/{agent_id}
func (h *AgentHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agent_id")
	
	agent, err := h.store.GetAgent(agentID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "not_found", "Agent not found")
		return
	}

	// Calculate statistics for the agent
	stats := h.calculateAgentStats(agentID)
	
	// Create response with stats
	agentWithStats := AgentWithStats{
		Agent:              agent,
		SessionCount:       stats.SessionCount,
		ActiveSessionCount: stats.ActiveSessionCount,
		LatestStatus:       stats.LatestStatus,
		LatestMessage:      stats.LatestMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(agentWithStats)
}

// SessionWithStatus represents a session with its current status
type SessionWithStatus struct {
	*models.Session
	CurrentStatus *string `json:"current_status,omitempty"`
}

// ListSessions handles GET /api/agents/{agent_id}/sessions
func (h *AgentHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agent_id")
	
	// Check if agent exists
	_, err := h.store.GetAgent(agentID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "not_found", "Agent not found")
		return
	}

	// Get expired parameter
	includeExpired := r.URL.Query().Get("expired") != "false"
	
	sessions := h.store.ListSessions(agentID, includeExpired)

	// Enrich sessions with current status
	sessionsWithStatus := make([]SessionWithStatus, 0, len(sessions))
	for _, session := range sessions {
		sessionWithStatus := SessionWithStatus{
			Session: session,
		}
		
		// Get latest status for this session
		latestStatus, err := h.store.GetLatestStatus(agentID, session.SessionTopic)
		if err == nil && latestStatus != nil {
			sessionWithStatus.CurrentStatus = &latestStatus.Status
		}
		
		sessionsWithStatus = append(sessionsWithStatus, sessionWithStatus)
	}

	response := map[string]interface{}{
		"sessions": sessionsWithStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetSession handles GET /api/agents/{agent_id}/sessions/{session_topic}
func (h *AgentHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agent_id")
	sessionTopic := chi.URLParam(r, "session_topic")
	
	session, err := h.store.GetSession(agentID, sessionTopic)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "not_found", "Session not found")
		return
	}

	// Get status history
	history, _ := h.store.GetStatusHistory(agentID, sessionTopic)
	
	// Sort by timestamp descending (newest first)
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp.After(history[j].Timestamp)
	})

	response := map[string]interface{}{
		"session":       session,
		"status_history": history,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetAgentStatus handles GET /api/agents/{agent_id}/status
func (h *AgentHandler) GetAgentStatus(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agent_id")
	
	// Check if agent exists
	_, err := h.store.GetAgent(agentID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "not_found", "Agent not found")
		return
	}

	// Get latest status across all sessions
	sessions := h.store.ListSessions(agentID, true)
	var latestStatus *models.AgentStatus
	
	for _, session := range sessions {
		status, err := h.store.GetLatestStatus(agentID, session.SessionTopic)
		if err != nil {
			continue
		}
		if latestStatus == nil || status.Timestamp.After(latestStatus.Timestamp) {
			latestStatus = status
		}
	}

	if latestStatus == nil {
		h.respondError(w, http.StatusNotFound, "not_found", "No status found for agent")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(latestStatus)
}

// respondError sends an error response
func (h *AgentHandler) respondError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   errorCode,
		"message": message,
	})
}
