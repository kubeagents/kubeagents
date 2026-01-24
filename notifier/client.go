package notifier

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"
)

const (
	maxRetries    = 3
	baseBackoff   = 100 * time.Millisecond
	backoffFactor = 2.0
)

// HTTPClient handles HTTP requests with retry logic
type HTTPClient struct {
	timeout    time.Duration
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP client
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Send sends payload to webhook URL with retry logic
func (c *HTTPClient) Send(ctx context.Context, url string, payload []byte) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(math.Pow(backoffFactor, float64(attempt-1))) * baseBackoff

			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		// Send request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed (attempt %d/%d): %w", attempt+1, maxRetries, err)
			log.Printf("Webhook notification failed: %v", lastErr)
			continue
		}

		// Read and close response body
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Check response status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("Webhook notification sent successfully (attempt %d/%d)", attempt+1, maxRetries)
			return nil
		}

		lastErr = fmt.Errorf("request failed with status %d (attempt %d/%d): %s",
			resp.StatusCode, attempt+1, maxRetries, string(body))
		log.Printf("Webhook notification failed: %v", lastErr)
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
