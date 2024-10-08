package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

const (
	DATABASE = "visits.db"
	apiPath  = "/api/count"
)

var dbConn *sql.DB

// init function to set up the database connection
func init() {
	var err error
	dbConn, err = sql.Open("sqlite3", DATABASE)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	// Configure connection pool
	dbConn.SetMaxOpenConns(10)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(time.Minute * 5)

	// Create the visits table if it doesn't exist
	if err := createTable(); err != nil {
		log.Fatalf("Error creating database table: %v", err)
	}
}

// createTable creates the visits table if it doesn't exist.
func createTable() error {
	log.Println("Attempting to create visits table if not exists")
	_, err := dbConn.Exec(`
                CREATE TABLE IF NOT EXISTS visits (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
                )
        `)
	if err != nil {
		log.Printf("Error creating table: %v", err)
		return fmt.Errorf("failed to create table: %w", err)
	}
	log.Println("Table creation or verification successful")
	return nil
}

// incrementVisitCount increments the visit count in the database.
func incrementVisitCount(w http.ResponseWriter, _ *http.Request) {
	_, err := dbConn.Exec("INSERT INTO visits (timestamp) VALUES (?)", time.Now())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to increment visit count: %v", err), http.StatusInternalServerError)
		log.Printf("Error incrementing visit count: %v", err)
		return
	}

	log.Println("Visit count incremented")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Ensure we send the correct status code
	response := map[string]string{"message": "Visit count incremented"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
}

// getVisitCount retrieves the visit count from the database.
func getVisitCount(w http.ResponseWriter, _ *http.Request) {
	var count int
	err := dbConn.QueryRow("SELECT COUNT(*) FROM visits").Scan(&count)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get visit count: %v", err), http.StatusInternalServerError)
		log.Printf("Error getting visit count: %v", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"visits": count})
}

// visitCountHandler handles POST and GET requests for the visit count.
func visitCountHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		incrementVisitCount(w, r)
	case http.MethodGet:
		getVisitCount(w, r)
	default:
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

// middleware for logging with request duration
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("Request: %s %s - Duration: %s", r.Method, r.URL, time.Since(start))
	})
}

// middleware for checking origin
func originCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
		if allowedOriginsStr == "" {
			http.Error(w, "Origin not allowed: ALLOWED_ORIGINS not set", http.StatusForbidden)
			return
		}
		allowedOrigins := strings.Split(allowedOriginsStr, ",")

		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = r.Host
		}

		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin { // Only allow exact matches
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "Origin not allowed", http.StatusForbidden)
	})
}

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

	// Create the handler
	var handler http.Handler

	// Apply middleware in the desired order
	handler = http.HandlerFunc(visitCountHandler)
	handler = loggingMiddleware(handler)
	handler = cors.Default().Handler(handler)
	if os.Getenv("APP_ENV") == "prod" {
		handler = originCheckMiddleware(handler)
	}

	// Use the handler for your API endpoint
	http.Handle(apiPath, handler)

	// Health check endpoint
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		if err := dbConn.Ping(); err != nil {
			log.Printf("Database health check failed: %v", err)
			http.Error(w, "Database health check failed", http.StatusInternalServerError)
			return
		}

		// If all checks pass, return 200 OK
		w.WriteHeader(http.StatusOK)
	})

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
