package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchSingleURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	ctx := context.Background()
	req := URLRequest{URL: server.URL}

	result := fetchSingleURL(ctx, req)

	if result.Code != 200 {
		t.Errorf("Code: got %d, want 200", result.Code)
	}
	if result.Payload != "Hello, World!" {
		t.Errorf("Payload: got %q, want %q", result.Payload, "Hello, World!")
	}
	if result.Error != "" {
		t.Errorf("Error should be empty, got %q", result.Error)
	}
}

func TestFetchSingleURL_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	req := URLRequest{
		URL: server.URL,
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
			"Authorization":   "Bearer token123",
		},
	}

	result := fetchSingleURL(ctx, req)

	if result.Code != 200 {
		t.Errorf("Code: got %d, want 200", result.Code)
	}
	if receivedHeaders.Get("X-Custom-Header") != "test-value" {
		t.Errorf("X-Custom-Header: got %q, want %q", receivedHeaders.Get("X-Custom-Header"), "test-value")
	}
	if receivedHeaders.Get("Authorization") != "Bearer token123" {
		t.Errorf("Authorization: got %q, want %q", receivedHeaders.Get("Authorization"), "Bearer token123")
	}
}

func TestFetchSingleURL_IndividualTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	timeout := 100
	req := URLRequest{
		URL:     server.URL,
		Timeout: &timeout,
	}

	result := fetchSingleURL(ctx, req)

	if result.Code != 0 {
		t.Errorf("Code: got %d, want 0 (timeout)", result.Code)
	}
	if !strings.Contains(result.Error, "individual timeout") {
		t.Errorf("Error should mention individual timeout, got %q", result.Error)
	}
}

func TestFetchSingleURL_GlobalTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := URLRequest{URL: server.URL}

	result := fetchSingleURL(ctx, req)

	if result.Code != 0 {
		t.Errorf("Code: got %d, want 0 (timeout)", result.Code)
	}
	if !strings.Contains(result.Error, "global timeout") {
		t.Errorf("Error should mention global timeout, got %q", result.Error)
	}
}

func TestFetchSingleURL_InvalidURL(t *testing.T) {
	ctx := context.Background()
	req := URLRequest{URL: "://invalid-url"}

	result := fetchSingleURL(ctx, req)

	if result.Code != 0 {
		t.Errorf("Code: got %d, want 0", result.Code)
	}
	if result.Error == "" {
		t.Error("Expected error for invalid URL")
	}
}

func TestFetchSingleURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	ctx := context.Background()
	req := URLRequest{URL: server.URL}

	result := fetchSingleURL(ctx, req)

	if result.Code != 500 {
		t.Errorf("Code: got %d, want 500", result.Code)
	}
	if result.Payload != "Internal Server Error" {
		t.Errorf("Payload: got %q, want %q", result.Payload, "Internal Server Error")
	}
}

func TestFetchURLs_Concurrent(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	ctx := context.Background()
	urls := []URLRequest{
		{URL: server.URL},
		{URL: server.URL},
		{URL: server.URL},
	}

	results := FetchURLs(ctx, urls)

	if len(results) != 3 {
		t.Fatalf("Results length: got %d, want 3", len(results))
	}
	for i, result := range results {
		if result.Code != 200 {
			t.Errorf("results[%d].Code: got %d, want 200", i, result.Code)
		}
	}
	if callCount != 3 {
		t.Errorf("Server call count: got %d, want 3", callCount)
	}
}

func TestFetchURLs_OrderPreserved(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return the path as payload
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(r.URL.Path))
	}))
	defer server.Close()

	ctx := context.Background()
	urls := []URLRequest{
		{URL: server.URL + "/first"},
		{URL: server.URL + "/second"},
		{URL: server.URL + "/third"},
	}

	results := FetchURLs(ctx, urls)

	if len(results) != 3 {
		t.Fatalf("Results length: got %d, want 3", len(results))
	}
	if results[0].Payload != "/first" {
		t.Errorf("results[0].Payload: got %q, want /first", results[0].Payload)
	}
	if results[1].Payload != "/second" {
		t.Errorf("results[1].Payload: got %q, want /second", results[1].Payload)
	}
	if results[2].Payload != "/third" {
		t.Errorf("results[2].Payload: got %q, want /third", results[2].Payload)
	}
}

func TestFetchURLs_MixedResults(t *testing.T) {
	fastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fast"))
	}))
	defer fastServer.Close()

	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("slow"))
	}))
	defer slowServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	timeout := 100
	urls := []URLRequest{
		{URL: fastServer.URL},
		{URL: slowServer.URL, Timeout: &timeout}, // Will timeout
		{URL: fastServer.URL},
	}

	results := FetchURLs(ctx, urls)

	if results[0].Code != 200 {
		t.Errorf("results[0].Code: got %d, want 200", results[0].Code)
	}
	if results[1].Code != 0 {
		t.Errorf("results[1].Code: got %d, want 0 (timeout)", results[1].Code)
	}
	if results[2].Code != 200 {
		t.Errorf("results[2].Code: got %d, want 200", results[2].Code)
	}
}

func TestFetchURLs_EmptyList(t *testing.T) {
	ctx := context.Background()
	results := FetchURLs(ctx, []URLRequest{})

	if len(results) != 0 {
		t.Errorf("Results length: got %d, want 0", len(results))
	}
}

func TestFetchURLs_GlobalTimeoutAffectsAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	urls := []URLRequest{
		{URL: server.URL},
		{URL: server.URL},
	}

	results := FetchURLs(ctx, urls)

	for i, result := range results {
		if result.Code != 0 {
			t.Errorf("results[%d].Code: got %d, want 0", i, result.Code)
		}
		if !strings.Contains(result.Error, "global timeout") {
			t.Errorf("results[%d].Error should mention global timeout, got %q", i, result.Error)
		}
	}
}

func TestFetchSingleURL_EffectiveTimeoutIsMinimum(t *testing.T) {
	// Test that when individual timeout > remaining global time, global wins
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Global timeout: 100ms, Individual timeout: 5000ms
	// Effective should be 100ms (global wins)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	timeout := 5000
	req := URLRequest{
		URL:     server.URL,
		Timeout: &timeout,
	}

	start := time.Now()
	result := fetchSingleURL(ctx, req)
	elapsed := time.Since(start)

	if result.Code != 0 {
		t.Errorf("Code: got %d, want 0", result.Code)
	}
	// Should timeout around 100ms, not 5000ms
	if elapsed > 500*time.Millisecond {
		t.Errorf("Took too long: %v (global timeout should have kicked in)", elapsed)
	}
}

