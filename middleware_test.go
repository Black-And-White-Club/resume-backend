package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func Test_loggingMiddleware(t *testing.T) {
	// Define a dummy handler to pass into the middleware
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create a mock request
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Create a response recorder to record the response
	rr := httptest.NewRecorder()

	// Wrap the dummy handler with the logging middleware
	handler := loggingMiddleware(dummyHandler)

	// Capture the log output
	start := time.Now()
	handler.ServeHTTP(rr, req)
	duration := time.Since(start)

	// Validate the response status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, status)
	}

	// The test here is just a basic validation. In a real scenario, you'd want
	// to capture the actual logs and validate the output, but that's a bit more involved.
	// This test ensures the middleware function executes without failure.
	t.Logf("Request completed with duration: %v", duration)
}

func Test_originCheckMiddleware(t *testing.T) {
	// Define allowed origins in environment variables
	os.Setenv("ALLOWED_ORIGINS", "http://allowed.com,http://anotherallowed.com")

	tests := []struct {
		name           string
		origin         string
		expectedStatus int
	}{
		{"Allowed origin", "http://allowed.com", http.StatusOK},
		{"Another allowed origin", "http://anotherallowed.com", http.StatusOK},
		{"Disallowed origin", "http://disallowed.com", http.StatusForbidden},
		{"No origin header", "", http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a dummy handler that will be wrapped by the middleware
			dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create a mock request with the Origin header
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Wrap the dummy handler with the originCheckMiddleware
			handler := originCheckMiddleware(dummyHandler)

			// Serve the request
			handler.ServeHTTP(rr, req)

			// Check the status code
			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, status)
			}

			// If the status is OK, check that the Access-Control-Allow-Origin header is set
			if tt.expectedStatus == http.StatusOK && rr.Header().Get("Access-Control-Allow-Origin") != tt.origin {
				t.Errorf("expected Access-Control-Allow-Origin header to be %s, got %s", tt.origin, rr.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}

	// Clean up environment variable
	os.Unsetenv("ALLOWED_ORIGINS")
}
