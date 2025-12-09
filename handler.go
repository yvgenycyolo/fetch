package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// ExecuteHandler handles POST /execute requests
func ExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	executionTimeout := DefaultExecutionTimeout
	if req.ExecutionTimeout != nil && *req.ExecutionTimeout > 0 {
		executionTimeout = *req.ExecutionTimeout
	}

	log.Printf("Received request: %d URLs, execution_timeout=%dms", len(req.URLs), executionTimeout)

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(executionTimeout)*time.Millisecond)
	defer cancel()

	// Fetch all URLs concurrently
	results := FetchURLs(ctx, req.URLs)

	response := ExecuteResponse{
		Results: results,
	}

	successCount := 0
	for _, result := range results {
		if result.Code >= 200 && result.Code < 300 {
			successCount++
		}
	}
	log.Printf("Completed: %d/%d successful", successCount, len(results))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// HealthHandler handles GET /health for health checks
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
