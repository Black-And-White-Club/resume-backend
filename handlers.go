package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// incrementVisitCount increments the visit count in the database.
func incrementVisitCount(w http.ResponseWriter, r *http.Request, dataStore DataStore) {
	err := dataStore.IncrementVisitCount(r.Context(), time.Now()) // Pass the request context
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to increment visit count: %v", err), http.StatusInternalServerError)
		return
	}

	log.Println("Visit count incremented")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{"message": "Visit count incremented"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
		return
	}
}

// getVisitCount retrieves the visit count from the database.
func getVisitCount(w http.ResponseWriter, r *http.Request, dataStore DataStore) {
	count, err := dataStore.GetVisitCount(r.Context()) // Pass the request context
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get visit count: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"visits": count})
}

// visitCountHandler handles POST and GET requests for the visit count.
func visitCountHandler(w http.ResponseWriter, r *http.Request, dataStore DataStore) {
	switch r.Method {
	case http.MethodPost:
		incrementVisitCount(w, r, dataStore)
	case http.MethodGet:
		getVisitCount(w, r, dataStore)
	default:
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}
