package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExecuteHandler_Success(t *testing.T) {
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	}))
	defer targetServer.Close()

	reqBody := ExecuteRequest{
		URLs: []URLRequest{
			{URL: targetServer.URL},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ExecuteHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp ExecuteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("Results length: got %d, want 1", len(resp.Results))
	}
	if resp.Results[0].Code != 200 {
		t.Errorf("Results[0].Code: got %d, want 200", resp.Results[0].Code)
	}
	if resp.Results[0].Payload != "Hello" {
		t.Errorf("Results[0].Payload: got %q, want %q", resp.Results[0].Payload, "Hello")
	}
}

func TestExecuteHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/execute", nil)
	w := httptest.NewRecorder()

	ExecuteHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Status code: got %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestExecuteHandler_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/execute", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ExecuteHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestExecuteHandler_EmptyURLs(t *testing.T) {
	reqBody := ExecuteRequest{
		URLs: []URLRequest{},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ExecuteHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp ExecuteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Results) != 0 {
		t.Errorf("Results length: got %d, want 0", len(resp.Results))
	}
}

func TestExecuteHandler_CustomExecutionTimeout(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	timeout := 100
	reqBody := ExecuteRequest{
		ExecutionTimeout: &timeout,
		URLs: []URLRequest{
			{URL: slowServer.URL},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ExecuteHandler(w, req)

	var resp ExecuteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("Results length: got %d, want 1", len(resp.Results))
	}
	if resp.Results[0].Code != 0 {
		t.Errorf("Results[0].Code: got %d, want 0 (timeout)", resp.Results[0].Code)
	}
	if !strings.Contains(resp.Results[0].Error, "timeout") {
		t.Errorf("Results[0].Error should mention timeout, got %q", resp.Results[0].Error)
	}
}

func TestExecuteHandler_DefaultTimeout(t *testing.T) {
	// Test that default timeout (800ms) is used when not specified
	fastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer fastServer.Close()

	reqBody := ExecuteRequest{
		// No ExecutionTimeout specified - should use default 800ms
		URLs: []URLRequest{
			{URL: fastServer.URL},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ExecuteHandler(w, req)

	var resp ExecuteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Should succeed within default 800ms timeout
	if resp.Results[0].Code != 200 {
		t.Errorf("Results[0].Code: got %d, want 200", resp.Results[0].Code)
	}
}

func TestExecuteHandler_MultipleURLs(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("server1"))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("server2"))
	}))
	defer server2.Close()

	reqBody := ExecuteRequest{
		URLs: []URLRequest{
			{URL: server1.URL},
			{URL: server2.URL},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ExecuteHandler(w, req)

	var resp ExecuteResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(resp.Results) != 2 {
		t.Fatalf("Results length: got %d, want 2", len(resp.Results))
	}
	if resp.Results[0].Payload != "server1" {
		t.Errorf("Results[0].Payload: got %q, want %q", resp.Results[0].Payload, "server1")
	}
	if resp.Results[1].Code != 201 {
		t.Errorf("Results[1].Code: got %d, want 201", resp.Results[1].Code)
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code: got %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("Status: got %q, want %q", resp["status"], "ok")
	}
}

func TestExecuteHandler_ContentTypeJSON(t *testing.T) {
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	reqBody := ExecuteRequest{
		URLs: []URLRequest{{URL: targetServer.URL}},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	w := httptest.NewRecorder()

	ExecuteHandler(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", contentType, "application/json")
	}
}
