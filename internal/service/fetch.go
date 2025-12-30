package service

import (
	"context"
	"errors"
	"fetch/cmd/model"
	"fetch/internal/ratelimit"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// Config holds service configuration
type Config struct {
	FetchTimeout       time.Duration
	MaxRedirects       int
	MaxContentSize     int64
	ResultTTL          time.Duration
	CleanupInterval    time.Duration
	MaxResultsInMemory int
}

// FetchService manages URL fetching operations
type FetchService struct {
	mu              sync.RWMutex
	results         []models.FetchResult
	lastSubmission  time.Time
	httpClient      *http.Client
	rateLimiter     *ratelimit.RateLimiter
	cleanupTicker   *time.Ticker
	cleanupStopChan chan struct{}
	cleanupStats    models.CleanupStats
	config          Config
}

// NewFetchService creates a new fetch service instance
func NewFetchService(cfg Config, rateLimiter *ratelimit.RateLimiter) *FetchService {
	fs := &FetchService{
		results: make([]models.FetchResult, 0),
		httpClient: &http.Client{
			Timeout: cfg.FetchTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= cfg.MaxRedirects {
					return fmt.Errorf("stopped after %d redirects", cfg.MaxRedirects)
				}
				return nil
			},
		},
		rateLimiter:     rateLimiter,
		cleanupTicker:   time.NewTicker(cfg.CleanupInterval),
		cleanupStopChan: make(chan struct{}),
		config:          cfg,
	}

	// Start automatic cleanup goroutine
	go fs.runCleanup()

	return fs
}

// SubmitURLs receives URLs and starts fetching them concurrently
func (fs *FetchService) SubmitURLs(urls []string) {
	fs.mu.Lock()
	fs.lastSubmission = time.Now()

	// Add all URLs with pending status
	now := time.Now()
	for _, url := range urls {
		fs.results = append(fs.results, models.FetchResult{
			URL:       url,
			Status:    "pending",
			CreatedAt: now,
		})
	}
	fs.mu.Unlock()

	// Fetch URLs concurrently
	var wg sync.WaitGroup
	for i := range urls {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			fs.fetchURL(index)
		}(len(fs.results) - len(urls) + i)
	}

	// Wait for all fetches to complete in a separate goroutine
	go func() {
		wg.Wait()
		log.Println("All URLs fetched")
	}()
}

// fetchURL fetches content from a single URL and updates the result
func (fs *FetchService) fetchURL(index int) {
	fs.mu.RLock()
	url := fs.results[index].URL
	fs.mu.RUnlock()

	startTime := time.Now()

	// Validate URL format
	if url == "" {
		fs.updateResult(index, models.FetchResult{
			URL:      url,
			Status:   "failed",
			Error:    "URL is empty",
			Duration: time.Since(startTime).String(),
		})
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), fs.config.FetchTimeout)
	defer cancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		fs.updateResult(index, models.FetchResult{
			URL:      url,
			Status:   "failed",
			Error:    fmt.Sprintf("Failed to create request: %v", err),
			Duration: time.Since(startTime).String(),
		})
		log.Printf("Failed to create request for %s: %v", url, err)
		return
	}

	// Set user agent to identify our service
	req.Header.Set("User-Agent", "URL-Fetch-Service/1.0")

	// Track redirects
	redirectCount := 0
	clientWithRedirectTracking := &http.Client{
		Timeout: fs.httpClient.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectCount = len(via)
			if redirectCount >= fs.config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", fs.config.MaxRedirects)
			}
			// Copy user agent to redirect requests
			req.Header.Set("User-Agent", "URL-Fetch-Service/1.0")
			return nil
		},
	}

	// Perform the HTTP request
	resp, err := clientWithRedirectTracking.Do(req)
	if err != nil {
		// Check if error is due to redirect limit
		errMsg := fmt.Sprintf("Failed to fetch URL: %v", err)
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			errMsg = "Request timeout exceeded"
		}

		fs.updateResult(index, models.FetchResult{
			URL:           url,
			Status:        "failed",
			Error:         errMsg,
			Duration:      time.Since(startTime).String(),
			RedirectCount: redirectCount,
		})
		log.Printf("Failed to fetch %s: %v", url, err)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Warning: failed to close response body for %s: %v", url, closeErr)
		}
	}()

	// Limit response body size to prevent memory issues
	limitedReader := io.LimitReader(resp.Body, fs.config.MaxContentSize)

	// Read response body
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		fs.updateResult(index, models.FetchResult{
			URL:           url,
			Status:        "failed",
			StatusCode:    resp.StatusCode,
			Error:         fmt.Sprintf("Failed to read response body: %v", err),
			Duration:      time.Since(startTime).String(),
			FinalURL:      resp.Request.URL.String(),
			RedirectCount: redirectCount,
		})
		log.Printf("Failed to read body from %s: %v", url, err)
		return
	}

	// Check if we hit the size limit
	if int64(len(body)) >= fs.config.MaxContentSize {
		fs.updateResult(index, models.FetchResult{
			URL:           url,
			Status:        "failed",
			StatusCode:    resp.StatusCode,
			Error:         fmt.Sprintf("Response body too large (exceeds %d bytes)", fs.config.MaxContentSize),
			Duration:      time.Since(startTime).String(),
			FinalURL:      resp.Request.URL.String(),
			RedirectCount: redirectCount,
		})
		log.Printf("Response too large for %s", url)
		return
	}

	// Get final URL after redirects
	finalURL := resp.Request.URL.String()

	// Update result with success
	fs.updateResult(index, models.FetchResult{
		URL:           url,
		Status:        "success",
		Content:       string(body),
		ContentLength: len(body),
		StatusCode:    resp.StatusCode,
		FetchedAt:     time.Now(),
		Duration:      time.Since(startTime).String(),
		FinalURL:      finalURL,
		RedirectCount: redirectCount,
	})

	log.Printf("Successfully fetched %s (status: %d, size: %d bytes, redirects: %d, duration: %s)",
		url, resp.StatusCode, len(body), redirectCount, time.Since(startTime))
}

// updateResult updates a specific result at the given index
func (fs *FetchService) updateResult(index int, result models.FetchResult) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if index >= 0 && index < len(fs.results) {
		// Preserve CreatedAt from original result
		result.CreatedAt = fs.results[index].CreatedAt
		fs.results[index] = result
	}
}

// GetResults returns all fetch results with statistics
func (fs *FetchService) GetResults() models.FetchResponse {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	response := models.FetchResponse{
		TotalURLs:      len(fs.results),
		Results:        make([]models.FetchResult, len(fs.results)),
		LastSubmission: fs.lastSubmission,
	}

	// Copy results and calculate statistics
	for i, result := range fs.results {
		response.Results[i] = result
		switch result.Status {
		case "success":
			response.SuccessCount++
		case "failed":
			response.FailedCount++
		case "pending":
			response.PendingCount++
		}
	}

	return response
}

// runCleanup runs periodic cleanup of old results
func (fs *FetchService) runCleanup() {
	for {
		select {
		case <-fs.cleanupTicker.C:
			fs.cleanupOldResults()
		case <-fs.cleanupStopChan:
			fs.cleanupTicker.Stop()
			return
		}
	}
}

// cleanupOldResults removes results older than ResultTTL
func (fs *FetchService) cleanupOldResults() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	now := time.Now()
	cleaned := 0
	newResults := make([]models.FetchResult, 0, len(fs.results))

	// Remove results older than TTL
	for _, result := range fs.results {
		age := now.Sub(result.CreatedAt)
		if age < fs.config.ResultTTL {
			newResults = append(newResults, result)
		} else {
			cleaned++
		}
	}

	// If still too many results, keep only the most recent ones
	if len(newResults) > fs.config.MaxResultsInMemory {
		excess := len(newResults) - fs.config.MaxResultsInMemory
		newResults = newResults[excess:]
		cleaned += excess
	}

	if cleaned > 0 {
		fs.results = newResults
		fs.cleanupStats.LastCleanup = now
		fs.cleanupStats.TotalCleaned += cleaned
		fs.cleanupStats.CleanupCount++
		fs.cleanupStats.ResultsInMemory = len(fs.results)

		log.Printf("Cleanup: Removed %d old results, %d remaining in memory", cleaned, len(fs.results))
	}
}

// ClearAllResults manually clears all results (for testing or admin purposes)
func (fs *FetchService) ClearAllResults() int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	count := len(fs.results)
	fs.results = make([]models.FetchResult, 0)
	fs.cleanupStats.TotalCleaned += count
	fs.cleanupStats.ResultsInMemory = 0

	log.Printf("Manually cleared all %d results", count)
	return count
}

// GetCleanupStats returns cleanup statistics
func (fs *FetchService) GetCleanupStats() models.CleanupStats {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	stats := fs.cleanupStats
	stats.ResultsInMemory = len(fs.results)
	return stats
}

// GetRateLimiter returns the rate limiter
func (fs *FetchService) GetRateLimiter() *ratelimit.RateLimiter {
	return fs.rateLimiter
}

// Stop gracefully stops the fetch service
func (fs *FetchService) Stop() {
	close(fs.cleanupStopChan)
	log.Println("Fetch service stopped")
}
