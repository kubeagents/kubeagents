package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kubeagents/kubeagents/models"
	"github.com/kubeagents/kubeagents/store"
)

func setupTestStoreWithAgents() *store.Store {
	st := store.NewStore()
	now := time.Now()
	
	// Create agents with sessions
	for i := 1; i <= 3; i++ {
		agentID := fmt.Sprintf("agent-%03d", i)
		agentName := fmt.Sprintf("Agent %d", i)
		agent := &models.Agent{
			AgentID:    agentID,
			Name:       agentName,
			Source:     "test-software",
			Registered: now,
			LastSeen:   now,
		}
		st.CreateOrUpdateAgent(agent)
		
		// Create sessions
		for j := 1; j <= 2; j++ {
			sessionTopic := fmt.Sprintf("task-%03d", j)
			session := &models.Session{
				AgentID:      agentID,
				SessionTopic: sessionTopic,
				Created:      now,
				LastUpdated:  now,
				Expired:      false,
			}
			st.CreateOrUpdateSession(session)
			
			// Add status
			status := &models.AgentStatus{
				AgentID:      agentID,
				SessionTopic: sessionTopic,
				Status:       "running",
				Timestamp:    now,
			}
			st.AddStatus(status)
		}
	}
	
	return st
}

func TestAgentHandler_ListAgents(t *testing.T) {
	st := setupTestStoreWithAgents()
	handler := NewAgentHandler(st)
	
	req := httptest.NewRequest("GET", "/api/agents", nil)
	rr := httptest.NewRecorder()
	
	handler.ListAgents(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ListAgents() status = %v, want %v", status, http.StatusOK)
	}
	
	var response struct {
		Agents []interface{} `json:"agents"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("ListAgents() invalid JSON: %v", err)
	}
	
	if len(response.Agents) != 3 {
		t.Errorf("ListAgents() agent count = %v, want 3", len(response.Agents))
	}
}

func TestAgentHandler_ListAgentsWithStatusFilter(t *testing.T) {
	st := setupTestStoreWithAgents()
	handler := NewAgentHandler(st)
	
	// Test with status filter
	req := httptest.NewRequest("GET", "/api/agents?status=running", nil)
	rr := httptest.NewRecorder()
	
	handler.ListAgents(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ListAgentsWithStatusFilter() status = %v, want %v", status, http.StatusOK)
	}
	
	var response struct {
		Agents []interface{} `json:"agents"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("ListAgentsWithStatusFilter() invalid JSON: %v", err)
	}
	
	// All agents should have running status
	if len(response.Agents) != 3 {
		t.Errorf("ListAgentsWithStatusFilter() agent count = %v, want 3", len(response.Agents))
	}
}

func TestAgentHandler_ListAgentsWithSearch(t *testing.T) {
	st := setupTestStoreWithAgents()
	handler := NewAgentHandler(st)
	
	// Test with search parameter (search by agent ID)
	req := httptest.NewRequest("GET", "/api/agents?search=agent-001", nil)
	rr := httptest.NewRecorder()
	
	handler.ListAgents(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ListAgentsWithSearch() status = %v, want %v", status, http.StatusOK)
	}
	
	var response struct {
		Agents []interface{} `json:"agents"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("ListAgentsWithSearch() invalid JSON: %v", err)
	}
	
	// Should return at least one agent matching search
	if len(response.Agents) < 1 {
		t.Errorf("ListAgentsWithSearch() agent count = %v, want >= 1", len(response.Agents))
	}
}

func TestAgentHandler_ListAgentsWithSessionStatistics(t *testing.T) {
	st := setupTestStoreWithAgents()
	handler := NewAgentHandler(st)
	
	req := httptest.NewRequest("GET", "/api/agents", nil)
	rr := httptest.NewRecorder()
	
	handler.ListAgents(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ListAgentsWithSessionStatistics() status = %v, want %v", status, http.StatusOK)
	}
	
	var response struct {
		Agents []map[string]interface{} `json:"agents"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("ListAgentsWithSessionStatistics() invalid JSON: %v", err)
	}
	
	// Verify session statistics are included
	if len(response.Agents) > 0 {
		agent := response.Agents[0]
		if _, ok := agent["session_count"]; !ok {
			t.Error("ListAgentsWithSessionStatistics() missing session_count")
		}
		if _, ok := agent["active_session_count"]; !ok {
			t.Error("ListAgentsWithSessionStatistics() missing active_session_count")
		}
	}
}

func TestAgentHandler_ListAgentsResponseTime(t *testing.T) {
	st := setupTestStoreWithAgents()
	handler := NewAgentHandler(st)
	
	req := httptest.NewRequest("GET", "/api/agents", nil)
	rr := httptest.NewRecorder()
	
	start := time.Now()
	handler.ListAgents(rr, req)
	duration := time.Since(start)
	
	if duration > 500*time.Millisecond {
		t.Errorf("ListAgentsResponseTime() response time = %v, want < 500ms", duration)
	}
}

func TestAgentHandler_ListAgentsEmpty(t *testing.T) {
	st := store.NewStore()
	handler := NewAgentHandler(st)
	
	req := httptest.NewRequest("GET", "/api/agents", nil)
	rr := httptest.NewRecorder()
	
	handler.ListAgents(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("ListAgentsEmpty() status = %v, want %v", status, http.StatusOK)
	}
	
	var response struct {
		Agents []interface{} `json:"agents"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("ListAgentsEmpty() invalid JSON: %v", err)
	}
	
	if len(response.Agents) != 0 {
		t.Errorf("ListAgentsEmpty() agent count = %v, want 0", len(response.Agents))
	}
}
