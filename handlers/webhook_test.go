package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kubeagents/kubeagents/auth"
	"github.com/kubeagents/kubeagents/middleware"
	"github.com/kubeagents/kubeagents/notifier"
	"github.com/kubeagents/kubeagents/store"
)

const testUserIDWebhook = "test-user-123"
const testUserEmailWebhook = "test@example.com"

// addTestUserToContextWebhook adds a test user to the request context
func addTestUserToContextWebhook(r *http.Request) *http.Request {
	claims := &auth.AccessTokenClaims{
		UserID: testUserIDWebhook,
		Email:  testUserEmailWebhook,
	}
	ctx := context.WithValue(r.Context(), middleware.UserContextKey, claims)
	return r.WithContext(ctx)
}

func TestWebhookHandler_NewAgentAutoRegistration(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)

	now := time.Now()
	reqBody := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Test Agent",
		"agent_source":  "test-software",
		"session_topic": "task-001",
		"status":        "running",
		"timestamp":     now.Format(time.RFC3339),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = addTestUserToContextWebhook(req)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("NewAgentAutoRegistration() status = %v, want %v", status, http.StatusOK)
	}
	
	// Verify agent was created
	agent, err := st.GetAgent("agent-001")
	if err != nil {
		t.Fatalf("NewAgentAutoRegistration() agent not created: %v", err)
	}
	if agent.Name != "Test Agent" {
		t.Errorf("NewAgentAutoRegistration() agent name = %v, want Test Agent", agent.Name)
	}
	if agent.Source != "test-software" {
		t.Errorf("NewAgentAutoRegistration() agent source = %v, want test-software", agent.Source)
	}
}

func TestWebhookHandler_ExistingAgentStatusUpdate(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)

	now := time.Now()

	// First request - create agent
	reqBody1 := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Test Agent",
		"session_topic": "task-001",
		"status":        "running",
		"timestamp":     now.Format(time.RFC3339),
	}
	body1, _ := json.Marshal(reqBody1)
	req1 := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1 = addTestUserToContextWebhook(req1)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	// Second request - update agent
	reqBody2 := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Updated Agent",
		"session_topic": "task-002",
		"status":        "running",
		"timestamp":     now.Add(time.Hour).Format(time.RFC3339),
	}
	body2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2 = addTestUserToContextWebhook(req2)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	
	// Verify agent was updated
	agent, err := st.GetAgent("agent-001")
	if err != nil {
		t.Fatalf("ExistingAgentStatusUpdate() agent not found: %v", err)
	}
	if agent.Name != "Updated Agent" {
		t.Errorf("ExistingAgentStatusUpdate() agent name = %v, want Updated Agent", agent.Name)
	}
	if agent.LastSeen.Before(now) {
		t.Errorf("ExistingAgentStatusUpdate() last_seen not updated")
	}
}

func TestWebhookHandler_SessionAutoCreation(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)

	now := time.Now()
	reqBody := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Test Agent",
		"session_topic": "task-001",
		"status":        "running",
		"timestamp":     now.Format(time.RFC3339),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = addTestUserToContextWebhook(req)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("SessionAutoCreation() status = %v, want %v", status, http.StatusOK)
	}
	
	// Verify session was created
	session, err := st.GetSession("agent-001", "task-001")
	if err != nil {
		t.Fatalf("SessionAutoCreation() session not created: %v", err)
	}
	if session.SessionTopic != "task-001" {
		t.Errorf("SessionAutoCreation() session_topic = %v, want task-001", session.SessionTopic)
	}
	if session.Expired {
		t.Errorf("SessionAutoCreation() session should not be expired")
	}
}

func TestWebhookHandler_SessionUpdateOnTaskEnd(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)

	now := time.Now()

	// First request - create session with running status
	reqBody1 := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Test Agent",
		"session_topic": "task-001",
		"status":        "running",
		"timestamp":     now.Format(time.RFC3339),
	}
	body1, _ := json.Marshal(reqBody1)
	req1 := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1 = addTestUserToContextWebhook(req1)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	// Second request - update session with success status
	reqBody2 := map[string]interface{}{
		"agent_id":      "agent-001",
		"session_topic": "task-001",
		"status":        "success",
		"timestamp":     now.Add(time.Hour).Format(time.RFC3339),
		"message":       "Task completed",
	}
	body2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2 = addTestUserToContextWebhook(req2)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	
	// Verify session was updated
	session, err := st.GetSession("agent-001", "task-001")
	if err != nil {
		t.Fatalf("SessionUpdateOnTaskEnd() session not found: %v", err)
	}
	// Verify last_updated is after the first request time
	if session.LastUpdated.Before(now) {
		t.Errorf("SessionUpdateOnTaskEnd() last_updated not updated, got %v, want >= %v", session.LastUpdated, now)
	}
	
	// Verify status history
	history, err := st.GetStatusHistory("agent-001", "task-001")
	if err != nil {
		t.Fatalf("SessionUpdateOnTaskEnd() failed to get status history: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("SessionUpdateOnTaskEnd() status history count = %v, want 2", len(history))
	}
	
	// Verify latest status is success
	latest, err := st.GetLatestStatus("agent-001", "task-001")
	if err != nil {
		t.Fatalf("SessionUpdateOnTaskEnd() failed to get latest status: %v", err)
	}
	if latest.Status != "success" {
		t.Errorf("SessionUpdateOnTaskEnd() latest status = %v, want success", latest.Status)
	}
}

func TestWebhookHandler_StatusHistoryRecording(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)

	now := time.Now()

	// Send multiple status reports
	statuses := []string{"running", "running", "success"}
	for i, status := range statuses {
		reqBody := map[string]interface{}{
			"agent_id":      "agent-001",
			"agent_name":    "Test Agent",
			"session_topic": "task-001",
			"status":        status,
			"timestamp":     now.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = addTestUserToContextWebhook(req)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
	
	// Verify status history
	history, err := st.GetStatusHistory("agent-001", "task-001")
	if err != nil {
		t.Fatalf("StatusHistoryRecording() failed to get status history: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("StatusHistoryRecording() status history count = %v, want 3", len(history))
	}
}

func TestWebhookHandler_InvalidStatusReportData(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)
	
	tests := []struct {
		name    string
		reqBody map[string]interface{}
		wantStatus int
	}{
		{
			name: "missing agent_id",
			reqBody: map[string]interface{}{
				"session_topic": "task-001",
				"status":         "running",
				"timestamp":      time.Now().Format(time.RFC3339),
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing session_topic",
			reqBody: map[string]interface{}{
				"agent_id": "agent-001",
				"status":    "running",
				"timestamp": time.Now().Format(time.RFC3339),
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid status",
			reqBody: map[string]interface{}{
				"agent_id":      "agent-001",
				"session_topic": "task-001",
				"status":         "invalid",
				"timestamp":      time.Now().Format(time.RFC3339),
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing timestamp",
			reqBody: map[string]interface{}{
				"agent_id":      "agent-001",
				"session_topic": "task-001",
				"status":         "running",
			},
			wantStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = addTestUserToContextWebhook(req)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			
			if status := rr.Code; status != tt.wantStatus {
				t.Errorf("InvalidStatusReportData(%s) status = %v, want %v", tt.name, status, tt.wantStatus)
			}
		})
	}
}

func TestWebhookHandler_ConcurrentStatusReports(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)
	
	now := time.Now()
	done := make(chan bool, 10)
	
	// Send 10 concurrent requests
	for i := 0; i < 10; i++ {
		go func(id int) {
			reqBody := map[string]interface{}{
				"agent_id":      "agent-001",
				"agent_name":    "Test Agent",
				"session_topic": "task-" + string(rune('0'+id)),
				"status":        "running",
				"timestamp":     now.Format(time.RFC3339),
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = addTestUserToContextWebhook(req)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			done <- true
		}(i)
	}
	
	// Wait for all requests
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify all sessions were created
	sessions := st.ListSessions("agent-001", true)
	if len(sessions) != 10 {
		t.Errorf("ConcurrentStatusReports() session count = %v, want 10", len(sessions))
	}
}

func TestWebhookHandler_ResponseTimeRequirement(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)

	now := time.Now()
	reqBody := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Test Agent",
		"session_topic": "task-001",
		"status":        "running",
		"timestamp":     now.Format(time.RFC3339),
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = addTestUserToContextWebhook(req)
	rr := httptest.NewRecorder()
	
	start := time.Now()
	handler.ServeHTTP(rr, req)
	duration := time.Since(start)
	
	if duration > 1*time.Second {
		t.Errorf("ResponseTimeRequirement() response time = %v, want < 1s", duration)
	}
}

func TestWebhookHandler_OptionalFields(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)

	now := time.Now()
	reqBody := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Test Agent",
		"session_topic": "task-001",
		"status":        "running",
		"timestamp":     now.Format(time.RFC3339),
		"message":       "Optional message",
		"content":       "Optional content",
		"ttl_minutes":   60,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = addTestUserToContextWebhook(req)
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("OptionalFields() status = %v, want %v", status, http.StatusOK)
	}
	
	// Verify optional fields were stored
	history, err := st.GetStatusHistory("agent-001", "task-001")
	if err != nil {
		t.Fatalf("OptionalFields() failed to get status history: %v", err)
	}
	if len(history) == 0 {
		t.Fatal("OptionalFields() no status history")
	}
	if history[0].Message != "Optional message" {
		t.Errorf("OptionalFields() message = %v, want Optional message", history[0].Message)
	}
	if history[0].Content != "Optional content" {
		t.Errorf("OptionalFields() content = %v, want Optional content", history[0].Content)
	}
	
	// Verify TTL was set
	session, err := st.GetSession("agent-001", "task-001")
	if err != nil {
		t.Fatalf("OptionalFields() session not found: %v", err)
	}
	if session.TTLMinutes != 60 {
		t.Errorf("OptionalFields() ttl_minutes = %v, want 60", session.TTLMinutes)
	}
}

func TestWebhookHandler_SessionExpirationTimeConfiguration(t *testing.T) {
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st)

	now := time.Now()

	// Create session with custom TTL
	reqBody := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Test Agent",
		"session_topic": "task-001",
		"status":        "running",
		"timestamp":     now.Format(time.RFC3339),
		"ttl_minutes":   120,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = addTestUserToContextWebhook(req)
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	// Verify TTL was set
	session, err := st.GetSession("agent-001", "task-001")
	if err != nil {
		t.Fatalf("SessionExpirationTimeConfiguration() session not found: %v", err)
	}
	if session.TTLMinutes != 120 {
		t.Errorf("SessionExpirationTimeConfiguration() ttl_minutes = %v, want 120", session.TTLMinutes)
	}
	
	// Verify expiration time calculation
	expectedExpiry := session.LastUpdated.Add(120 * time.Minute)
	if session.ExpiredAt != nil && !session.ExpiredAt.Equal(expectedExpiry) {
		t.Logf("Note: expired_at will be set when session expires")
	}
}

func TestWebhookHandler_StatusTransitionNotification_RunningToSuccess(t *testing.T) {
	// Mock notification server
	var notificationReceived atomic.Bool
	var receivedPayload notifier.WebhookPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notificationReceived.Store(true)
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create store and handler with notifier
	st := store.NewMemoryStore()
	nm := notifier.NewNotificationManager(server.URL, 5*time.Second)
	handler := NewWebhookHandlerWithNotifier(st, nm)

	now := time.Now()

	// First request: running status
	reqBody1 := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Test Agent",
		"session_topic": "task-001",
		"status":        "running",
		"timestamp":     now.Format(time.RFC3339),
	}
	body1, _ := json.Marshal(reqBody1)
	req1 := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1 = addTestUserToContextWebhook(req1)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Errorf("First request status = %v, want %v", rr1.Code, http.StatusOK)
	}

	time.Sleep(100 * time.Millisecond) // Wait for async
	if notificationReceived.Load() {
		t.Error("No notification should be sent for running status")
	}

	// Second request: success status (transition)
	reqBody2 := map[string]interface{}{
		"agent_id":      "agent-001",
		"agent_name":    "Test Agent",
		"session_topic": "task-001",
		"status":        "success",
		"timestamp":     now.Add(time.Minute).Format(time.RFC3339),
		"message":       "Task completed",
		"content":       "Result: OK",
	}
	body2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2 = addTestUserToContextWebhook(req2)
	rr2 := httptest.NewRecorder()

	start := time.Now()
	handler.ServeHTTP(rr2, req2)
	duration := time.Since(start)

	// Should return immediately (non-blocking)
	if rr2.Code != http.StatusOK {
		t.Errorf("Second request status = %v, want %v", rr2.Code, http.StatusOK)
	}

	if duration > 100*time.Millisecond {
		t.Errorf("Response time = %v, want < 100ms (should be non-blocking)", duration)
	}

	// Wait for async notification
	time.Sleep(200 * time.Millisecond)

	// Verify notification was sent
	if !notificationReceived.Load() {
		t.Fatal("Notification should be sent for running → success transition")
	}

	if receivedPayload.MsgType != "text" {
		t.Errorf("Payload msg_type = %v, want text", receivedPayload.MsgType)
	}

	// Verify notification content
	text := receivedPayload.Content.Text
	expectedSubstrings := []string{
		"agent-001",
		"task-001",
		"running → success",
		"Task completed",
		"Result: OK",
	}

	for _, substr := range expectedSubstrings {
		if !bytes.Contains([]byte(text), []byte(substr)) {
			t.Errorf("Notification text missing %q\nGot: %s", substr, text)
		}
	}
}

func TestWebhookHandler_StatusTransitionNotification_RunningToFailed(t *testing.T) {
	// Mock notification server
	var notificationReceived atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notificationReceived.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	st := store.NewMemoryStore()
	nm := notifier.NewNotificationManager(server.URL, 5*time.Second)
	handler := NewWebhookHandlerWithNotifier(st, nm)

	now := time.Now()

	// running → failed transition
	sendStatus(t, handler, "agent-001", "task-001", "running", now, "", "")
	sendStatus(t, handler, "agent-001", "task-001", "failed", now.Add(time.Minute), "Task failed", "Error: timeout")

	time.Sleep(200 * time.Millisecond)

	if !notificationReceived.Load() {
		t.Fatal("Notification should be sent for running → failed transition")
	}
}

func TestWebhookHandler_NoNotificationForNonRunningTransition(t *testing.T) {
	// Transition from pending → running should NOT trigger notification
	var notificationReceived atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		notificationReceived.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	st := store.NewMemoryStore()
	nm := notifier.NewNotificationManager(server.URL, 5*time.Second)
	handler := NewWebhookHandlerWithNotifier(st, nm)

	now := time.Now()

	// pending → running (should not notify)
	sendStatus(t, handler, "agent-001", "task-001", "pending", now, "", "")
	sendStatus(t, handler, "agent-001", "task-001", "running", now.Add(time.Second), "", "")

	time.Sleep(200 * time.Millisecond)

	if notificationReceived.Load() {
		t.Error("No notification should be sent for pending → running transition")
	}

	// success → running (should not notify, but weird)
	sendStatus(t, handler, "agent-002", "task-002", "success", now, "", "")
	sendStatus(t, handler, "agent-002", "task-002", "running", now.Add(time.Second), "", "")

	time.Sleep(200 * time.Millisecond)

	if notificationReceived.Load() {
		t.Error("No notification should be sent for success → running transition")
	}
}

func TestWebhookHandler_NotificationFailureDoesNotBlockResponse(t *testing.T) {
	// Even if notification fails, webhook should respond with 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // Fail
	}))
	defer server.Close()

	st := store.NewMemoryStore()
	nm := notifier.NewNotificationManager(server.URL, 5*time.Second)
	handler := NewWebhookHandlerWithNotifier(st, nm)

	now := time.Now()

	sendStatus(t, handler, "agent-001", "task-001", "running", now, "", "")

	start := time.Now()
	rr := sendStatusWithResult(t, handler, "agent-001", "task-001", "success", now.Add(time.Second), "", "")
	duration := time.Since(start)

	// Should respond immediately
	if rr.Code != http.StatusOK {
		t.Errorf("Response status = %v, want %v", rr.Code, http.StatusOK)
	}

	if duration > 100*time.Millisecond {
		t.Errorf("Response time = %v, want < 100ms", duration)
	}
}

func TestWebhookHandler_NoNotificationWhenNotifierIsNil(t *testing.T) {
	// Handler without notifier should work normally
	st := store.NewMemoryStore()
	handler := NewWebhookHandler(st) // No notifier

	now := time.Now()

	// running → success (no crash, no notification)
	rr1 := sendStatusWithResult(t, handler, "agent-001", "task-001", "running", now, "", "")
	if rr1.Code != http.StatusOK {
		t.Errorf("First request status = %v, want %v", rr1.Code, http.StatusOK)
	}

	rr2 := sendStatusWithResult(t, handler, "agent-001", "task-001", "success", now.Add(time.Second), "Done", "")
	if rr2.Code != http.StatusOK {
		t.Errorf("Second request status = %v, want %v", rr2.Code, http.StatusOK)
	}

	// Should not crash or error
}

// Helper function to send status
func sendStatus(t *testing.T, handler *WebhookHandler, agentID, sessionTopic, status string, timestamp time.Time, message, content string) {
	reqBody := map[string]interface{}{
		"agent_id":      agentID,
		"agent_name":    "Test Agent",
		"session_topic": sessionTopic,
		"status":        status,
		"timestamp":     timestamp.Format(time.RFC3339),
	}
	if message != "" {
		reqBody["message"] = message
	}
	if content != "" {
		reqBody["content"] = content
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = addTestUserToContextWebhook(req)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("sendStatus() status = %v, want %v", rr.Code, http.StatusOK)
	}
}

// Helper function to send status and return response
func sendStatusWithResult(t *testing.T, handler *WebhookHandler, agentID, sessionTopic, status string, timestamp time.Time, message, content string) *httptest.ResponseRecorder {
	reqBody := map[string]interface{}{
		"agent_id":      agentID,
		"agent_name":    "Test Agent",
		"session_topic": sessionTopic,
		"status":        status,
		"timestamp":     timestamp.Format(time.RFC3339),
	}
	if message != "" {
		reqBody["message"] = message
	}
	if content != "" {
		reqBody["content"] = content
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/webhook/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = addTestUserToContextWebhook(req)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}
