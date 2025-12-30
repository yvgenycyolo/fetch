package ratelimit

import (
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting per IP
type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*Visitor
	rate     int
	burst    int
	window   time.Duration
}

// Visitor tracks rate limit info for a single IP
type Visitor struct {
	tokens       int
	lastSeen     time.Time
	windowStart  time.Time
	requestCount int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, burst int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		burst:    burst,
		window:   window,
	}

	// Cleanup old visitors every minute
	go rl.cleanupVisitors()

	return rl
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	visitor, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &Visitor{
			tokens:       rl.burst - 1,
			lastSeen:     now,
			windowStart:  now,
			requestCount: 1,
		}
		return true
	}

	visitor.lastSeen = now

	// Check if we're in a new window
	if now.Sub(visitor.windowStart) > rl.window {
		visitor.windowStart = now
		visitor.requestCount = 1
		visitor.tokens = rl.burst - 1
		return true
	}

	// Check request count limit
	if visitor.requestCount >= rl.rate {
		return false
	}

	// Refill tokens based on time passed
	elapsed := now.Sub(visitor.windowStart)
	windowSeconds := rl.window.Seconds()
	
	// Avoid divide by zero - if window is too small, just use the burst
	var tokensToAdd int
	if windowSeconds > 0 {
		tokensToAdd = int(elapsed.Seconds() * float64(rl.burst) / windowSeconds)
	} else {
		tokensToAdd = rl.burst
	}
	visitor.tokens = min(visitor.tokens+tokensToAdd, rl.burst)

	// Check if we have tokens available
	if visitor.tokens > 0 {
		visitor.tokens--
		visitor.requestCount++
		return true
	}

	return false
}

// cleanupVisitors removes old visitor entries
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, visitor := range rl.visitors {
			if now.Sub(visitor.lastSeen) > rl.window*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"active_ips":     len(rl.visitors),
		"rate_limit":     rl.rate,
		"burst_size":     rl.burst,
		"window_seconds": int(rl.window.Seconds()),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}


