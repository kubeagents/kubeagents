package models

import (
	"testing"
	"time"
)

func TestAgent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		agent   Agent
		wantErr bool
	}{
		{
			name: "valid agent",
			agent: Agent{
				AgentID:    "agent-001",
				Name:       "Test Agent",
				Source:     "test-software",
				Registered: time.Now(),
				LastSeen:   time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing agent_id",
			agent: Agent{
				Name:       "Test Agent",
				Registered: time.Now(),
				LastSeen:   time.Now(),
			},
			wantErr: true,
		},
		{
			name: "agent_id too long",
			agent: Agent{
				AgentID:    string(make([]byte, 101)),
				Registered: time.Now(),
				LastSeen:   time.Now(),
			},
			wantErr: true,
		},
		{
			name: "name too long",
			agent: Agent{
				AgentID:    "agent-001",
				Name:       string(make([]byte, 201)),
				Registered: time.Now(),
				LastSeen:   time.Now(),
			},
			wantErr: true,
		},
		{
			name: "source too long",
			agent: Agent{
				AgentID:    "agent-001",
				Source:     string(make([]byte, 201)),
				Registered: time.Now(),
				LastSeen:   time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing registered time",
			agent: Agent{
				AgentID:  "agent-001",
				LastSeen: time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing last_seen time",
			agent: Agent{
				AgentID:    "agent-001",
				Registered: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSession_Validate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		session Session
		wantErr bool
	}{
		{
			name: "valid session",
			session: Session{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Created:      now,
				LastUpdated:  now,
				Expired:      false,
				TTLMinutes:   30,
			},
			wantErr: false,
		},
		{
			name: "missing agent_id",
			session: Session{
				SessionTopic: "task-001",
				Created:      now,
				LastUpdated:  now,
			},
			wantErr: true,
		},
		{
			name: "missing session_topic",
			session: Session{
				AgentID:     "agent-001",
				Created:     now,
				LastUpdated: now,
			},
			wantErr: true,
		},
		{
			name: "session_topic too long",
			session: Session{
				AgentID:      "agent-001",
				SessionTopic: string(make([]byte, 501)),
				Created:      now,
				LastUpdated:  now,
			},
			wantErr: true,
		},
		{
			name: "missing created time",
			session: Session{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				LastUpdated:  now,
			},
			wantErr: true,
		},
		{
			name: "missing last_updated time",
			session: Session{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Created:      now,
			},
			wantErr: true,
		},
		{
			name: "last_updated before created",
			session: Session{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Created:      now,
				LastUpdated:  now.Add(-time.Hour),
			},
			wantErr: true,
		},
		{
			name: "ttl_minutes out of range",
			session: Session{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Created:      now,
				LastUpdated:  now,
				TTLMinutes:   2000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Session.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgentStatus_Validate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name        string
		agentStatus AgentStatus
		wantErr     bool
	}{
		{
			name: "valid status",
			agentStatus: AgentStatus{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "running",
				Timestamp:    now,
				Message:      "Task started",
			},
			wantErr: false,
		},
		{
			name: "missing agent_id",
			agentStatus: AgentStatus{
				SessionTopic: "task-001",
				Status:       "running",
				Timestamp:    now,
			},
			wantErr: true,
		},
		{
			name: "missing session_topic",
			agentStatus: AgentStatus{
				AgentID:   "agent-001",
				Status:    "running",
				Timestamp: now,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			agentStatus: AgentStatus{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "invalid",
				Timestamp:    now,
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			agentStatus: AgentStatus{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "running",
			},
			wantErr: true,
		},
		{
			name: "message too long",
			agentStatus: AgentStatus{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "running",
				Timestamp:    now,
				Message:      string(make([]byte, 1001)),
			},
			wantErr: true,
		},
		{
			name: "content too long",
			agentStatus: AgentStatus{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "running",
				Timestamp:    now,
				Content:      string(make([]byte, 10001)),
			},
			wantErr: true,
		},
		{
			name: "valid status values",
			agentStatus: AgentStatus{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "success",
				Timestamp:    now,
			},
			wantErr: false,
		},
		{
			name: "valid status failed",
			agentStatus: AgentStatus{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "failed",
				Timestamp:    now,
			},
			wantErr: false,
		},
		{
			name: "valid status pending",
			agentStatus: AgentStatus{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "pending",
				Timestamp:    now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agentStatus.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("AgentStatus.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
