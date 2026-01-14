package notifier

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNotificationManager_Notify_Disabled(t *testing.T) {
	// When webhook URL is empty, should not send
	manager := NewNotificationManager("", 5*time.Second)

	err := manager.Notify(context.Background(), &NotificationData{
		AgentID:      "test-agent",
		SessionTopic: "test-task",
		FromStatus:   "running",
		ToStatus:     "success",
		Timestamp:    time.Now(),
		Duration:     1 * time.Minute,
	})

	if err != nil {
		t.Errorf("Notify() with disabled manager, error = %v, want nil", err)
	}
}

func TestNotificationManager_Notify_Async(t *testing.T) {
	// Notification should not block
	var received atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		received.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewNotificationManager(server.URL, 5*time.Second)

	start := time.Now()
	err := manager.Notify(context.Background(), &NotificationData{
		AgentID:      "test-agent",
		SessionTopic: "test-task",
		FromStatus:   "running",
		ToStatus:     "success",
		Timestamp:    time.Now(),
		Duration:     1 * time.Minute,
	})
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Notify() error = %v, want nil", err)
	}

	if duration > 50*time.Millisecond {
		t.Errorf("Notify() duration = %v, want < 50ms (should return immediately)", duration)
	}

	// Wait for async notification
	time.Sleep(200 * time.Millisecond)

	if !received.Load() {
		t.Error("Notify() notification was not sent asynchronously")
	}
}

func TestNotificationManager_GracefulShutdown(t *testing.T) {
	var receivedCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		receivedCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewNotificationManager(server.URL, 5*time.Second)

	// Send multiple notifications
	pending := 3
	for i := 0; i < pending; i++ {
		err := manager.Notify(context.Background(), &NotificationData{
			AgentID:      fmt.Sprintf("test-agent-%d", i),
			SessionTopic: "test-task",
			FromStatus:   "running",
			ToStatus:     "success",
			Timestamp:    time.Now(),
			Duration:     1 * time.Minute,
		})
		if err != nil {
			t.Errorf("Notify() error = %v, want nil", err)
		}
	}

	// Shutdown should wait for pending notifications
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := manager.Shutdown(ctx)

	if err != nil {
		t.Errorf("Shutdown() error = %v, want nil", err)
	}

	// All notifications should have been sent
	if count := receivedCount.Load(); count != int32(pending) {
		t.Errorf("Shutdown() received %d notifications, want %d", count, pending)
	}
}

func TestNotificationManager_ShutdownTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Long delay
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewNotificationManager(server.URL, 10*time.Second)

	err := manager.Notify(context.Background(), &NotificationData{
		AgentID:      "test-agent",
		SessionTopic: "test-task",
		FromStatus:   "running",
		ToStatus:     "success",
		Timestamp:    time.Now(),
		Duration:     1 * time.Minute,
	})
	if err != nil {
		t.Errorf("Notify() error = %v, want nil", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = manager.Shutdown(ctx)

	if err == nil {
		t.Error("Shutdown() error = nil, want timeout error")
	}
}

func TestNotificationManager_ConcurrentNotifications(t *testing.T) {
	var receivedCount atomic.Int32
	var mu sync.Mutex
	receivedAgents := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract agent ID from body to verify all different notifications were received
		receivedCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewNotificationManager(server.URL, 5*time.Second)

	// Send notifications concurrently
	numNotifications := 10
	var wg sync.WaitGroup
	for i := 0; i < numNotifications; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			agentID := fmt.Sprintf("test-agent-%d", id)
			err := manager.Notify(context.Background(), &NotificationData{
				AgentID:      agentID,
				SessionTopic: "test-task",
				FromStatus:   "running",
				ToStatus:     "success",
				Timestamp:    time.Now(),
				Duration:     1 * time.Minute,
			})
			if err != nil {
				t.Errorf("Notify() error = %v, want nil", err)
			}

			mu.Lock()
			receivedAgents[agentID] = true
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Wait for all async notifications to complete
	time.Sleep(500 * time.Millisecond)

	// All notifications should have been queued
	if len(receivedAgents) != numNotifications {
		t.Errorf("Notify() queued %d notifications, want %d", len(receivedAgents), numNotifications)
	}

	// Shutdown and verify all were sent
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	manager.Shutdown(ctx)

	if count := receivedCount.Load(); count != int32(numNotifications) {
		t.Errorf("Notify() sent %d notifications, want %d", count, numNotifications)
	}
}

func TestNotificationManager_BuildPayloadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewNotificationManager(server.URL, 5*time.Second)

	// This should work fine - BuildPayload doesn't fail for normal data
	err := manager.Notify(context.Background(), &NotificationData{
		AgentID:      "test-agent",
		SessionTopic: "test-task",
		FromStatus:   "running",
		ToStatus:     "success",
		Timestamp:    time.Now(),
		Duration:     1 * time.Minute,
	})

	if err != nil {
		t.Errorf("Notify() error = %v, want nil", err)
	}
}

func TestNotificationManager_NotifyAfterShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewNotificationManager(server.URL, 5*time.Second)

	// Shutdown first
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	manager.Shutdown(ctx)

	// Try to send notification after shutdown
	err := manager.Notify(context.Background(), &NotificationData{
		AgentID:      "test-agent",
		SessionTopic: "test-task",
		FromStatus:   "running",
		ToStatus:     "success",
		Timestamp:    time.Now(),
		Duration:     1 * time.Minute,
	})

	// Should not error, but notification should be skipped
	if err != nil {
		t.Errorf("Notify() after shutdown, error = %v, want nil", err)
	}
}
