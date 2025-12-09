package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// URLRequest represents a single URL to fetch
type URLRequest struct {
	URL     string            `json:"url"`
	Timeout *int              `json:"timeout,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ExecuteRequest represents the request payload
type ExecuteRequest struct {
	ExecutionTimeout *int         `json:"execution_timeout,omitempty"`
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

// Test scenarios for demonstrating different behaviors
var scenarios = map[string]struct {
	Description string
	Request     ExecuteRequest
}{
	"assignment": {
		Description: "google.com (300ms) + amazon.com (200ms)",
		Request: ExecuteRequest{
			ExecutionTimeout: intPtr(800),
			URLs: []URLRequest{
				{URL: "https://google.com", Timeout: intPtr(300), Headers: map[string]string{"X-Device-IP": "10.20.3.15"}},
				{URL: "https://amazon.com", Timeout: intPtr(200), Headers: map[string]string{"X-Device-IP": "10.10.30.150"}},
			},
		},
	},
	"success": {
		Description: "All requests succeed",
		Request: ExecuteRequest{
			URLs: []URLRequest{
				{URL: "https://httpbin.org/get"},
			},
		},
	},
	"individual-timeout": {
		Description: "Individual timeout triggers (500ms timeout, 2s delay)",
		Request: ExecuteRequest{
			ExecutionTimeout: intPtr(5000),
			URLs: []URLRequest{
				{URL: "https://httpbin.org/delay/2", Timeout: intPtr(500)},
			},
		},
	},
	"global-timeout": {
		Description: "Global timeout triggers (500ms global, 2s delay)",
		Request: ExecuteRequest{
			ExecutionTimeout: intPtr(500),
			URLs: []URLRequest{
				{URL: "https://httpbin.org/delay/2"},
			},
		},
	},
	"mixed": {
		Description: "Mixed results: success + timeout + 404",
		Request: ExecuteRequest{
			ExecutionTimeout: intPtr(3000),
			URLs: []URLRequest{
				{URL: "https://httpbin.org/get", Timeout: intPtr(1000)},
				{URL: "https://httpbin.org/delay/5", Timeout: intPtr(500)},
				{URL: "https://httpbin.org/status/404"},
			},
		},
	},
}

func intPtr(i int) *int {
	return &i
}

func listScenarios() {
	fmt.Println("Available test scenarios:")
	fmt.Println()
	for name, s := range scenarios {
		fmt.Printf("  %-20s %s\n", name, s.Description)
	}
	fmt.Println()
	fmt.Println("Usage: go run ./client -scenario <name>")
}

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "Server URL")
	timeout := flag.Int("timeout", 800, "Global execution timeout in milliseconds")
	urlsStr := flag.String("urls", "", "Comma-separated list of URLs to fetch")
	jsonFile := flag.String("json", "", "Path to JSON file with request payload")
	scenario := flag.String("scenario", "", "Run a predefined test scenario (use -list-scenarios to see options)")
	listScenariosFlag := flag.Bool("list-scenarios", false, "List available test scenarios")
	pretty := flag.Bool("pretty", true, "Pretty print JSON output")
	payloadOnly := flag.Bool("payload-only", false, "Only print response payloads (truncated)")

	flag.Parse()

	if *listScenariosFlag {
		listScenarios()
		return
	}

	var reqBody []byte
	var err error

	if *jsonFile != "" {
		reqBody, err = os.ReadFile(*jsonFile)
		if err != nil {
			log.Fatalf("Failed to read JSON file: %v", err)
		}
	} else if *scenario != "" {
		s, ok := scenarios[*scenario]
		if !ok {
			fmt.Printf("Unknown scenario: %s\n\n", *scenario)
			listScenarios()
			os.Exit(1)
		}
		fmt.Printf("Running scenario: %s\n", s.Description)
		reqBody, err = json.Marshal(s.Request)
		if err != nil {
			log.Fatalf("Failed to marshal request: %v", err)
		}
	} else if *urlsStr != "" {
		urls := strings.Split(*urlsStr, ",")
		urlRequests := make([]URLRequest, len(urls))
		for i, url := range urls {
			urlRequests[i] = URLRequest{URL: strings.TrimSpace(url)}
		}

		req := ExecuteRequest{
			ExecutionTimeout: timeout,
			URLs:             urlRequests,
		}

		reqBody, err = json.Marshal(req)
		if err != nil {
			log.Fatalf("Failed to marshal request: %v", err)
		}
	} else {
		// Default: use assignment example
		fmt.Println("No URLs specified, using assignment example...")
		fmt.Println("Tip: Use -list-scenarios to see all test scenarios")
		fmt.Println()
		s := scenarios["assignment"]
		reqBody, err = json.Marshal(s.Request)
		if err != nil {
			log.Fatalf("Failed to marshal request: %v", err)
		}
	}

	fmt.Println("=== Request ===")
	var prettyReq bytes.Buffer
	_ = json.Indent(&prettyReq, reqBody, "", "  ")
	fmt.Println(prettyReq.String())

	endpoint := *serverURL + "/execute"
	fmt.Printf("\n=== Sending to %s ===\n", endpoint)

	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	fmt.Printf("Status: %s\n\n", resp.Status)

	if *payloadOnly {
		var response ExecuteResponse
		if err := json.Unmarshal(body, &response); err != nil {
			log.Fatalf("Failed to parse response: %v", err)
		}

		fmt.Println("=== Results Summary ===")
		for i, result := range response.Results {
			fmt.Printf("\n[%d] Status: %d\n", i, result.Code)
			if result.Error != "" {
				fmt.Printf("    Error: %s\n", result.Error)
			}
			if result.Payload != "" {
				payload := result.Payload
				if len(payload) > 200 {
					payload = payload[:200] + "... (truncated)"
				}
				fmt.Printf("    Payload: %s\n", payload)
			}
		}
	} else {
		// Print full response
		fmt.Println("=== Response ===")
		if *pretty {
			var prettyResp bytes.Buffer
			if err := json.Indent(&prettyResp, body, "", "  "); err != nil {
				fmt.Println(string(body))
			} else {
				fmt.Println(prettyResp.String())
			}
		} else {
			fmt.Println(string(body))
		}
	}
}
