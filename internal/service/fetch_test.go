package service

import (
	"fetch/cmd/model"
	"fetch/internal/ratelimit"
	"testing"
	"time"
)

func createTestService() *FetchService {
	cfg := Config{
		FetchTimeout:       5 * time.Second,
		MaxRedirects:       10,
		MaxContentSize:     10 * 1024 * 1024,
		ResultTTL:          1 * time.Hour,
		CleanupInterval:    10 * time.Minute,
		MaxResultsInMemory: 10000,
	}
	rateLimiter := ratelimit.NewRateLimiter(100, 20, 1*time.Minute)
	return NewFetchService(cfg, rateLimiter)
}

func TestNewFetchService(t *testing.T) {
	service := createTestService()
	defer service.Stop()

	if service == nil {
		t.Fatal("NewFetchService returned nil")
	}

	if service.results == nil {
		t.Error("results slice not initialized")
	}

	if service.httpClient == nil {
		t.Error("httpClient not initialized")
	}

	if service.httpClient.Timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", service.httpClient.Timeout)
	}
}

func TestSubmitURLs(t *testing.T) {
	service := createTestService()
	defer service.Stop()

	urls := []string{
		"https://example.com",
		"https://google.com",
	}

	service.SubmitURLs(urls)

	// Give a moment for goroutines to start
	time.Sleep(100 * time.Millisecond)

	results := service.GetResults()

	if results.TotalURLs != len(urls) {
		t.Errorf("expected %d URLs, got %d", len(urls), results.TotalURLs)
	}

	if results.LastSubmission.IsZero() {
		t.Error("last submission timestamp not set")
	}
}

func TestGetResultsEmpty(t *testing.T) {
	service := createTestService()
	defer service.Stop()

	results := service.GetResults()

	if results.TotalURLs != 0 {
		t.Errorf("expected 0 URLs, got %d", results.TotalURLs)
	}

	if results.SuccessCount != 0 {
		t.Errorf("expected 0 successes, got %d", results.SuccessCount)
	}

	if results.FailedCount != 0 {
		t.Errorf("expected 0 failures, got %d", results.FailedCount)
	}

	if results.PendingCount != 0 {
		t.Errorf("expected 0 pending, got %d", results.PendingCount)
	}
}

func TestCleanupOldResults(t *testing.T) {
	service := createTestService()
	defer service.Stop()

	// Add some old results
	oldTime := time.Now().Add(-2 * time.Hour)       // Older than TTL
	recentTime := time.Now().Add(-30 * time.Minute) // Within TTL

	service.mu.Lock()
	service.results = []models.FetchResult{
		{URL: "https://old1.com", Status: "success", CreatedAt: oldTime},
		{URL: "https://old2.com", Status: "success", CreatedAt: oldTime},
		{URL: "https://recent1.com", Status: "success", CreatedAt: recentTime},
		{URL: "https://recent2.com", Status: "success", CreatedAt: recentTime},
	}
	service.mu.Unlock()

	// Run cleanup
	service.cleanupOldResults()

	// Check that only recent results remain
	results := service.GetResults()
	if results.TotalURLs != 2 {
		t.Errorf("expected 2 results after cleanup, got %d", results.TotalURLs)
	}

	// Verify cleanup stats
	stats := service.GetCleanupStats()
	if stats.TotalCleaned != 2 {
		t.Errorf("expected 2 cleaned results, got %d", stats.TotalCleaned)
	}

	if stats.CleanupCount != 1 {
		t.Errorf("expected 1 cleanup run, got %d", stats.CleanupCount)
	}
}

func TestMaxResultsInMemory(t *testing.T) {
	cfg := Config{
		FetchTimeout:       5 * time.Second,
		MaxRedirects:       10,
		MaxContentSize:     10 * 1024 * 1024,
		ResultTTL:          1 * time.Hour,
		CleanupInterval:    10 * time.Minute,
		MaxResultsInMemory: 100, // Use smaller value for test
	}
	rateLimiter := ratelimit.NewRateLimiter(100, 20, 1*time.Minute)
	service := NewFetchService(cfg, rateLimiter)
	defer service.Stop()

	// Add more than MaxResultsInMemory results (all recent)
	now := time.Now()
	service.mu.Lock()
	for i := 0; i < 150; i++ {
		service.results = append(service.results, models.FetchResult{
			URL:       "https://example.com",
			Status:    "success",
			CreatedAt: now,
		})
	}
	service.mu.Unlock()

	// Run cleanup
	service.cleanupOldResults()

	// Check that only MaxResultsInMemory remain
	results := service.GetResults()
	if results.TotalURLs != 100 {
		t.Errorf("expected 100 results after cleanup, got %d", results.TotalURLs)
	}

	// Verify cleanup stats
	stats := service.GetCleanupStats()
	if stats.TotalCleaned != 50 {
		t.Errorf("expected 50 cleaned results, got %d", stats.TotalCleaned)
	}
}

func TestClearAllResults(t *testing.T) {
	service := createTestService()
	defer service.Stop()

	// Add some results
	service.mu.Lock()
	for i := 0; i < 10; i++ {
		service.results = append(service.results, models.FetchResult{
			URL:       "https://example.com",
			Status:    "success",
			CreatedAt: time.Now(),
		})
	}
	service.mu.Unlock()

	// Clear all results
	count := service.ClearAllResults()

	if count != 10 {
		t.Errorf("expected to clear 10 results, got %d", count)
	}

	// Verify all results are gone
	results := service.GetResults()
	if results.TotalURLs != 0 {
		t.Errorf("expected 0 results after clear, got %d", results.TotalURLs)
	}

	// Verify cleanup stats
	stats := service.GetCleanupStats()
	if stats.TotalCleaned != 10 {
		t.Errorf("expected 10 total cleaned, got %d", stats.TotalCleaned)
	}
}

func TestCreatedAtPreserved(t *testing.T) {
	service := createTestService()
	defer service.Stop()

	// Submit a URL
	urls := []string{"https://example.com"}
	service.SubmitURLs(urls)

	// Get the created time
	service.mu.RLock()
	originalCreatedAt := service.results[0].CreatedAt
	service.mu.RUnlock()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Update the result (simulating a fetch completion)
	service.updateResult(0, models.FetchResult{
		URL:           "https://example.com",
		Status:        "success",
		Content:       "test",
		ContentLength: 4,
		StatusCode:    200,
		FetchedAt:     time.Now(),
	})

	// Verify CreatedAt is preserved
	service.mu.RLock()
	updatedCreatedAt := service.results[0].CreatedAt
	service.mu.RUnlock()

	if !updatedCreatedAt.Equal(originalCreatedAt) {
		t.Error("CreatedAt was not preserved after update")
	}
}

func TestCleanupDoesNotRemoveRecentResults(t *testing.T) {
	service := createTestService()
	defer service.Stop()

	// Add recent results (all within TTL)
	now := time.Now()
	service.mu.Lock()
	for i := 0; i < 5; i++ {
		service.results = append(service.results, models.FetchResult{
			URL:       "https://example.com",
			Status:    "success",
			CreatedAt: now.Add(-time.Duration(i) * time.Minute),
		})
	}
	service.mu.Unlock()

	// Run cleanup
	service.cleanupOldResults()

	// All results should still be there
	results := service.GetResults()
	if results.TotalURLs != 5 {
		t.Errorf("expected 5 results (none cleaned), got %d", results.TotalURLs)
	}

	// No results should have been cleaned
	stats := service.GetCleanupStats()
	if stats.TotalCleaned != 0 {
		t.Errorf("expected 0 cleaned results, got %d", stats.TotalCleaned)
	}
}

func TestGetCleanupStats(t *testing.T) {
	service := createTestService()
	defer service.Stop()

	// Add some results
	service.mu.Lock()
	for i := 0; i < 3; i++ {
		service.results = append(service.results, models.FetchResult{
			URL:       "https://example.com",
			Status:    "success",
			CreatedAt: time.Now(),
		})
	}
	service.mu.Unlock()

	// Get stats
	stats := service.GetCleanupStats()

	if stats.ResultsInMemory != 3 {
		t.Errorf("expected 3 results in memory, got %d", stats.ResultsInMemory)
	}

	if stats.CleanupCount != 0 {
		t.Error("expected 0 cleanup runs initially")
	}

	if stats.TotalCleaned != 0 {
		t.Error("expected 0 total cleaned initially")
	}
}

func TestServiceStop(t *testing.T) {
	service := createTestService()

	// Stop the service
	service.Stop()

	// Verify cleanup ticker is stopped (no panic should occur)
	// This is mainly to ensure Stop() can be called safely
}

func TestGetResultsStatistics(t *testing.T) {
	service := createTestService()
	defer service.Stop()

	service.mu.Lock()
	service.results = []models.FetchResult{
		{URL: "https://example.com", Status: "success", CreatedAt: time.Now()},
		{URL: "https://google.com", Status: "success", CreatedAt: time.Now()},
		{URL: "https://failed.com", Status: "failed", CreatedAt: time.Now()},
		{URL: "https://pending.com", Status: "pending", CreatedAt: time.Now()},
		{URL: "https://another-failed.com", Status: "failed", CreatedAt: time.Now()},
	}
	service.mu.Unlock()

	results := service.GetResults()

	if results.TotalURLs != 5 {
		t.Errorf("expected 5 total URLs, got %d", results.TotalURLs)
	}

	if results.SuccessCount != 2 {
		t.Errorf("expected 2 successes, got %d", results.SuccessCount)
	}

	if results.FailedCount != 2 {
		t.Errorf("expected 2 failures, got %d", results.FailedCount)
	}

	if results.PendingCount != 1 {
		t.Errorf("expected 1 pending, got %d", results.PendingCount)
	}
}
