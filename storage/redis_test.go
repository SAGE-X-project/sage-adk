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

//go:build integration

package storage

import (
	"context"
	"errors"
	"testing"
	"time"
)

// setupRedis creates a Redis storage for testing.
// Skips the test if Redis is not available.
func setupRedis(t *testing.T) *RedisStorage {
	t.Helper()

	config := DefaultRedisConfig()
	config.TTL = 5 * time.Second // Short TTL for tests

	storage, err := NewRedisStorage(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clean up test namespace before tests
	ctx := context.Background()
	_ = storage.Clear(ctx, "test")

	t.Cleanup(func() {
		// Clean up after tests
		_ = storage.Clear(ctx, "test")
		storage.Close()
	})

	return storage
}

func TestRedisStorage_Store_Get(t *testing.T) {
	storage := setupRedis(t)
	ctx := context.Background()

	tests := []struct {
		name      string
		namespace string
		key       string
		value     interface{}
		wantErr   bool
	}{
		{
			name:      "string value",
			namespace: "test",
			key:       "key1",
			value:     "hello world",
			wantErr:   false,
		},
		{
			name:      "number value",
			namespace: "test",
			key:       "key2",
			value:     float64(42),
			wantErr:   false,
		},
		{
			name:      "map value",
			namespace: "test",
			key:       "key3",
			value:     map[string]interface{}{"foo": "bar", "num": float64(123)},
			wantErr:   false,
		},
		{
			name:      "empty namespace",
			namespace: "",
			key:       "key4",
			value:     "test",
			wantErr:   true,
		},
		{
			name:      "empty key",
			namespace: "test",
			key:       "",
			value:     "test",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store
			err := storage.Store(ctx, tt.namespace, tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Store() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Get
			got, err := storage.Get(ctx, tt.namespace, tt.key)
			if err != nil {
				t.Errorf("Get() error = %v", err)
				return
			}

			// Compare values
			switch expected := tt.value.(type) {
			case string:
				if got != expected {
					t.Errorf("Get() = %v, want %v", got, expected)
				}
			case float64:
				if got != expected {
					t.Errorf("Get() = %v, want %v", got, expected)
				}
			case map[string]interface{}:
				gotMap, ok := got.(map[string]interface{})
				if !ok {
					t.Errorf("Get() returned wrong type: %T", got)
					return
				}
				for k, v := range expected {
					if gotMap[k] != v {
						t.Errorf("Get() map[%s] = %v, want %v", k, gotMap[k], v)
					}
				}
			}
		})
	}
}

func TestRedisStorage_Get_NotFound(t *testing.T) {
	storage := setupRedis(t)
	ctx := context.Background()

	_, err := storage.Get(ctx, "test", "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestRedisStorage_Delete(t *testing.T) {
	storage := setupRedis(t)
	ctx := context.Background()

	// Store a value
	if err := storage.Store(ctx, "test", "key1", "value1"); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Delete it
	if err := storage.Delete(ctx, "test", "key1"); err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err := storage.Get(ctx, "test", "key1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrNotFound", err)
	}

	// Delete non-existent key
	err = storage.Delete(ctx, "test", "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() non-existent error = %v, want ErrNotFound", err)
	}
}

func TestRedisStorage_List(t *testing.T) {
	storage := setupRedis(t)
	ctx := context.Background()

	// Store multiple values
	values := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, value := range values {
		if err := storage.Store(ctx, "test", key, value); err != nil {
			t.Fatalf("Store() error = %v", err)
		}
	}

	// List all values
	list, err := storage.List(ctx, "test")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != len(values) {
		t.Errorf("List() returned %d items, want %d", len(list), len(values))
	}

	// Verify all values are present
	found := make(map[string]bool)
	for _, item := range list {
		if str, ok := item.(string); ok {
			found[str] = true
		}
	}

	for _, expected := range values {
		if !found[expected.(string)] {
			t.Errorf("List() missing value: %v", expected)
		}
	}
}

func TestRedisStorage_List_Empty(t *testing.T) {
	storage := setupRedis(t)
	ctx := context.Background()

	list, err := storage.List(ctx, "empty-namespace")
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(list) != 0 {
		t.Errorf("List() returned %d items, want 0", len(list))
	}
}

func TestRedisStorage_Clear(t *testing.T) {
	storage := setupRedis(t)
	ctx := context.Background()

	// Store multiple values
	for i := 1; i <= 5; i++ {
		key := "key" + string(rune('0'+i))
		if err := storage.Store(ctx, "test", key, i); err != nil {
			t.Fatalf("Store() error = %v", err)
		}
	}

	// Clear namespace
	if err := storage.Clear(ctx, "test"); err != nil {
		t.Errorf("Clear() error = %v", err)
	}

	// Verify all keys are gone
	list, err := storage.List(ctx, "test")
	if err != nil {
		t.Errorf("List() error = %v", err)
	}

	if len(list) != 0 {
		t.Errorf("List() after Clear() returned %d items, want 0", len(list))
	}
}

func TestRedisStorage_Exists(t *testing.T) {
	storage := setupRedis(t)
	ctx := context.Background()

	// Store a value
	if err := storage.Store(ctx, "test", "key1", "value1"); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Check existence
	exists, err := storage.Exists(ctx, "test", "key1")
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}

	// Check non-existent key
	exists, err = storage.Exists(ctx, "test", "nonexistent")
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Exists() = true, want false")
	}
}

func TestRedisStorage_TTL(t *testing.T) {
	storage := setupRedis(t)
	ctx := context.Background()

	// Store a value
	if err := storage.Store(ctx, "test", "key1", "value1"); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Get TTL
	ttl, err := storage.GetTTL(ctx, "test", "key1")
	if err != nil {
		t.Errorf("GetTTL() error = %v", err)
	}

	if ttl <= 0 || ttl > 5*time.Second {
		t.Errorf("GetTTL() = %v, want between 0 and 5s", ttl)
	}

	// Set new TTL
	if err := storage.SetTTL(ctx, "test", "key1", 10*time.Second); err != nil {
		t.Errorf("SetTTL() error = %v", err)
	}

	// Verify new TTL
	ttl, err = storage.GetTTL(ctx, "test", "key1")
	if err != nil {
		t.Errorf("GetTTL() error = %v", err)
	}

	if ttl <= 5*time.Second || ttl > 10*time.Second {
		t.Errorf("GetTTL() after SetTTL() = %v, want between 5s and 10s", ttl)
	}

	// Remove TTL
	if err := storage.SetTTL(ctx, "test", "key1", 0); err != nil {
		t.Errorf("SetTTL(0) error = %v", err)
	}

	// Verify no expiration
	ttl, err = storage.GetTTL(ctx, "test", "key1")
	if err != nil {
		t.Errorf("GetTTL() error = %v", err)
	}

	if ttl != -1*time.Second {
		t.Errorf("GetTTL() after removing expiration = %v, want -1s", ttl)
	}
}

func TestRedisStorage_Expiration(t *testing.T) {
	config := DefaultRedisConfig()
	config.TTL = 2 * time.Second // Very short TTL

	storage, err := NewRedisStorage(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Store a value
	if err := storage.Store(ctx, "test", "expiring", "value"); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	// Verify it exists
	exists, err := storage.Exists(ctx, "test", "expiring")
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Verify it's gone
	exists, err = storage.Exists(ctx, "test", "expiring")
	if err != nil {
		t.Errorf("Exists() after expiration error = %v", err)
	}
	if exists {
		t.Error("Exists() after expiration = true, want false")
	}
}

func TestRedisStorage_Ping(t *testing.T) {
	storage := setupRedis(t)
	ctx := context.Background()

	if err := storage.Ping(ctx); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestRedisStorage_Close(t *testing.T) {
	config := DefaultRedisConfig()
	storage, err := NewRedisStorage(config)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	if err := storage.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Operations should fail after close
	ctx := context.Background()
	err = storage.Store(ctx, "test", "key", "value")
	if err == nil {
		t.Error("Store() after Close() should fail")
	}
}
