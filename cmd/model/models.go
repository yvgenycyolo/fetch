package models

import "time"

// Status constants for FetchResult
const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusPending = "pending"
)

// FetchRequest represents the incoming POST request payload
type FetchRequest struct {
	URLs []string `json:"urls"`
}

// FetchResult represents the result of fetching a single URL
type FetchResult struct {
	URL           string    `json:"url"`
	Status        string    `json:"status"` // "success", "failed", "pending"
	Content       string    `json:"content,omitempty"`
	ContentLength int       `json:"content_length"`
	StatusCode    int       `json:"status_code,omitempty"`
	Error         string    `json:"error,omitempty"`
	FetchedAt     time.Time `json:"fetched_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"` // When the result was created
	Duration      string    `json:"duration,omitempty"`
	RedirectCount int       `json:"redirect_count,omitempty"`
	FinalURL      string    `json:"final_url,omitempty"` // Final URL after redirects
}

// FetchResponse represents the GET response containing all fetch results
type FetchResponse struct {
	TotalURLs      int           `json:"total_urls"`
	SuccessCount   int           `json:"success_count"`
	FailedCount    int           `json:"failed_count"`
	PendingCount   int           `json:"pending_count"`
	Results        []FetchResult `json:"results"`
	LastSubmission time.Time     `json:"last_submission,omitempty"`
}

// CleanupStats tracks cleanup statistics
type CleanupStats struct {
	LastCleanup     time.Time `json:"last_cleanup"`
	TotalCleaned    int       `json:"total_cleaned"`
	CleanupCount    int       `json:"cleanup_count"`
	ResultsInMemory int       `json:"results_in_memory"`
}

