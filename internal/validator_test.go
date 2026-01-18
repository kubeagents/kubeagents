package internal

import (
	"testing"
	"time"
)

func TestStatusReport_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		report  StatusReport
		wantErr bool
	}{
		{
			name: "valid report",
			report: StatusReport{
				AgentID:      "agent-001",
				AgentName:    "Test Agent",
				AgentSource:  "test-software",
				SessionTopic: "task-001",
				Status:       "running",
				Timestamp:    now,
				Message:      "Task started",
				TTLMinutes:   30,
			},
			wantErr: false,
		},
		{
			name: "missing agent_id",
			report: StatusReport{
				SessionTopic: "task-001",
				Status:       "running",
				Timestamp:    now,
			},
			wantErr: true,
		},
		{
			name: "missing session_topic",
			report: StatusReport{
				AgentID:   "agent-001",
				Status:    "running",
				Timestamp: now,
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			report: StatusReport{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "invalid",
				Timestamp:    now,
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			report: StatusReport{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "running",
			},
			wantErr: true,
		},
		{
			name: "ttl_minutes out of range",
			report: StatusReport{
				AgentID:      "agent-001",
				SessionTopic: "task-001",
				Status:       "running",
				Timestamp:    now,
				TTLMinutes:   2000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.report.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("StatusReport.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
