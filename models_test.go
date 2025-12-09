package main

import (
	"encoding/json"
	"testing"
)

func TestURLRequestUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected URLRequest
	}{
		{
			name:  "minimal request",
			input: `{"url": "https://example.com"}`,
			expected: URLRequest{
				URL: "https://example.com",
			},
		},
		{
			name:  "with timeout",
			input: `{"url": "https://example.com", "timeout": 500}`,
			expected: URLRequest{
				URL:     "https://example.com",
				Timeout: intPtr(500),
			},
		},
		{
			name:  "with headers",
			input: `{"url": "https://example.com", "headers": {"X-Custom": "value"}}`,
			expected: URLRequest{
				URL:     "https://example.com",
				Headers: map[string]string{"X-Custom": "value"},
			},
		},
		{
			name:  "full request",
			input: `{"url": "https://example.com", "timeout": 1000, "headers": {"Content-Type": "text/plain"}}`,
			expected: URLRequest{
				URL:     "https://example.com",
				Timeout: intPtr(1000),
				Headers: map[string]string{"Content-Type": "text/plain"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req URLRequest
			err := json.Unmarshal([]byte(tt.input), &req)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if req.URL != tt.expected.URL {
				t.Errorf("URL: got %q, want %q", req.URL, tt.expected.URL)
			}
			if (req.Timeout == nil) != (tt.expected.Timeout == nil) {
				t.Errorf("Timeout nil mismatch: got %v, want %v", req.Timeout, tt.expected.Timeout)
			}
			if req.Timeout != nil && tt.expected.Timeout != nil && *req.Timeout != *tt.expected.Timeout {
				t.Errorf("Timeout: got %d, want %d", *req.Timeout, *tt.expected.Timeout)
			}
		})
	}
}

func TestExecuteRequestUnmarshal(t *testing.T) {
	input := `{
		"execution_timeout": 2000,
		"urls": [
			{"url": "https://google.com", "timeout": 500},
			{"url": "https://amazon.com", "timeout": 300}
		]
	}`

	var req ExecuteRequest
	err := json.Unmarshal([]byte(input), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if req.ExecutionTimeout == nil || *req.ExecutionTimeout != 2000 {
		t.Errorf("ExecutionTimeout: got %v, want 2000", req.ExecutionTimeout)
	}
	if len(req.URLs) != 2 {
		t.Fatalf("URLs length: got %d, want 2", len(req.URLs))
	}
	if req.URLs[0].URL != "https://google.com" {
		t.Errorf("URLs[0].URL: got %q, want %q", req.URLs[0].URL, "https://google.com")
	}
	if req.URLs[1].URL != "https://amazon.com" {
		t.Errorf("URLs[1].URL: got %q, want %q", req.URLs[1].URL, "https://amazon.com")
	}
}

func TestURLResultMarshal(t *testing.T) {
	tests := []struct {
		name     string
		result   URLResult
		expected string
	}{
		{
			name:     "success response",
			result:   URLResult{Code: 200, Payload: "Hello"},
			expected: `{"code":200,"payload":"Hello"}`,
		},
		{
			name:     "error response",
			result:   URLResult{Code: 0, Error: "timeout"},
			expected: `{"code":0,"error":"timeout"}`,
		},
		{
			name:     "404 with body",
			result:   URLResult{Code: 404, Payload: "Not Found"},
			expected: `{"code":404,"payload":"Not Found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal both for comparison (avoids whitespace issues)
			var got, want map[string]interface{}
			err = json.Unmarshal(data, &got)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			err = json.Unmarshal([]byte(tt.expected), &want)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if got["code"] != want["code"] {
				t.Errorf("code mismatch: got %v, want %v", got["code"], want["code"])
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
