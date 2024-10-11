package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// middleware for logging with request duration
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("Request: %s %s - Duration: %s", r.Method, r.URL, time.Since(start))
	})
}

func originCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			http.Error(w, "Allowed origins not set", http.StatusInternalServerError)
			return
		}

		allowed := false
		for _, origin := range strings.Split(allowedOrigins, ",") {
			if r.Header.Get("Origin") == origin {
				allowed = true
				break
			}
		}

		if !allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Allow CORS
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		next.ServeHTTP(w, r)
	})
}
