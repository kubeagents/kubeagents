package notifier

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name         string
		data         *NotificationData
		wantContains []string
	}{
		{
			name: "running to success transition with all fields",
			data: &NotificationData{
				AgentID:      "agent-001",
				AgentName:    "Test Agent",
				SessionTopic: "task-001",
				FromStatus:   "running",
				ToStatus:     "success",
				Timestamp:    time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
				Message:      "Task completed",
				Content:      "Result: OK",
				Duration:     5 * time.Minute,
			},
			wantContains: []string{
				"Session Status Change",
				"Agent ID: agent-001",
				"Agent Name: Test Agent",
				"Session: task-001",
				"Status: running → success",
				"2024-01-15T10:30:45Z",
				"Duration: 5m0s",
				"Message: Task completed",
				"Content: Result: OK",
			},
		},
		{
			name: "running to failed transition",
			data: &NotificationData{
				AgentID:      "agent-002",
				AgentName:    "Failed Agent",
				SessionTopic: "task-002",
				FromStatus:   "running",
				ToStatus:     "failed",
				Timestamp:    time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
				Message:      "Task failed due to timeout",
				Content:      "Error: timeout exceeded",
				Duration:     10 * time.Minute,
			},
			wantContains: []string{
				"Agent ID: agent-002",
				"Status: running → failed",
				"Duration: 10m0s",
				"Message: Task failed due to timeout",
				"Content: Error: timeout exceeded",
			},
		},
		{
			name: "minimal fields - no message or content",
			data: &NotificationData{
				AgentID:      "agent-003",
				AgentName:    "",
				SessionTopic: "task-003",
				FromStatus:   "running",
				ToStatus:     "success",
				Timestamp:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				Message:      "",
				Content:      "",
				Duration:     1 * time.Minute,
			},
			wantContains: []string{
				"Agent ID: agent-003",
				"Session: task-003",
				"Status: running → success",
				"Duration: 1m0s",
			},
		},
		{
			name: "empty agent name",
			data: &NotificationData{
				AgentID:      "agent-004",
				AgentName:    "",
				SessionTopic: "task-004",
				FromStatus:   "running",
				ToStatus:     "success",
				Timestamp:    time.Now(),
				Duration:     2 * time.Minute,
			},
			wantContains: []string{
				"Agent ID: agent-004",
				"Agent Name: ",
				"Session: task-004",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMessage(tt.data)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatMessage() missing expected substring\nwant substring: %q\ngot: %s", want, got)
				}
			}

			// Verify no "Message:" or "Content:" lines when those fields are empty
			if tt.data.Message == "" {
				if strings.Contains(got, "Message:") {
					t.Errorf("FormatMessage() should not include 'Message:' when message is empty\ngot: %s", got)
				}
			}
			if tt.data.Content == "" {
				if strings.Contains(got, "Content:") {
					t.Errorf("FormatMessage() should not include 'Content:' when content is empty\ngot: %s", got)
				}
			}
		})
	}
}

func TestBuildPayload(t *testing.T) {
	tests := []struct {
		name    string
		data    *NotificationData
		wantErr bool
	}{
		{
			name: "valid notification data",
			data: &NotificationData{
				AgentID:      "agent-001",
				AgentName:    "Test Agent",
				SessionTopic: "task-001",
				FromStatus:   "running",
				ToStatus:     "success",
				Timestamp:    time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
				Message:      "Task completed",
				Content:      "Result: OK",
				Duration:     5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "minimal data",
			data: &NotificationData{
				AgentID:      "agent-002",
				SessionTopic: "task-002",
				FromStatus:   "running",
				ToStatus:     "failed",
				Timestamp:    time.Now(),
				Duration:     1 * time.Minute,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildPayload(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify JSON structure
			var payload WebhookPayload
			if err := json.Unmarshal(got, &payload); err != nil {
				t.Errorf("BuildPayload() returned invalid JSON: %v", err)
				return
			}

			// Verify required fields
			if payload.MsgType != "text" {
				t.Errorf("BuildPayload() msg_type = %v, want text", payload.MsgType)
			}

			if payload.Content.Text == "" {
				t.Errorf("BuildPayload() content.text is empty")
			}

			// Verify content contains agent ID
			if !strings.Contains(payload.Content.Text, tt.data.AgentID) {
				t.Errorf("BuildPayload() content.text missing agent_id %q\ngot: %s", tt.data.AgentID, payload.Content.Text)
			}

			// Verify content contains status transition
			statusTransition := tt.data.FromStatus + " → " + tt.data.ToStatus
			if !strings.Contains(payload.Content.Text, statusTransition) {
				t.Errorf("BuildPayload() content.text missing status transition %q\ngot: %s", statusTransition, payload.Content.Text)
			}
		})
	}
}

func TestWebhookPayload_JSONStructure(t *testing.T) {
	// Test that WebhookPayload marshals to correct JSON format
	payload := WebhookPayload{
		MsgType: "text",
		Content: WebhookContent{
			Text: "Hello from robot",
		},
	}

	got, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	want := `{"msg_type":"text","content":{"text":"Hello from robot"}}`
	if string(got) != want {
		t.Errorf("WebhookPayload JSON structure mismatch\ngot:  %s\nwant: %s", string(got), want)
	}
}
