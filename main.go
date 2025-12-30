package main

import (
	"fetch/internal/config"
	"fetch/internal/handler"
	"fetch/internal/ratelimit"
	"fetch/internal/service"
	"log"
	"net/http"
)

func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	log.Println("Starting URL Fetch Service")
	cfg.LogConfig()

	// Create rate limiter
	rateLimiter := ratelimit.NewRateLimiter(
		cfg.RateLimitRequests,
		cfg.RateLimitBurst,
		cfg.RateLimitWindow,
	)

	// Create service config
	serviceConfig := service.Config{
		FetchTimeout:       cfg.FetchTimeout,
		MaxRedirects:       cfg.MaxRedirects,
		MaxContentSize:     cfg.MaxContentSize,
		ResultTTL:          cfg.ResultTTL,
		CleanupInterval:    cfg.CleanupInterval,
		MaxResultsInMemory: cfg.MaxResultsInMemory,
	}

	// Create fetch service
	fetchService := service.NewFetchService(serviceConfig, rateLimiter)

	// Create HTTP handlers
	handler := handlers.NewHandler(
		fetchService,
		cfg.RateLimitRequests,
		cfg.RateLimitWindow.String(),
	)

	// Register routes
	http.HandleFunc("/fetch", handler.HandleFetch)
	http.HandleFunc("/health", handler.HandleHealth)
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleStats(
			w, r,
			cfg.ResultTTL.String(),
			cfg.CleanupInterval.String(),
			cfg.MaxResultsInMemory,
		)
	})
	http.HandleFunc("/admin/clear", handler.HandleAdminClear)

	// Log endpoints
	log.Println("\nAvailable Endpoints:")
	log.Println("  POST /fetch        - Submit URLs for fetching")
	log.Println("  GET  /fetch        - Retrieve fetch results")
	log.Println("  GET  /health       - Health check")
	log.Println("  GET  /stats        - Service statistics")
	log.Println("  POST /admin/clear  - Clear all results (admin)")

	// Start server
	log.Printf("\n Server listening on %s\n", cfg.ServerAddress)
	if err := http.ListenAndServe(cfg.ServerAddress, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
