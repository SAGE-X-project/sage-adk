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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorage implements Storage using Redis.
type RedisStorage struct {
	client *redis.Client
	ttl    time.Duration
}

// RedisConfig contains Redis connection configuration.
type RedisConfig struct {
	// Address is the Redis server address (host:port).
	// Default: "localhost:6379"
	Address string

	// Password is the Redis password.
	// Default: "" (no password)
	Password string

	// DB is the Redis database number.
	// Default: 0
	DB int

	// TTL is the default time-to-live for keys.
	// Default: 24 hours
	// Set to 0 for no expiration.
	TTL time.Duration

	// PoolSize is the maximum number of socket connections.
	// Default: 10 connections per CPU
	PoolSize int

	// MinIdleConns is the minimum number of idle connections.
	// Default: 2
	MinIdleConns int

	// MaxRetries is the maximum number of retries before giving up.
	// Default: 3
	MaxRetries int

	// DialTimeout is the timeout for establishing new connections.
	// Default: 5 seconds
	DialTimeout time.Duration

	// ReadTimeout is the timeout for socket reads.
	// Default: 3 seconds
	ReadTimeout time.Duration

	// WriteTimeout is the timeout for socket writes.
	// Default: 3 seconds
	WriteTimeout time.Duration
}

// DefaultRedisConfig returns the default Redis configuration.
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Address:      "localhost:6379",
		Password:     "",
		DB:           0,
		TTL:          24 * time.Hour,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// NewRedisStorage creates a new Redis storage instance.
//
// Example:
//
//	storage := storage.NewRedisStorage(&storage.RedisConfig{
//	    Address: "localhost:6379",
//	    DB:      0,
//	    TTL:     24 * time.Hour,
//	})
func NewRedisStorage(config *RedisConfig) (*RedisStorage, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:         config.Address,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStorage{
		client: client,
		ttl:    config.TTL,
	}, nil
}

// Store stores an item with the given key in a namespace.
func (s *RedisStorage) Store(ctx context.Context, namespace, key string, value interface{}) error {
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

	// Build Redis key
	redisKey := s.buildKey(namespace, key)

	// Store in Redis
	if err := s.client.Set(ctx, redisKey, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to store value: %w", err)
	}

	return nil
}

// Get retrieves an item by key from a namespace.
func (s *RedisStorage) Get(ctx context.Context, namespace, key string) (interface{}, error) {
	if namespace == "" {
		return nil, errors.New("namespace cannot be empty")
	}
	if key == "" {
		return nil, errors.New("key cannot be empty")
	}

	// Build Redis key
	redisKey := s.buildKey(namespace, key)

	// Get from Redis
	data, err := s.client.Get(ctx, redisKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
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
func (s *RedisStorage) List(ctx context.Context, namespace string) ([]interface{}, error) {
	if namespace == "" {
		return nil, errors.New("namespace cannot be empty")
	}

	// Build pattern for namespace
	pattern := s.buildKey(namespace, "*")

	// Get all keys matching pattern
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		return []interface{}{}, nil
	}

	// Get all values
	values := make([]interface{}, 0, len(keys))
	for _, key := range keys {
		data, err := s.client.Get(ctx, key).Bytes()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue // Key was deleted between Keys and Get
			}
			return nil, fmt.Errorf("failed to get value for key %s: %w", key, err)
		}

		var value interface{}
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal value for key %s: %w", key, err)
		}

		values = append(values, value)
	}

	return values, nil
}

// Delete removes an item by key from a namespace.
func (s *RedisStorage) Delete(ctx context.Context, namespace, key string) error {
	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}
	if key == "" {
		return errors.New("key cannot be empty")
	}

	// Build Redis key
	redisKey := s.buildKey(namespace, key)

	// Delete from Redis
	result, err := s.client.Del(ctx, redisKey).Result()
	if err != nil {
		return fmt.Errorf("failed to delete value: %w", err)
	}

	if result == 0 {
		return ErrNotFound
	}

	return nil
}

// Clear removes all items in a namespace.
func (s *RedisStorage) Clear(ctx context.Context, namespace string) error {
	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}

	// Build pattern for namespace
	pattern := s.buildKey(namespace, "*")

	// Get all keys matching pattern
	keys, err := s.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		return nil // Nothing to clear
	}

	// Delete all keys
	if err := s.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to clear namespace: %w", err)
	}

	return nil
}

// Exists checks if a key exists in a namespace.
func (s *RedisStorage) Exists(ctx context.Context, namespace, key string) (bool, error) {
	if namespace == "" {
		return false, errors.New("namespace cannot be empty")
	}
	if key == "" {
		return false, errors.New("key cannot be empty")
	}

	// Build Redis key
	redisKey := s.buildKey(namespace, key)

	// Check existence
	result, err := s.client.Exists(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return result > 0, nil
}

// Close closes the Redis connection.
func (s *RedisStorage) Close() error {
	return s.client.Close()
}

// Ping checks if the Redis connection is alive.
func (s *RedisStorage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// SetTTL updates the TTL for a specific key.
func (s *RedisStorage) SetTTL(ctx context.Context, namespace, key string, ttl time.Duration) error {
	if namespace == "" {
		return errors.New("namespace cannot be empty")
	}
	if key == "" {
		return errors.New("key cannot be empty")
	}

	// Build Redis key
	redisKey := s.buildKey(namespace, key)

	// Set TTL
	var err error
	if ttl == 0 {
		// Remove expiration
		err = s.client.Persist(ctx, redisKey).Err()
	} else {
		// Set expiration
		err = s.client.Expire(ctx, redisKey, ttl).Err()
	}

	if err != nil {
		return fmt.Errorf("failed to set TTL: %w", err)
	}

	return nil
}

// GetTTL returns the remaining TTL for a key.
// Returns -1 if the key has no expiration.
// Returns -2 if the key does not exist.
func (s *RedisStorage) GetTTL(ctx context.Context, namespace, key string) (time.Duration, error) {
	if namespace == "" {
		return 0, errors.New("namespace cannot be empty")
	}
	if key == "" {
		return 0, errors.New("key cannot be empty")
	}

	// Build Redis key
	redisKey := s.buildKey(namespace, key)

	// Get TTL
	ttl, err := s.client.TTL(ctx, redisKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

// buildKey builds a Redis key from namespace and key.
func (s *RedisStorage) buildKey(namespace, key string) string {
	return fmt.Sprintf("sage:%s:%s", namespace, key)
}
