//go:build !test

package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"

	_ "github.com/mattn/go-sqlite3"
)

const apiPath = "/api/count"

func main() {
	// Initialize logger to write to stdout
	log.SetOutput(os.Stdout)

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, proceeding with default or environment variables")
	}

	// Validate required environment variables
	if os.Getenv("ALLOWED_ORIGINS") == "" {
		log.Fatal("ALLOWED_ORIGINS environment variable is not set")
	}

	// Initialize Prometheus metrics
	initPrometheusMetrics()

	// Database setup
	db, err := sql.Open("sqlite3", "visits.db")
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 5)

	// Create the DataStore
	dataStore := NewSQLiteDataStore(db)

	// Create the handler with dependency injection
	var handler http.Handler
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visitCountHandler(w, r, dataStore) // Inject dataStore
	})

	// Apply middleware in the desired order
	handler = prometheusMiddleware(handler)   // Wrap with Prometheus middleware
	handler = loggingMiddleware(handler)      // Logging middleware
	handler = cors.Default().Handler(handler) // CORS middleware

	// Apply origin check middleware for production
	if os.Getenv("APP_ENV") == "prod" {
		handler = originCheckMiddleware(handler)
	}

	// Use the handler for your API endpoint
	http.Handle(apiPath, handler)

	// Expose Prometheus metrics endpoint
	handlePrometheusMetrics()

	// Graceful shutdown
	server := &http.Server{Addr: ":8000", Handler: nil}
	go func() {
		log.Println("Server listening on :8000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Handle SIGINT and SIGTERM signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
