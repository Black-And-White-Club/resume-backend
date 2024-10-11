package main

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB() (*SQLiteDataStore, error) {
	db, err := sql.Open("sqlite3", ":memory:") // Use an in-memory database for testing
	if err != nil {
		return nil, err
	}

	// Create the DataStore
	dataStore := NewSQLiteDataStore(db)

	// Create the visits table
	if err := createTable(dataStore); err != nil {
		return nil, err
	}

	return dataStore, nil
}

func TestSQLiteDataStore_IncrementVisitCount(t *testing.T) {
	dataStore, err := setupTestDB()
	if err != nil {
		t.Fatalf("failed to set up test DB: %v", err)
	}
	defer dataStore.db.Close()

	tests := []struct {
		name      string
		timestamp time.Time
		wantErr   bool
	}{
		{"Valid Timestamp", time.Now(), false},
		{"Future Timestamp", time.Now().Add(time.Hour), false},
		{"Past Timestamp", time.Now().Add(-time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := dataStore.IncrementVisitCount(tt.timestamp); (err != nil) != tt.wantErr {
				t.Errorf("SQLiteDataStore.IncrementVisitCount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSQLiteDataStore_GetVisitCount(t *testing.T) {
	dataStore, err := setupTestDB()
	if err != nil {
		t.Fatalf("failed to set up test DB: %v", err)
	}
	defer dataStore.db.Close()

	// Increment visit counts as needed
	if err := dataStore.IncrementVisitCount(time.Now()); err != nil {
		t.Fatalf("failed to increment visit count: %v", err)
	}
	if err := dataStore.IncrementVisitCount(time.Now()); err != nil {
		t.Fatalf("failed to increment visit count: %v", err)
	}

	gotCount, err := dataStore.GetVisitCount()
	if err != nil {
		t.Fatalf("failed to get visit count: %v", err)
	}

	if gotCount != 2 {
		t.Errorf("SQLiteDataStore.GetVisitCount() = %v, want %v", gotCount, 2)
	}
}

func TestNewSQLiteDataStore(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name string
		args struct {
			db *sql.DB
		}
		want *SQLiteDataStore
	}{
		{"Create DataStore", struct{ db *sql.DB }{db}, &SQLiteDataStore{db: db}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSQLiteDataStore(tt.args.db)

			// Compare fields explicitly
			if got.db != tt.want.db {
				t.Errorf("NewSQLiteDataStore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createTable(t *testing.T) {
	dataStore, err := setupTestDB()
	if err != nil {
		t.Fatalf("failed to set up test DB: %v", err)
	}
	defer dataStore.db.Close()

	tests := []struct {
		name      string
		dataStore *SQLiteDataStore
		wantErr   bool
	}{
		{"Valid Creation", dataStore, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := createTable(tt.dataStore); (err != nil) != tt.wantErr {
				t.Errorf("createTable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
