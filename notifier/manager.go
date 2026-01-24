package notifier

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// NotificationManager manages async notification delivery
type NotificationManager struct {
	client     *HTTPClient
	wg         sync.WaitGroup
	shutdownCh chan struct{}
	mu         sync.Mutex
	shutdown   bool
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(timeout time.Duration) *NotificationManager {
	return &NotificationManager{
		client:     NewHTTPClient(timeout),
		shutdownCh: make(chan struct{}),
	}
}

// Notify sends a notification asynchronously
func (nm *NotificationManager) Notify(ctx context.Context, data *NotificationData, webhookURL string) error {
	if webhookURL == "" {
		return nil
	}

	// Check if already shutdown
	nm.mu.Lock()
	if nm.shutdown {
		nm.mu.Unlock()
		return nil // Skip if shutdown
	}
	nm.mu.Unlock()

	// Build payload
	payload, err := BuildPayload(data)
	if err != nil {
		return fmt.Errorf("failed to build payload: %w", err)
	}

	// Launch async worker
	nm.wg.Add(1)
	go func() {
		defer nm.wg.Done()

		// Create context with timeout for this notification
		notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Send notification (no shutdown check - let queued notifications complete)
		if err := nm.client.Send(notifyCtx, webhookURL, payload); err != nil {
			log.Printf("Failed to send notification: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the notification manager
func (nm *NotificationManager) Shutdown(ctx context.Context) error {
	nm.mu.Lock()
	if nm.shutdown {
		nm.mu.Unlock()
		return nil
	}
	nm.shutdown = true
	nm.mu.Unlock()

	// Wait for pending notifications with timeout
	done := make(chan struct{})
	go func() {
		nm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All pending notifications completed")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: some notifications may not have completed")
	}
}
