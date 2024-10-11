package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database-related constants
const (
	DATABASE = "visits.db"
)

type DataStore interface {
	IncrementVisitCount(timestamp time.Time) error
	GetVisitCount() (int, error)
}

// SQLiteDataStore struct
type SQLiteDataStore struct {
	db *sql.DB
}

// IncrementVisitCount implements DataStore.IncrementVisitCount
func (s *SQLiteDataStore) IncrementVisitCount(timestamp time.Time) error {
	_, err := s.db.Exec("INSERT INTO visits (timestamp) VALUES (?)", timestamp)
	if err != nil {
		log.Printf("Error incrementing visit count: %v", err)
		return fmt.Errorf("failed to increment visit count: %w", err)
	}
	return nil
}

// GetVisitCount implements DataStore.GetVisitCount
func (s *SQLiteDataStore) GetVisitCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM visits").Scan(&count)
	if err != nil {
		log.Printf("Error getting visit count: %v", err)
		return 0, fmt.Errorf("failed to get visit count: %w", err)
	}
	return count, nil
}

// NewSQLiteDataStore creates a new SQLiteDataStore
func NewSQLiteDataStore(db *sql.DB) *SQLiteDataStore {
	return &SQLiteDataStore{db: db}
}

// init function to set up the database connection
func init() {
	var err error
	dbConn, err := sql.Open("sqlite3", DATABASE)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}

	// Configure connection pool
	dbConn.SetMaxOpenConns(10)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(time.Minute * 5)

	// Create the DataStore
	dataStore := NewSQLiteDataStore(dbConn)

	// Create the visits table if it doesn't exist
	if err := createTable(dataStore); err != nil {
		log.Fatalf("Error creating database table: %v", err)
	}
}

// createTable creates the visits table if it doesn't exist.
func createTable(dataStore *SQLiteDataStore) error {
	log.Println("Attempting to create visits table if not exists")
	_, err := dataStore.db.Exec(`
                CREATE TABLE IF NOT EXISTS visits (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
                )
        `)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	log.Println("Table creation or verification successful")
	return nil
}
