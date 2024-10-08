package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3" // Import SQLite driver
	"github.com/rs/cors"
)

// createTestTable creates a new in-memory SQLite database for testing purposes.
func createTestTable(t *testing.T) (*sql.DB, func()) {
	tmpDB, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to create temporary database: %v", err)
	}

	// Create the visits table
	if _, err = tmpDB.Exec(`
		CREATE TABLE visits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	return tmpDB, func() {
		tmpDB.Close()
	}
}

// createTestableHandler initializes the HTTP handler for testing.
func createTestableHandler(t *testing.T) (http.Handler, func()) {
	db, cleanup := createTestTable(t) // Create a new test database
	dbConn = db                       // Assign the dbConn to the global variable for tests

	// Create the handler with middleware
	var handler http.Handler
	handler = loggingMiddleware(http.HandlerFunc(visitCountHandler))
	handler = cors.Default().Handler(handler)

	return handler, cleanup // Return both the handler and the cleanup function
}

// TestIncrementVisitCount checks the POST request to increment the visit count.
func TestIncrementVisitCount(t *testing.T) {
	handler, cleanup := createTestableHandler(t)
	defer cleanup() // Ensure cleanup is called after the test

	req, err := http.NewRequest(http.MethodPost, apiPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := map[string]string{"message": "Visit count incremented"}
	var actual map[string]string

	if err := json.Unmarshal(rr.Body.Bytes(), &actual); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}

	// Compare actual and expected responses
	if actual["message"] != expected["message"] {
		t.Errorf("handler returned unexpected body: got %v want %v", actual, expected)
	}
}

// TestGetVisitCount checks the GET request to retrieve visit count.
func TestGetVisitCount(t *testing.T) {
	handler, cleanup := createTestableHandler(t)
	defer cleanup() // Ensure cleanup is called after the test

	// Insert test data
	if _, err := dbConn.Exec("INSERT INTO visits (timestamp) VALUES (?), (?)", time.Now(), time.Now()); err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, apiPath, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]int
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}

	if response["visits"] != 2 {
		t.Errorf("handler returned unexpected visit count: got %v want %v", response["visits"], 2)
	}
}

// TestVisitCountHandler tests both POST and GET requests on visitCountHandler.
func TestVisitCountHandler(t *testing.T) {
	handler, cleanup := createTestableHandler(t)
	defer cleanup() // Ensure cleanup is called after the test

	t.Run("POST request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiPath, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("GET request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, apiPath, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("Invalid method", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, apiPath, nil)
		if err != nil {
			t.Fatal(err)
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
		}
	})
}

// TestLoggingMiddleware checks if logging middleware works correctly.
func TestLoggingMiddleware(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handler := loggingMiddleware(next)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

// TestOriginCheckMiddleware tests the origin check middleware.
func TestOriginCheckMiddleware(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://example.com,http://another-example.com")

	t.Run("Allowed origin", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", "http://example.com")
		rr := httptest.NewRecorder()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		handler := originCheckMiddleware(next)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("Disallowed origin", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Origin", "http://bad-origin.com")
		rr := httptest.NewRecorder()

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called for disallowed origin")
		})

		handler := originCheckMiddleware(next)
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
		}
	})
}
