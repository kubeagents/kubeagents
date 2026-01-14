package notifier

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPClient_Send_Success(t *testing.T) {
	// Mock HTTP server that responds with 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("Send() method = %v, want POST", r.Method)
		}

		// Verify Content-Type header
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Send() Content-Type = %v, want application/json", ct)
		}

		// Verify payload
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "msg_type") {
			t.Errorf("Send() body missing msg_type field: %s", string(body))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second)
	payload := []byte(`{"msg_type":"text","content":{"text":"test"}}`)

	err := client.Send(context.Background(), payload)

	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}
}

func TestHTTPClient_Send_RetryOnFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second)
	payload := []byte(`{"msg_type":"text"}`)

	err := client.Send(context.Background(), payload)

	if err != nil {
		t.Errorf("Send() error = %v, want nil (should succeed on 3rd attempt)", err)
	}

	if attempts != 3 {
		t.Errorf("Send() attempts = %d, want 3", attempts)
	}
}

func TestHTTPClient_Send_ExponentialBackoff(t *testing.T) {
	timestamps := []time.Time{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamps = append(timestamps, time.Now())
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second)
	payload := []byte(`{"msg_type":"text"}`)

	client.Send(context.Background(), payload)

	if len(timestamps) != 3 {
		t.Fatalf("Send() attempts = %d, want 3", len(timestamps))
	}

	// Verify exponential backoff: ~100ms, ~200ms
	delay1 := timestamps[1].Sub(timestamps[0])
	delay2 := timestamps[2].Sub(timestamps[1])

	// Allow some tolerance for timing (50ms tolerance)
	if delay1 < 80*time.Millisecond || delay1 > 150*time.Millisecond {
		t.Errorf("Send() first backoff = %v, want ~100ms", delay1)
	}

	if delay2 < 180*time.Millisecond || delay2 > 250*time.Millisecond {
		t.Errorf("Send() second backoff = %v, want ~200ms", delay2)
	}
}

func TestHTTPClient_Send_MaxRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 5*time.Second)
	payload := []byte(`{"msg_type":"text"}`)

	err := client.Send(context.Background(), payload)

	if err == nil {
		t.Error("Send() error = nil, want error after max retries")
	}

	if attempts != 3 {
		t.Errorf("Send() attempts = %d, want 3 (max retries)", attempts)
	}

	if !strings.Contains(err.Error(), "max retries") {
		t.Errorf("Send() error message should contain 'max retries', got: %v", err)
	}
}

func TestHTTPClient_Send_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client := NewHTTPClient(server.URL, 5*time.Second)
	payload := []byte(`{"msg_type":"text"}`)

	err := client.Send(ctx, payload)

	if err == nil {
		t.Error("Send() error = nil, want context error")
	}

	// Should be context deadline exceeded or context canceled
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Send() error should contain 'context', got: %v", err)
	}
}

func TestHTTPClient_Send_SuccessOn2xxStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"200 OK", http.StatusOK, false},
		{"201 Created", http.StatusCreated, false},
		{"204 No Content", http.StatusNoContent, false},
		{"400 Bad Request", http.StatusBadRequest, true},
		{"404 Not Found", http.StatusNotFound, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewHTTPClient(server.URL, 5*time.Second)
			payload := []byte(`{"msg_type":"text"}`)

			err := client.Send(context.Background(), payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("Send() with status %d, error = %v, wantErr %v", tt.statusCode, err, tt.wantErr)
			}
		})
	}
}

func TestHTTPClient_Send_InvalidURL(t *testing.T) {
	client := NewHTTPClient("http://invalid-url-that-does-not-exist-12345.com", 5*time.Second)
	payload := []byte(`{"msg_type":"text"}`)

	err := client.Send(context.Background(), payload)

	if err == nil {
		t.Error("Send() with invalid URL, error = nil, want error")
	}
}

func TestHTTPClient_Send_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, 100*time.Millisecond)
	payload := []byte(`{"msg_type":"text"}`)

	err := client.Send(context.Background(), payload)

	if err == nil {
		t.Error("Send() with timeout, error = nil, want timeout error")
	}
}
