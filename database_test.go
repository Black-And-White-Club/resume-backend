package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockDatabasePool struct {
	mock.Mock
}

func Test_mustGetenv(t *testing.T) {
	t.Run("Environment variable exists", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "test_value")
		defer os.Unsetenv("TEST_ENV_VAR")

		got, err := mustGetenv("TEST_ENV_VAR") // Now capturing both the string and the error
		if err != nil {
			t.Fatalf("mustGetenv() returned an error: %v", err)
		}
		want := "test_value"
		if got != want {
			t.Errorf("mustGetenv() = %v, want %v", got, want)
		}
	})

	t.Run("Environment variable does not exist", func(t *testing.T) {
		os.Unsetenv("NON_EXISTENT_ENV_VAR")

		got, err := mustGetenv("NON_EXISTENT_ENV_VAR") // This will return an error
		if err == nil {
			t.Fatal("Expected an error, got none")
		}

		want := "environment variable not set: NON_EXISTENT_ENV_VAR"
		if err.Error() != want {
			t.Errorf("mustGetenv() returned error: %v, want %v", err.Error(), want)
		}
		if got != "" {
			t.Errorf("mustGetenv() = %v, want empty string", got)
		}
	})
}

func Test_IncrementVisitCount(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Create an instance of PostgresStore with the mock pool
	s := &PostgresStore{pool: mock} // This works now because mock implements DatabasePool

	ctx := context.Background()
	timestamp := time.Now()

	// Set up expectations
	mock.ExpectExec("INSERT INTO visits").WithArgs(timestamp).WillReturnResult(pgxmock.NewResult("INSERT", 1))

	// Call the method under test
	err = s.IncrementVisitCount(ctx, timestamp)
	assert.NoError(t, err)

	// Ensure all expectations were met
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStore_GetVisitCount(t *testing.T) {
	// Create a mock pool
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func()
		want    int
		wantErr bool
	}{
		{
			name: "success",
			mock: func() {
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM visits").
					WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(10))
			},
			want:    10,
			wantErr: false,
		},
		{
			name: "error",
			mock: func() {
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM visits").
					WillReturnError(fmt.Errorf("query error"))
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock behavior
			tt.mock()

			// Create a new PostgresStore with the mock pool
			s := &PostgresStore{
				pool: mock,
			}

			got, err := s.GetVisitCount(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("PostgresStore.GetVisitCount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PostgresStore.GetVisitCount() = %v, want %v", got, tt.want)
			}

			// Ensure all expectations were met
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func (m *MockDatabasePool) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockDatabasePool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	// Implement this if needed for other tests
	return nil
}

func (m *MockDatabasePool) Close() {
	m.Called()
}

func TestPostgresStore_Close(t *testing.T) {
	tests := []struct {
		name string
		mock func(*MockDatabasePool)
	}{
		{
			name: "close pool",
			mock: func(m *MockDatabasePool) {
				m.On("Close").Once() // Expect Close to be called once
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool := new(MockDatabasePool)
			tt.mock(mockPool)

			s := &PostgresStore{
				pool: mockPool,
			}
			s.Close()

			// Assert that all expectations were met
			mockPool.AssertExpectations(t)
		})
	}
}

func Test_createTable(t *testing.T) {
	// Create a mock pool
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func()
		wantErr bool
	}{
		{
			name: "success",
			mock: func() {
				mockPool.ExpectExec("CREATE TABLE IF NOT EXISTS visits").
					WillReturnResult(pgxmock.NewResult("CREATE", 0))
			},
			wantErr: false,
		},
		{
			name: "error",
			mock: func() {
				mockPool.ExpectExec("CREATE TABLE IF NOT EXISTS visits").
					WillReturnError(fmt.Errorf("query error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock behavior
			tt.mock()

			// Call createTable
			err := createTable(ctx, mockPool)
			if (err != nil) != tt.wantErr {
				t.Errorf("createTable() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Ensure all expectations were met
			require.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestSetupDatabase(t *testing.T) {
	// Create a mock pool
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mockPool.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func()
		want    DataStore // Assuming DataStore is an interface or struct
		wantErr bool
	}{
		{
			name: "success",
			mock: func() {
				mockPool.ExpectExec("CREATE TABLE IF NOT EXISTS visits").
					WillReturnResult(pgxmock.NewResult("CREATE", 0))
			},
			want:    &PostgresStore{pool: mockPool}, // Assuming PostgresStore implements DataStore
			wantErr: false,
		},
		{
			name: "error creating table",
			mock: func() {
				mockPool.ExpectExec("CREATE TABLE IF NOT EXISTS visits").
					WillReturnError(fmt.Errorf("table creation error"))
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock behavior
			tt.mock()

			// Call SetupDatabase
			got, err := SetupDatabase(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetupDatabase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetupDatabase() = %v, want %v", got, tt.want)
			}

			// Ensure all expectations were met
			require.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}
