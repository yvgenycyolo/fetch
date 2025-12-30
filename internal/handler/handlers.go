package handlers

import (
	"encoding/json"
	"fetch/cmd/model"
	"fetch/internal/service"
	"fmt"
	"log"
	"net"
	"net/http"
)

// Handler holds the dependencies for HTTP handlers
type Handler struct {
	service         *service.FetchService
	rateLimitReqs   int
	rateLimitWindow string
}

// NewHandler creates a new HTTP handler
func NewHandler(svc *service.FetchService, rateLimitReqs int, rateLimitWindow string) *Handler {
	return &Handler{
		service:         svc,
		rateLimitReqs:   rateLimitReqs,
		rateLimitWindow: rateLimitWindow,
	}
}

// getIPFromRequest extracts the IP address from the request
func getIPFromRequest(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP if multiple are present
		return forwarded
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// HandlePostFetch handles POST /fetch - submit URLs for fetching
func (h *Handler) HandlePostFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check rate limit
	ip := getIPFromRequest(r)
	rateLimiter := h.service.GetRateLimiter()
	if !rateLimiter.Allow(ip) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", h.rateLimitReqs))
		w.Header().Set("X-RateLimit-Window", h.rateLimitWindow)
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Rate limit exceeded",
			"message": fmt.Sprintf("Maximum %d requests per %s allowed", h.rateLimitReqs, h.rateLimitWindow),
		})
		log.Printf("Rate limit exceeded for IP: %s", ip)
		return
	}

	var req models.FetchRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.URLs) == 0 {
		http.Error(w, "No URLs provided", http.StatusBadRequest)
		return
	}

	log.Printf("Received request to fetch %d URLs from IP: %s", len(req.URLs), ip)

	// Submit URLs for fetching
	h.service.SubmitURLs(req.URLs)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "URLs submitted for fetching",
		"total_urls": len(req.URLs),
		"status":     "processing",
	})
}

// HandleGetFetch handles GET /fetch - retrieve fetch results
func (h *Handler) HandleGetFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	results := h.service.GetResults()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.Encode(results)
}

// HandleFetch routes based on HTTP method
func (h *Handler) HandleFetch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.HandlePostFetch(w, r)
	case http.MethodGet:
		h.HandleGetFetch(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleHealth handles GET /health - health check
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// HandleStats handles GET /stats - service statistics
func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request, resultTTL, cleanupInterval string, maxResults int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rateLimiter := h.service.GetRateLimiter()
	stats := rateLimiter.GetStats()
	results := h.service.GetResults()
	cleanupStats := h.service.GetCleanupStats()

	response := map[string]interface{}{
		"rate_limiter": stats,
		"fetch_stats": map[string]interface{}{
			"total_urls":    results.TotalURLs,
			"success_count": results.SuccessCount,
			"failed_count":  results.FailedCount,
			"pending_count": results.PendingCount,
		},
		"cleanup": map[string]interface{}{
			"last_cleanup":      cleanupStats.LastCleanup,
			"total_cleaned":     cleanupStats.TotalCleaned,
			"cleanup_count":     cleanupStats.CleanupCount,
			"results_in_memory": cleanupStats.ResultsInMemory,
			"ttl":               resultTTL,
			"max_results":       maxResults,
			"cleanup_interval":  cleanupInterval,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAdminClear handles POST /admin/clear - clear all results
func (h *Handler) HandleAdminClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count := h.service.ClearAllResults()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "All results cleared",
		"results_cleared": count,
	})
}
