// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresStorage implements Storage using PostgreSQL.
type PostgresStorage struct {
	db        *sql.DB
	tableName string
}

// PostgresConfig contains PostgreSQL connection configuration.
type PostgresConfig struct {
	// Host is the PostgreSQL server host.
	// Default: "localhost"
	Host string

	// Port is the PostgreSQL server port.
	// Default: 5432
	Port int

	// User is the PostgreSQL user.
	// Default: "postgres"
	User string

	// Password is the PostgreSQL password.
	// Default: ""
	Password string

	// Database is the PostgreSQL database name.
	// Default: "sage"
	Database string

	// SSLMode is the SSL mode for connection.
	// Options: "disable", "require", "verify-ca", "verify-full"
	// Default: "disable"
	SSLMode string

	// TableName is the name of the table to store data.
	// Default: "sage_storage"
	TableName string

	// MaxOpenConns is the maximum number of open connections.
	// Default: 25
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	// Default: 5
	MaxIdleConns int

	// ConnMaxLifetime is the maximum lifetime of a connection.
	// Default: 5 minutes
	ConnMaxLifetime time.Duration

	// AutoMigrate automatically creates the table if it doesn't exist.
	// Default: true
	AutoMigrate bool
}

// DefaultPostgresConfig returns the default PostgreSQL configuration.
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "",
		Database:        "sage",
		SSLMode:         "disable",
		TableName:       "sage_storage",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		AutoMigrate:     true,
	}
}

// NewPostgresStorage creates a new PostgreSQL storage instance.
//
// Example:
//
//	storage := storage.NewPostgresStorage(&storage.PostgresConfig{
//	    Host:     "localhost",
//	    Database: "sage",
//	    User:     "postgres",
//	    Password: "password",
//	})
func NewPostgresStorage(config *PostgresConfig) (*PostgresStorage, error) {
	if config == nil {
		config = DefaultPostgresConfig()
	}

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	storage := &PostgresStorage{
		db:        db,
		tableName: config.TableName,
	}

	// Auto-migrate if enabled
	if config.AutoMigrate {
		if err := storage.migrate(ctx); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to migrate database: %w", err)
		}
	}

	return storage, nil
}

// migrate creates the storage table if it doesn't exist.
func (s *PostgresStorage) migrate(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			namespace VARCHAR(255) NOT NULL,
			key VARCHAR(255) NOT NULL,
			value JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (namespace, key)
		);

		CREATE INDEX IF NOT EXISTS idx_%s_namespace ON %s(namespace);
		CREATE INDEX IF NOT EXISTS idx_%s_created_at ON %s(created_at);
		CREATE INDEX IF NOT EXISTS idx_%s_updated_at ON %s(updated_at);
	`, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName)

	_, err := s.db.ExecContext(ctx, query)
	return err
}

// Store stores an item with the given key in a namespace.
func (s *PostgresStorage) Store(ctx context.Context, namespace, key string, value interface{}) error {
	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}
	if key == "" {
		return errors.New("key cannot be empty")
	}

	// Serialize value to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Upsert into database
	query := fmt.Sprintf(`
		INSERT INTO %s (namespace, key, value, created_at, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (namespace, key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP
	`, s.tableName)

	_, err = s.db.ExecContext(ctx, query, namespace, key, data)
	if err != nil {
		return fmt.Errorf("failed to store value: %w", err)
	}

	return nil
}

// Get retrieves an item by key from a namespace.
func (s *PostgresStorage) Get(ctx context.Context, namespace, key string) (interface{}, error) {
	if namespace == "" {
		return nil, errors.New("namespace cannot be empty")
	}
	if key == "" {
		return nil, errors.New("key cannot be empty")
	}

	// Query from database
	query := fmt.Sprintf(`
		SELECT value FROM %s
		WHERE namespace = $1 AND key = $2
	`, s.tableName)

	var data []byte
	err := s.db.QueryRowContext(ctx, query, namespace, key).Scan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get value: %w", err)
	}

	// Deserialize JSON
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return value, nil
}

// List retrieves all items in a namespace.
func (s *PostgresStorage) List(ctx context.Context, namespace string) ([]interface{}, error) {
	if namespace == "" {
		return nil, errors.New("namespace cannot be empty")
	}

	// Query from database
	query := fmt.Sprintf(`
		SELECT value FROM %s
		WHERE namespace = $1
		ORDER BY created_at ASC
	`, s.tableName)

	rows, err := s.db.QueryContext(ctx, query, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list values: %w", err)
	}
	defer rows.Close()

	values := make([]interface{}, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		var value interface{}
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal value: %w", err)
		}

		values = append(values, value)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return values, nil
}

// Delete removes an item by key from a namespace.
func (s *PostgresStorage) Delete(ctx context.Context, namespace, key string) error {
	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}
	if key == "" {
		return errors.New("key cannot be empty")
	}

	// Delete from database
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE namespace = $1 AND key = $2
	`, s.tableName)

	result, err := s.db.ExecContext(ctx, query, namespace, key)
	if err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// Clear removes all items in a namespace.
func (s *PostgresStorage) Clear(ctx context.Context, namespace string) error {
	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}

	// Delete from database
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE namespace = $1
	`, s.tableName)

	_, err := s.db.ExecContext(ctx, query, namespace)
	if err != nil {
		return fmt.Errorf("failed to clear namespace: %w", err)
	}

	return nil
}

// Exists checks if a key exists in a namespace.
func (s *PostgresStorage) Exists(ctx context.Context, namespace, key string) (bool, error) {
	if namespace == "" {
		return false, errors.New("namespace cannot be empty")
	}
	if key == "" {
		return false, errors.New("key cannot be empty")
	}

	// Check existence
	query := fmt.Sprintf(`
		SELECT EXISTS(
			SELECT 1 FROM %s
			WHERE namespace = $1 AND key = $2
		)
	`, s.tableName)

	var exists bool
	err := s.db.QueryRowContext(ctx, query, namespace, key).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return exists, nil
}

// Close closes the database connection.
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// Ping checks if the database connection is alive.
func (s *PostgresStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Count returns the number of items in a namespace.
func (s *PostgresStorage) Count(ctx context.Context, namespace string) (int, error) {
	if namespace == "" {
		return 0, errors.New("namespace cannot be empty")
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE namespace = $1
	`, s.tableName)

	var count int
	err := s.db.QueryRowContext(ctx, query, namespace).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count items: %w", err)
	}

	return count, nil
}

// ListNamespaces returns all unique namespaces.
func (s *PostgresStorage) ListNamespaces(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT namespace FROM %s
		ORDER BY namespace ASC
	`, s.tableName)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	defer rows.Close()

	namespaces := make([]string, 0)
	for rows.Next() {
		var namespace string
		if err := rows.Scan(&namespace); err != nil {
			return nil, fmt.Errorf("failed to scan namespace: %w", err)
		}
		namespaces = append(namespaces, namespace)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return namespaces, nil
}

// GetWithMetadata retrieves an item with its metadata.
func (s *PostgresStorage) GetWithMetadata(ctx context.Context, namespace, key string) (interface{}, map[string]interface{}, error) {
	if namespace == "" {
		return nil, nil, errors.New("namespace cannot be empty")
	}
	if key == "" {
		return nil, nil, errors.New("key cannot be empty")
	}

	query := fmt.Sprintf(`
		SELECT value, created_at, updated_at FROM %s
		WHERE namespace = $1 AND key = $2
	`, s.tableName)

	var data []byte
	var createdAt, updatedAt time.Time
	err := s.db.QueryRowContext(ctx, query, namespace, key).Scan(&data, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("failed to get value: %w", err)
	}

	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	metadata := map[string]interface{}{
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	return value, metadata, nil
}
