package store

import (
	"testing"
	"time"

	"github.com/kubeagents/kubeagents/models"
)

func TestStore_CreateOrUpdateAgent(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	agent := &models.Agent{
		AgentID:    "agent-001",
		Name:       "Test Agent",
		Source:     "test-software",
		Registered: now,
		LastSeen:   now,
	}
	
	err := s.CreateOrUpdateAgent(agent)
	if err != nil {
		t.Fatalf("CreateOrUpdateAgent() error = %v, want nil", err)
	}
	
	// Test update
	agent.Name = "Updated Agent"
	err = s.CreateOrUpdateAgent(agent)
	if err != nil {
		t.Fatalf("CreateOrUpdateAgent() update error = %v, want nil", err)
	}
	
	retrieved, err := s.GetAgent("agent-001")
	if err != nil {
		t.Fatalf("GetAgent() error = %v, want nil", err)
	}
	if retrieved.Name != "Updated Agent" {
		t.Errorf("GetAgent() name = %v, want Updated Agent", retrieved.Name)
	}
}

func TestStore_GetAgent(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	agent := &models.Agent{
		AgentID:    "agent-001",
		Name:       "Test Agent",
		Registered: now,
		LastSeen:   now,
	}
	s.CreateOrUpdateAgent(agent)
	
	// Test existing agent
	retrieved, err := s.GetAgent("agent-001")
	if err != nil {
		t.Fatalf("GetAgent() error = %v, want nil", err)
	}
	if retrieved.AgentID != "agent-001" {
		t.Errorf("GetAgent() agent_id = %v, want agent-001", retrieved.AgentID)
	}
	
	// Test non-existing agent
	_, err = s.GetAgent("agent-999")
	if err != ErrNotFound {
		t.Errorf("GetAgent() error = %v, want ErrNotFound", err)
	}
}

func TestStore_ListAgents(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	// Add multiple agents
	for i := 1; i <= 3; i++ {
		agent := &models.Agent{
			AgentID:    "agent-00" + string(rune('0'+i)),
			Name:       "Agent " + string(rune('0'+i)),
			Registered: now,
			LastSeen:   now,
		}
		s.CreateOrUpdateAgent(agent)
	}
	
	agents := s.ListAgents()
	if len(agents) != 3 {
		t.Errorf("ListAgents() count = %v, want 3", len(agents))
	}
}

func TestStore_CreateOrUpdateSession(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	// Create agent first
	agent := &models.Agent{
		AgentID:    "agent-001",
		Registered: now,
		LastSeen:   now,
	}
	s.CreateOrUpdateAgent(agent)
	
	session := &models.Session{
		AgentID:      "agent-001",
		SessionTopic: "task-001",
		Created:      now,
		LastUpdated:  now,
		Expired:      false,
		TTLMinutes:   30,
	}
	
	err := s.CreateOrUpdateSession(session)
	if err != nil {
		t.Fatalf("CreateOrUpdateSession() error = %v, want nil", err)
	}
	
	// Test update
	session.LastUpdated = now.Add(time.Hour)
	err = s.CreateOrUpdateSession(session)
	if err != nil {
		t.Fatalf("CreateOrUpdateSession() update error = %v, want nil", err)
	}
	
	retrieved, err := s.GetSession("agent-001", "task-001")
	if err != nil {
		t.Fatalf("GetSession() error = %v, want nil", err)
	}
	if retrieved.SessionTopic != "task-001" {
		t.Errorf("GetSession() session_topic = %v, want task-001", retrieved.SessionTopic)
	}
}

func TestStore_GetSession(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	agent := &models.Agent{
		AgentID:    "agent-001",
		Registered: now,
		LastSeen:   now,
	}
	s.CreateOrUpdateAgent(agent)
	
	session := &models.Session{
		AgentID:      "agent-001",
		SessionTopic: "task-001",
		Created:      now,
		LastUpdated:  now,
		Expired:      false,
	}
	s.CreateOrUpdateSession(session)
	
	// Test existing session
	retrieved, err := s.GetSession("agent-001", "task-001")
	if err != nil {
		t.Fatalf("GetSession() error = %v, want nil", err)
	}
	if retrieved.SessionTopic != "task-001" {
		t.Errorf("GetSession() session_topic = %v, want task-001", retrieved.SessionTopic)
	}
	
	// Test non-existing session
	_, err = s.GetSession("agent-001", "task-999")
	if err != ErrNotFound {
		t.Errorf("GetSession() error = %v, want ErrNotFound", err)
	}
	
	// Test non-existing agent
	_, err = s.GetSession("agent-999", "task-001")
	if err != ErrNotFound {
		t.Errorf("GetSession() error = %v, want ErrNotFound", err)
	}
}

func TestStore_ListSessions(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	agent := &models.Agent{
		AgentID:    "agent-001",
		Registered: now,
		LastSeen:   now,
	}
	s.CreateOrUpdateAgent(agent)
	
	// Add multiple sessions
	for i := 1; i <= 3; i++ {
		session := &models.Session{
			AgentID:      "agent-001",
			SessionTopic: "task-00" + string(rune('0'+i)),
			Created:      now,
			LastUpdated:  now,
			Expired:      i == 3, // Third session is expired
		}
		s.CreateOrUpdateSession(session)
	}
	
	// Test include expired
	sessions := s.ListSessions("agent-001", true)
	if len(sessions) != 3 {
		t.Errorf("ListSessions(includeExpired=true) count = %v, want 3", len(sessions))
	}
	
	// Test exclude expired
	sessions = s.ListSessions("agent-001", false)
	if len(sessions) != 2 {
		t.Errorf("ListSessions(includeExpired=false) count = %v, want 2", len(sessions))
	}
}

func TestStore_AddStatus(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	// Create agent and session first
	agent := &models.Agent{
		AgentID:    "agent-001",
		Registered: now,
		LastSeen:   now,
	}
	s.CreateOrUpdateAgent(agent)
	
	session := &models.Session{
		AgentID:      "agent-001",
		SessionTopic: "task-001",
		Created:      now,
		LastUpdated:  now,
		Expired:      false,
	}
	s.CreateOrUpdateSession(session)
	
	status := &models.AgentStatus{
		AgentID:      "agent-001",
		SessionTopic: "task-001",
		Status:       "running",
		Timestamp:    now,
		Message:      "Task started",
	}
	
	err := s.AddStatus(status)
	if err != nil {
		t.Fatalf("AddStatus() error = %v, want nil", err)
	}
	
	history, err := s.GetStatusHistory("agent-001", "task-001")
	if err != nil {
		t.Fatalf("GetStatusHistory() error = %v, want nil", err)
	}
	if len(history) != 1 {
		t.Errorf("GetStatusHistory() count = %v, want 1", len(history))
	}
	if history[0].Status != "running" {
		t.Errorf("GetStatusHistory() status = %v, want running", history[0].Status)
	}
}

func TestStore_GetStatusHistory(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	agent := &models.Agent{
		AgentID:    "agent-001",
		Registered: now,
		LastSeen:   now,
	}
	s.CreateOrUpdateAgent(agent)
	
	session := &models.Session{
		AgentID:      "agent-001",
		SessionTopic: "task-001",
		Created:      now,
		LastUpdated:  now,
		Expired:      false,
	}
	s.CreateOrUpdateSession(session)
	
	// Add multiple statuses
	statuses := []*models.AgentStatus{
		{
			AgentID:      "agent-001",
			SessionTopic: "task-001",
			Status:       "running",
			Timestamp:    now,
		},
		{
			AgentID:      "agent-001",
			SessionTopic: "task-001",
			Status:       "success",
			Timestamp:    now.Add(time.Hour),
		},
	}
	
	for _, status := range statuses {
		s.AddStatus(status)
	}
	
	history, err := s.GetStatusHistory("agent-001", "task-001")
	if err != nil {
		t.Fatalf("GetStatusHistory() error = %v, want nil", err)
	}
	if len(history) != 2 {
		t.Errorf("GetStatusHistory() count = %v, want 2", len(history))
	}
}

func TestStore_GetLatestStatus(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	agent := &models.Agent{
		AgentID:    "agent-001",
		Registered: now,
		LastSeen:   now,
	}
	s.CreateOrUpdateAgent(agent)
	
	session := &models.Session{
		AgentID:      "agent-001",
		SessionTopic: "task-001",
		Created:      now,
		LastUpdated:  now,
		Expired:      false,
	}
	s.CreateOrUpdateSession(session)
	
	// Add statuses with different timestamps
	s.AddStatus(&models.AgentStatus{
		AgentID:      "agent-001",
		SessionTopic: "task-001",
		Status:       "running",
		Timestamp:    now,
	})
	
	s.AddStatus(&models.AgentStatus{
		AgentID:      "agent-001",
		SessionTopic: "task-001",
		Status:       "success",
		Timestamp:    now.Add(time.Hour),
	})
	
	latest, err := s.GetLatestStatus("agent-001", "task-001")
	if err != nil {
		t.Fatalf("GetLatestStatus() error = %v, want nil", err)
	}
	if latest.Status != "success" {
		t.Errorf("GetLatestStatus() status = %v, want success", latest.Status)
	}
}

func TestStore_CheckExpiredSessions(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	agent := &models.Agent{
		AgentID:    "agent-001",
		Registered: now,
		LastSeen:   now,
	}
	s.CreateOrUpdateAgent(agent)
	
	// Create expired session (last updated 1 hour ago, TTL 30 minutes)
	expiredSession := &models.Session{
		AgentID:      "agent-001",
		SessionTopic: "task-expired",
		Created:      now.Add(-2 * time.Hour),
		LastUpdated:  now.Add(-1 * time.Hour),
		Expired:      false,
		TTLMinutes:   30,
	}
	s.CreateOrUpdateSession(expiredSession)
	
	// Create active session
	activeSession := &models.Session{
		AgentID:      "agent-001",
		SessionTopic: "task-active",
		Created:      now,
		LastUpdated:  now,
		Expired:      false,
		TTLMinutes:   30,
	}
	s.CreateOrUpdateSession(activeSession)
	
	// Check expired sessions
	s.CheckExpiredSessions()
	
	// Verify expired session is marked
	expired, _ := s.GetSession("agent-001", "task-expired")
	if !expired.Expired {
		t.Errorf("CheckExpiredSessions() expired session not marked as expired")
	}
	if expired.ExpiredAt == nil {
		t.Errorf("CheckExpiredSessions() expired_at not set")
	}
	
	// Verify active session is not marked
	active, _ := s.GetSession("agent-001", "task-active")
	if active.Expired {
		t.Errorf("CheckExpiredSessions() active session incorrectly marked as expired")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewStore()
	now := time.Now()
	
	// Create agent
	agent := &models.Agent{
		AgentID:    "agent-001",
		Registered: now,
		LastSeen:   now,
	}
	s.CreateOrUpdateAgent(agent)
	
	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			session := &models.Session{
				AgentID:      "agent-001",
				SessionTopic: "task-" + string(rune('0'+id)),
				Created:      now,
				LastUpdated:  now,
				Expired:      false,
			}
			s.CreateOrUpdateSession(session)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify all sessions were created
	sessions := s.ListSessions("agent-001", true)
	if len(sessions) != 10 {
		t.Errorf("ConcurrentAccess() session count = %v, want 10", len(sessions))
	}
}
