package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	port := flag.Int("port", 8080, "Server port")
	flag.Parse()

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/execute", ExecuteHandler)
	mux.HandleFunc("/health", HealthHandler)

	// Create server
	addr := fmt.Sprintf(":%d", *port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		_ = server.Close()
	}()

	// Start server
	log.Printf("URL Fetcher Service starting on http://localhost%s", addr)
	log.Printf("Endpoints:")
	log.Printf("  POST /execute - Execute URL fetch requests")
	log.Printf("  GET  /health  - Health check")

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}
