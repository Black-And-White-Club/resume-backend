package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockDataStore is a mock implementation of the DataStore interface for testing.
type MockDataStore struct {
	visitCount int
}

func (m *MockDataStore) IncrementVisitCount(timestamp time.Time) error {
	m.visitCount++
	return nil
}

func (m *MockDataStore) GetVisitCount() (int, error) {
	return m.visitCount, nil
}

func Test_incrementVisitCount(t *testing.T) {
	mockDataStore := &MockDataStore{}

	// Create a response recorder
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/increment", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	incrementVisitCount(w, req, mockDataStore)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK; got %v", res.Status)
	}

	var response map[string]string
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["message"] != "Visit count incremented" {
		t.Errorf("expected message 'Visit count incremented'; got %v", response["message"])
	}

	if mockDataStore.visitCount != 1 {
		t.Errorf("expected visit count to be 1; got %d", mockDataStore.visitCount)
	}
}

func Test_getVisitCount(t *testing.T) {
	mockDataStore := &MockDataStore{visitCount: 5} // Set a predefined visit count

	// Create a response recorder
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/count", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	getVisitCount(w, req, mockDataStore)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK; got %v", res.Status)
	}

	var response map[string]int
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	if response["visits"] != 5 {
		t.Errorf("expected visit count to be 5; got %d", response["visits"])
	}
}

func Test_visitCountHandler(t *testing.T) {
	mockDataStore := &MockDataStore{}

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{"POST Increment Visit Count", http.MethodPost, http.StatusOK},
		{"GET Retrieve Visit Count", http.MethodGet, http.StatusOK},
		{"Invalid Method", http.MethodPut, http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder
			w := httptest.NewRecorder()
			req, err := http.NewRequest(tt.method, "/visits", nil)
			if err != nil {
				t.Fatalf("could not create request: %v", err)
			}

			visitCountHandler(w, req, mockDataStore)

			res := w.Result()
			if res.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d; got %v", tt.expectedStatus, res.Status)
			}
		})
	}
}
