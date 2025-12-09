package main

// URLRequest represents a single URL to fetch with optional timeout and headers
type URLRequest struct {
	URL     string            `json:"url"`
	Timeout *int              `json:"timeout,omitempty"` // milliseconds, optional
	Headers map[string]string `json:"headers,omitempty"`
}

// ExecuteRequest represents the incoming request payload
type ExecuteRequest struct {
	ExecutionTimeout *int         `json:"execution_timeout,omitempty"` // milliseconds, optional (default: 800ms)
	URLs             []URLRequest `json:"urls"`
}

// URLResult represents the result of fetching a single URL
type URLResult struct {
	Code    int    `json:"code"`
	Error   string `json:"error,omitempty"`
	Payload string `json:"payload,omitempty"`
}

// ExecuteResponse represents the response payload
type ExecuteResponse struct {
	Results []URLResult `json:"results"`
}

// DefaultExecutionTimeout is the default global timeout in milliseconds
const DefaultExecutionTimeout = 800


