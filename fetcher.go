package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// FetchURLs concurrently fetches all URLs respecting both global and individual timeouts.
// Results are returned in the same order as the input URLs.
func FetchURLs(ctx context.Context, urls []URLRequest) []URLResult {
	results := make([]URLResult, len(urls))
	var wg sync.WaitGroup

	for i, urlReq := range urls {
		wg.Add(1)
		go func(index int, req URLRequest) {
			defer wg.Done()
			results[index] = fetchSingleURL(ctx, req)
		}(i, urlReq)
	}

	wg.Wait()
	return results
}

// fetchSingleURL fetches a single URL with proper timeout handling
func fetchSingleURL(parentCtx context.Context, req URLRequest) URLResult {
	var ctx context.Context
	var cancel context.CancelFunc

	if req.Timeout != nil && *req.Timeout > 0 {
		individualTimeout := time.Duration(*req.Timeout) * time.Millisecond

		if deadline, ok := parentCtx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return URLResult{
					Code:  0,
					Error: "Request aborted by global timeout",
				}
			}
			if individualTimeout < remaining {
				ctx, cancel = context.WithTimeout(parentCtx, individualTimeout)
			} else {
				ctx, cancel = context.WithTimeout(parentCtx, remaining)
			}
		} else {
			ctx, cancel = context.WithTimeout(parentCtx, individualTimeout)
		}
	} else {
		ctx, cancel = context.WithCancel(parentCtx)
	}
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return URLResult{
			Code:  0,
			Error: fmt.Sprintf("Failed to create request: %s", err.Error()),
		}
	}

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	client := &http.Client{
		// Don't follow redirects automatically to capture redirect responses
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return handleFetchError(ctx, parentCtx, req, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if ctx.Err() != nil {
			return handleContextError(ctx, parentCtx, req)
		}
		return URLResult{
			Code:  0,
			Error: fmt.Sprintf("Failed to read response body: %s", err.Error()),
		}
	}

	return URLResult{
		Code:    resp.StatusCode,
		Payload: string(body),
	}
}

// handleFetchError processes errors from HTTP client.Do()
func handleFetchError(ctx, parentCtx context.Context, req URLRequest, err error) URLResult {
	if ctx.Err() != nil {
		return handleContextError(ctx, parentCtx, req)
	}

	return URLResult{
		Code:  0,
		Error: fmt.Sprintf("Request failed: %s", err.Error()),
	}
}

// handleContextError determines if timeout was global or individual
func handleContextError(ctx, parentCtx context.Context, req URLRequest) URLResult {
	if parentCtx.Err() != nil {
		return URLResult{
			Code:  0,
			Error: "Request aborted by global timeout",
		}
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		if req.Timeout != nil && *req.Timeout > 0 {
			return URLResult{
				Code:  0,
				Error: "Request aborted by individual timeout",
			}
		}
		return URLResult{
			Code:  0,
			Error: "Request aborted by timeout",
		}
	}

	return URLResult{
		Code:  0,
		Error: "Request cancelled",
	}
}
