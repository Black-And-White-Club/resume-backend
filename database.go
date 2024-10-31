package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// mustGetenv retrieves the value of the environment variable or logs a fatal error if not set.
func mustGetenv(k string) (string, error) {
	v := os.Getenv(k)
	if v == "" {
		return "", fmt.Errorf("environment variable not set: %s", k)
	}
	return v, nil
}

// DatabasePool interface includes methods needed for the database operations
type DatabasePool interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) // Use pgx.CommandTag for Exec
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Close()
}

// DataStore interface for data operations
type DataStore interface {
	IncrementVisitCount(ctx context.Context, timestamp time.Time) error
	GetVisitCount(ctx context.Context) (int, error)
	Close()
}

// PostgresStore implements DataStore
type PostgresStore struct {
	pool DatabasePool
}

// IncrementVisitCount increments the visit count in the database
func (s *PostgresStore) IncrementVisitCount(ctx context.Context, timestamp time.Time) error {
	_, err := s.pool.Exec(ctx, "INSERT INTO visits (timestamp) VALUES ($1)", timestamp)
	if err != nil {
		log.Printf("Error incrementing visit count: %v", err)
		return fmt.Errorf("failed to increment visit count: %w", err)
	}
	return nil
}

// GetVisitCount retrieves the visit count from the database
func (s *PostgresStore) GetVisitCount(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM visits").Scan(&count)
	if err != nil {
		log.Printf("Error getting visit count: %v", err)
		return 0, fmt.Errorf("failed to get visit count: %w", err)
	}
	return count, nil
}

// Close closes the database connection pool
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// createTable creates the visits table if it does not exist
func createTable(ctx context.Context, pool DatabasePool) error {
	query := `
		CREATE TABLE IF NOT EXISTS visits (
			id SERIAL PRIMARY KEY,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`

	_, err := pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

// SetupDatabase initializes and configures the database
func SetupDatabase(ctx context.Context) (DataStore, error) {
	dbUser, _ := mustGetenv("DB_USER")         // Ignoring the error
	dbPassword, _ := mustGetenv("DB_PASSWORD") // Ignoring the error
	dbHost, _ := mustGetenv("DB_HOST")         // Ignoring the error
	dbPort, _ := mustGetenv("DB_PORT")         // Ignoring the error
	dbName, _ := mustGetenv("DB_NAME")         // Ignoring the error

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Configure the connection pool
	config.MaxConns = 20
	config.MinConns = 10
	config.MaxConnLifetime = time.Minute * 5

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create table if it doesn't exist
	if err := createTable(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStore{pool: pool}, nil
}
