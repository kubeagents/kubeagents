package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheck)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("HealthCheck() status = %v, want %v", status, http.StatusOK)
	}

	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("HealthCheck() invalid JSON: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("HealthCheck() status = %v, want ok", response.Status)
	}

	if response.Timestamp.IsZero() {
		t.Error("HealthCheck() timestamp is zero")
	}

	// Test method not allowed
	req, _ = http.NewRequest("POST", "/health", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("HealthCheck() POST status = %v, want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHealthCheck_ContentType(t *testing.T) {
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheck)

	handler.ServeHTTP(rr, req)

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("HealthCheck() Content-Type = %v, want application/json", contentType)
	}
}
