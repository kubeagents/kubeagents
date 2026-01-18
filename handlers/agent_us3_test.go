package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kubeagents/kubeagents/auth"
	"github.com/kubeagents/kubeagents/middleware"
	"github.com/kubeagents/kubeagents/models"
	"github.com/kubeagents/kubeagents/store"
)

const testUserIDUS3 = "test-user-123"
const testUserEmailUS3 = "test@example.com"

// addTestUserToContextUS3 adds a test user to the request context
func addTestUserToContextUS3(r *http.Request) *http.Request {
	claims := &auth.AccessTokenClaims{
		UserID: testUserIDUS3,
		Email:  testUserEmailUS3,
	}
	ctx := context.WithValue(r.Context(), middleware.UserContextKey, claims)
	return r.WithContext(ctx)
}

func setupTestStoreForUS3() store.Store {
	st := store.NewMemoryStore()
	now := time.Now()

	// Create test user
	user := &models.User{
		ID:            testUserIDUS3,
		Email:         testUserEmailUS3,
		Name:          "Test User",
		PasswordHash:  "dummy-hash",
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	st.CreateUser(user)

	// Create agent
	agent := &models.Agent{
		AgentID:    "agent-001",
		UserID:     testUserIDUS3, // Associate with test user
		Name:       "Test Agent",
		Source:     "test-software",
		Registered: now,
		LastSeen:   now,
	}
	st.CreateOrUpdateAgent(agent)

	// Create multiple sessions with different statuses
	sessions := []struct {
		topic   string
		status  string
		expired bool
	}{
		{"task-001", "running", false},
		{"task-002", "success", false},
		{"task-003", "failed", true},
	}

	for i, s := range sessions {
		session := &models.Session{
			AgentID:      "agent-001",
			SessionTopic: s.topic,
			Created:      now.Add(time.Duration(i) * time.Hour),
			LastUpdated:  now.Add(time.Duration(i) * time.Hour),
			Expired:      s.expired,
		}
		st.CreateOrUpdateSession(session)

		// Add status history
		status1 := &models.AgentStatus{
			AgentID:      "agent-001",
			SessionTopic: s.topic,
			Status:       "running",
			Timestamp:    now.Add(time.Duration(i) * time.Hour),
			Message:      "Task started",
		}
		st.AddStatus(status1)

		if s.status != "running" {
			status2 := &models.AgentStatus{
				AgentID:      "agent-001",
				SessionTopic: s.topic,
				Status:       s.status,
				Timestamp:    now.Add(time.Duration(i)*time.Hour + 30*time.Minute),
				Message:      "Task " + s.status,
			}
			st.AddStatus(status2)
		}
	}

	return st
}

func TestAgentHandler_GetAgent(t *testing.T) {
	st := setupTestStoreForUS3()
	handler := NewAgentHandler(st)

	req := httptest.NewRequest("GET", "/api/agents/agent-001", nil)
	req = addTestUserToContextUS3(req)
	rr := httptest.NewRecorder()

	// Set up chi context with route parameter
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agent_id", "agent-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetAgent(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("GetAgent() status = %v, want %v", status, http.StatusOK)
	}

	var response struct {
		models.Agent
		SessionCount       int    `json:"session_count"`
		ActiveSessionCount int    `json:"active_session_count"`
		LatestStatus       string `json:"latest_status,omitempty"`
		LatestMessage      string `json:"latest_message,omitempty"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("GetAgent() invalid JSON: %v", err)
	}

	if response.AgentID != "agent-001" {
		t.Errorf("GetAgent() agent_id = %v, want agent-001", response.AgentID)
	}
	if response.Name != "Test Agent" {
		t.Errorf("GetAgent() name = %v, want Test Agent", response.Name)
	}

	// Verify statistics are included
	if response.SessionCount != 3 {
		t.Errorf("GetAgent() session_count = %v, want 3", response.SessionCount)
	}
	if response.ActiveSessionCount != 2 {
		t.Errorf("GetAgent() active_session_count = %v, want 2", response.ActiveSessionCount)
	}

	// Verify latest_status is populated
	if response.LatestStatus == "" {
		t.Error("GetAgent() latest_status should not be empty")
	} else {
		validStatuses := map[string]bool{
			"running": true,
			"success": true,
			"failed":  true,
			"pending": true,
		}
		if !validStatuses[response.LatestStatus] {
			t.Errorf("GetAgent() invalid latest_status = %v", response.LatestStatus)
		}
	}
}

func TestAgentHandler_GetAgentNotFound(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewAgentHandler(st)

	req := httptest.NewRequest("GET", "/api/agents/agent-999", nil)
	req = addTestUserToContextUS3(req)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agent_id", "agent-999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetAgent(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("GetAgentNotFound() status = %v, want %v", status, http.StatusNotFound)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("GetAgentNotFound() invalid JSON: %v", err)
	}

	if response["error"] != "not_found" {
		t.Errorf("GetAgentNotFound() error = %v, want not_found", response["error"])
	}
}

func TestAgentHandler_ListSessions(t *testing.T) {
	st := setupTestStoreForUS3()
	handler := NewAgentHandler(st)

	req := httptest.NewRequest("GET", "/api/agents/agent-001/sessions", nil)
	req = addTestUserToContextUS3(req)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agent_id", "agent-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.ListSessions(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ListSessions() status = %v, want %v", status, http.StatusOK)
	}

	var response struct {
		Sessions []struct {
			models.Session
			CurrentStatus *string `json:"current_status,omitempty"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("ListSessions() invalid JSON: %v", err)
	}

	if len(response.Sessions) != 3 {
		t.Errorf("ListSessions() session count = %v, want 3", len(response.Sessions))
	}

	// Verify that current_status is populated
	for _, session := range response.Sessions {
		if session.CurrentStatus == nil {
			t.Errorf("ListSessions() current_status is nil for session %s", session.SessionTopic)
		} else {
			validStatuses := map[string]bool{
				"running": true,
				"success": true,
				"failed":  true,
				"pending": true,
			}
			if !validStatuses[*session.CurrentStatus] {
				t.Errorf("ListSessions() invalid current_status = %v for session %s", *session.CurrentStatus, session.SessionTopic)
			}
		}
	}
}

func TestAgentHandler_ListSessionsWithExpiredFilter(t *testing.T) {
	st := setupTestStoreForUS3()
	handler := NewAgentHandler(st)

	// Test excluding expired sessions
	req := httptest.NewRequest("GET", "/api/agents/agent-001/sessions?expired=false", nil)
	req = addTestUserToContextUS3(req)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agent_id", "agent-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.ListSessions(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ListSessionsWithExpiredFilter() status = %v, want %v", status, http.StatusOK)
	}

	var response struct {
		Sessions []struct {
			models.Session
			CurrentStatus *string `json:"current_status,omitempty"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("ListSessionsWithExpiredFilter() invalid JSON: %v", err)
	}

	// Should return only non-expired sessions
	if len(response.Sessions) != 2 {
		t.Errorf("ListSessionsWithExpiredFilter() session count = %v, want 2", len(response.Sessions))
	}

	// Verify that current_status is populated for non-expired sessions
	for _, session := range response.Sessions {
		if session.CurrentStatus == nil {
			t.Errorf("ListSessionsWithExpiredFilter() current_status is nil for session %s", session.SessionTopic)
		}
	}
}

func TestAgentHandler_GetSession(t *testing.T) {
	st := setupTestStoreForUS3()
	handler := NewAgentHandler(st)

	req := httptest.NewRequest("GET", "/api/agents/agent-001/sessions/task-001", nil)
	req = addTestUserToContextUS3(req)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agent_id", "agent-001")
	rctx.URLParams.Add("session_topic", "task-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetSession(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("GetSession() status = %v, want %v", status, http.StatusOK)
	}

	var response struct {
		Session       models.Session       `json:"session"`
		StatusHistory []models.AgentStatus `json:"status_history"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("GetSession() invalid JSON: %v", err)
	}

	if response.Session.SessionTopic != "task-001" {
		t.Errorf("GetSession() session_topic = %v, want task-001", response.Session.SessionTopic)
	}

	if len(response.StatusHistory) == 0 {
		t.Error("GetSession() status_history is empty")
	}
}

func TestAgentHandler_GetSessionNotFound(t *testing.T) {
	st := setupTestStoreForUS3()
	handler := NewAgentHandler(st)

	req := httptest.NewRequest("GET", "/api/agents/agent-001/sessions/task-999", nil)
	req = addTestUserToContextUS3(req)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agent_id", "agent-001")
	rctx.URLParams.Add("session_topic", "task-999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetSession(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("GetSessionNotFound() status = %v, want %v", status, http.StatusNotFound)
	}
}

func TestAgentHandler_GetAgentStatus(t *testing.T) {
	st := setupTestStoreForUS3()
	handler := NewAgentHandler(st)

	req := httptest.NewRequest("GET", "/api/agents/agent-001/status", nil)
	req = addTestUserToContextUS3(req)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agent_id", "agent-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetAgentStatus(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("GetAgentStatus() status = %v, want %v", status, http.StatusOK)
	}

	var status models.AgentStatus
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatalf("GetAgentStatus() invalid JSON: %v", err)
	}

	if status.AgentID != "agent-001" {
		t.Errorf("GetAgentStatus() agent_id = %v, want agent-001", status.AgentID)
	}
}

func TestAgentHandler_GetAgentDetailResponseTime(t *testing.T) {
	st := setupTestStoreForUS3()
	handler := NewAgentHandler(st)

	req := httptest.NewRequest("GET", "/api/agents/agent-001", nil)
	req = addTestUserToContextUS3(req)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agent_id", "agent-001")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	start := time.Now()
	handler.GetAgent(rr, req)
	duration := time.Since(start)

	if duration > 200*time.Millisecond {
		t.Errorf("GetAgentDetailResponseTime() response time = %v, want < 200ms", duration)
	}
}

func TestAgentHandler_StatusHistoryOrdering(t *testing.T) {
	st := setupTestStoreForUS3()
	handler := NewAgentHandler(st)

	req := httptest.NewRequest("GET", "/api/agents/agent-001/sessions/task-002", nil)
	req = addTestUserToContextUS3(req)
	rr := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agent_id", "agent-001")
	rctx.URLParams.Add("session_topic", "task-002")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetSession(rr, req)

	var response struct {
		Session       models.Session       `json:"session"`
		StatusHistory []models.AgentStatus `json:"status_history"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("StatusHistoryOrdering() invalid JSON: %v", err)
	}

	// Verify status history has at least 2 entries
	if len(response.StatusHistory) < 2 {
		t.Fatalf("StatusHistoryOrdering() status history count = %v, want >= 2", len(response.StatusHistory))
	}

	// Verify ordering (should be in descending order by timestamp)
	for i := 1; i < len(response.StatusHistory); i++ {
		if response.StatusHistory[i].Timestamp.After(response.StatusHistory[i-1].Timestamp) {
			t.Errorf("StatusHistoryOrdering() status history not in descending order")
		}
	}
}
