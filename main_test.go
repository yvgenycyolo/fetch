package main

import (
	"encoding/json"
	"fetch/cmd/model"
	"fetch/internal/handler"
	"fetch/internal/ratelimit"
	"fetch/internal/service"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Helper function to create a test service
func createTestService() *service.FetchService {
	cfg := service.Config{
		FetchTimeout:       5 * time.Second,
		MaxRedirects:       10,
		MaxContentSize:     10 * 1024 * 1024,
		ResultTTL:          1 * time.Hour,
		CleanupInterval:    10 * time.Minute,
		MaxResultsInMemory: 10000,
	}
	rateLimiter := ratelimit.NewRateLimiter(100, 20, 1*time.Minute)
	return service.NewFetchService(cfg, rateLimiter)
}

// Helper function to create test handler
func createTestHandler() *handlers.Handler {
	svc := createTestService()
	return handlers.NewHandler(svc, 100, "1m")
}

func TestHandlePostFetchSuccess(t *testing.T) {
	handler := createTestHandler()

	reqBody := `{"urls": ["https://example.com", "https://google.com"]}`
	req := httptest.NewRequest("POST", "/fetch", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandlePostFetch(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["total_urls"].(float64) != 2 {
		t.Errorf("expected 2 URLs, got %v", response["total_urls"])
	}

	if response["status"] != "processing" {
		t.Errorf("expected status 'processing', got '%v'", response["status"])
	}
}

func TestHandlePostFetchInvalidJSON(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("POST", "/fetch", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandlePostFetch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandlePostFetchEmptyURLs(t *testing.T) {
	handler := createTestHandler()

	reqBody := `{"urls": []}`
	req := httptest.NewRequest("POST", "/fetch", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandlePostFetch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleGetFetchSuccess(t *testing.T) {
	svc := createTestService()
	defer svc.Stop()

	handler := handlers.NewHandler(svc, 100, "1m")

	// Add some mock results
	svc.SubmitURLs([]string{"https://example.com"})
	time.Sleep(100 * time.Millisecond)

	req := httptest.NewRequest("GET", "/fetch", nil)
	w := httptest.NewRecorder()

	handler.HandleGetFetch(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.FetchResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TotalURLs != 1 {
		t.Errorf("expected 1 total URL, got %d", response.TotalURLs)
	}
}

func TestHandleFetchRouting(t *testing.T) {
	handler := createTestHandler()

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "POST with valid body",
			method:         "POST",
			body:           `{"urls": ["https://example.com"]}`,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "GET request",
			method:         "GET",
			body:           "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT request (not allowed)",
			method:         "PUT",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "DELETE request (not allowed)",
			method:         "DELETE",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/fetch", strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			handler.HandleFetch(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandleHealth(t *testing.T) {
	handler := createTestHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got '%s'", w.Body.String())
	}
}

func TestIntegrationFullWorkflow(t *testing.T) {
	// Create mock server for successful requests
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success response"))
	}))
	defer successServer.Close()

	svc := createTestService()
	defer svc.Stop()

	handler := handlers.NewHandler(svc, 100, "1m")

	// Step 1: Submit URLs via POST
	reqBody := `{"urls": ["` + successServer.URL + `"]}`
	postReq := httptest.NewRequest("POST", "/fetch", strings.NewReader(reqBody))
	postReq.Header.Set("Content-Type", "application/json")
	postWriter := httptest.NewRecorder()

	handler.HandleFetch(postWriter, postReq)

	if postWriter.Code != http.StatusAccepted {
		t.Fatalf("POST request failed with status %d", postWriter.Code)
	}

	// Wait for fetches to complete
	time.Sleep(500 * time.Millisecond)

	// Step 2: Retrieve results via GET
	getReq := httptest.NewRequest("GET", "/fetch", nil)
	getWriter := httptest.NewRecorder()

	handler.HandleFetch(getWriter, getReq)

	if getWriter.Code != http.StatusOK {
		t.Fatalf("GET request failed with status %d", getWriter.Code)
	}

	// Step 3: Validate results
	var response models.FetchResponse
	if err := json.NewDecoder(getWriter.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TotalURLs != 1 {
		t.Errorf("expected 1 total URL, got %d", response.TotalURLs)
	}

	if response.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", response.SuccessCount)
	}

	// Verify individual results
	for _, result := range response.Results {
		if result.Status != models.StatusSuccess {
			t.Errorf("expected success for %s, got %s", result.URL, result.Status)
		}

		if result.StatusCode == 0 {
			t.Errorf("status code not set for %s", result.URL)
		}

		if result.ContentLength == 0 {
			t.Errorf("content length not set for %s", result.URL)
		}
	}
}

func TestHandleAdminClear(t *testing.T) {
	svc := createTestService()
	defer svc.Stop()

	handler := handlers.NewHandler(svc, 100, "1m")

	// Add some results
	svc.SubmitURLs([]string{"https://example.com", "https://google.com"})
	time.Sleep(100 * time.Millisecond)

	// Clear results
	req := httptest.NewRequest("POST", "/admin/clear", nil)
	w := httptest.NewRecorder()

	handler.HandleAdminClear(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["results_cleared"].(float64) != 2 {
		t.Errorf("expected 2 results cleared, got %v", response["results_cleared"])
	}

	// Verify results are cleared
	results := svc.GetResults()
	if results.TotalURLs != 0 {
		t.Errorf("expected 0 results after clear, got %d", results.TotalURLs)
	}
}
