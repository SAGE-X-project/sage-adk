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
// +build integration

package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Run Redis container before tests:
// docker run -d -p 6381:6379 --name sage-redis redis:7-alpine
// OR use: docker-compose -f docker-compose.test.yml up -d

func getTestRedisConfig() *RedisConfig {
	config := DefaultRedisConfig()
	config.Address = "localhost:6381"
	return config
}

func TestRedisStorage_Integration(t *testing.T) {
	store, err := NewRedisStorage(getTestRedisConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Clear test data
	err = store.Clear(ctx, "test")
	assert.NoError(t, err)

	// Store
	err = store.Store(ctx, "test", "key1", "value1")
	assert.NoError(t, err)

	// Get
	val, err := store.Get(ctx, "test", "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	// Store multiple
	err = store.Store(ctx, "test", "key2", "value2")
	assert.NoError(t, err)
	err = store.Store(ctx, "test", "key3", "value3")
	assert.NoError(t, err)

	// List
	list, err := store.List(ctx, "test")
	assert.NoError(t, err)
	assert.Len(t, list, 3)

	// Delete
	err = store.Delete(ctx, "test", "key1")
	assert.NoError(t, err)

	_, err = store.Get(ctx, "test", "key1")
	assert.Error(t, err)

	// Exists
	exists, err := store.Exists(ctx, "test", "key2")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = store.Exists(ctx, "test", "key1")
	assert.NoError(t, err)
	assert.False(t, exists)

	// Clear
	err = store.Clear(ctx, "test")
	assert.NoError(t, err)

	list, err = store.List(ctx, "test")
	assert.NoError(t, err)
	assert.Len(t, list, 0)
}

func TestRedisStorage_ConnectionFailure(t *testing.T) {
	// Try to connect to invalid address
	invalidConfig := DefaultRedisConfig()
	invalidConfig.Address = "invalid-host:9999"
	store, err := NewRedisStorage(invalidConfig)
	require.NoError(t, err) // Connection is created, but not tested yet

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Actual connection will fail on operation
	err = store.Store(ctx, "test", "key", "value")
	assert.Error(t, err)
}

func TestRedisStorage_Timeout(t *testing.T) {
	store, err := NewRedisStorage(getTestRedisConfig())
	require.NoError(t, err)
	defer store.Close()

	// Very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Should timeout
	time.Sleep(1 * time.Millisecond)
	err = store.Store(ctx, "test", "key", "value")
	assert.Error(t, err)
}

func TestRedisStorage_Concurrent(t *testing.T) {
	store, err := NewRedisStorage(getTestRedisConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()
	err = store.Clear(ctx, "concurrent")
	require.NoError(t, err)

	// 100 concurrent goroutines
	const numGoroutines = 100
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			key := fmt.Sprintf("key-%d", id)
			value := fmt.Sprintf("value-%d", id)

			if err := store.Store(ctx, "concurrent", key, value); err != nil {
				errors <- err
				done <- false
				return
			}

			val, err := store.Get(ctx, "concurrent", key)
			if err != nil {
				errors <- err
				done <- false
				return
			}

			if val != value {
				errors <- fmt.Errorf("expected %s, got %s", value, val)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		if <-done {
			successCount++
		}
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent error: %v", err)
	}

	assert.Equal(t, numGoroutines, successCount)

	// Verify all items stored
	list, err := store.List(ctx, "concurrent")
	assert.NoError(t, err)
	assert.Len(t, list, numGoroutines)
}

func TestRedisStorage_LargeData(t *testing.T) {
	store, err := NewRedisStorage(getTestRedisConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// 1MB data
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// Store
	start := time.Now()
	err = store.Store(ctx, "test", "large", largeData)
	storeDuration := time.Since(start)

	assert.NoError(t, err)
	t.Logf("Stored 1MB in %v", storeDuration)

	// Get
	start = time.Now()
	val, err := store.Get(ctx, "test", "large")
	getDuration := time.Since(start)

	assert.NoError(t, err)

	// Compare data
	valBytes, ok := val.([]byte)
	assert.True(t, ok)
	assert.Equal(t, largeData, valBytes)

	t.Logf("Retrieved 1MB in %v", getDuration)

	// Cleanup
	err = store.Delete(ctx, "test", "large")
	assert.NoError(t, err)
}

func TestRedisStorage_ComplexTypes(t *testing.T) {
	store, err := NewRedisStorage(getTestRedisConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Test with map
	complexData := map[string]interface{}{
		"string": "value",
		"number": 42,
		"bool":   true,
		"nested": map[string]interface{}{
			"key": "value",
		},
		"array": []interface{}{1, 2, 3},
	}

	err = store.Store(ctx, "test", "complex", complexData)
	assert.NoError(t, err)

	val, err := store.Get(ctx, "test", "complex")
	assert.NoError(t, err)
	assert.NotNil(t, val)

	// Cleanup
	err = store.Delete(ctx, "test", "complex")
	assert.NoError(t, err)
}

func TestRedisStorage_NamespaceIsolation(t *testing.T) {
	store, err := NewRedisStorage(getTestRedisConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Store in different namespaces
	err = store.Store(ctx, "ns1", "key", "value1")
	assert.NoError(t, err)

	err = store.Store(ctx, "ns2", "key", "value2")
	assert.NoError(t, err)

	// Get from different namespaces
	val1, err := store.Get(ctx, "ns1", "key")
	assert.NoError(t, err)
	assert.Equal(t, "value1", val1)

	val2, err := store.Get(ctx, "ns2", "key")
	assert.NoError(t, err)
	assert.Equal(t, "value2", val2)

	// List should be isolated
	list1, err := store.List(ctx, "ns1")
	assert.NoError(t, err)
	assert.Len(t, list1, 1)

	list2, err := store.List(ctx, "ns2")
	assert.NoError(t, err)
	assert.Len(t, list2, 1)

	// Clear one namespace shouldn't affect the other
	err = store.Clear(ctx, "ns1")
	assert.NoError(t, err)

	list1, err = store.List(ctx, "ns1")
	assert.NoError(t, err)
	assert.Len(t, list1, 0)

	list2, err = store.List(ctx, "ns2")
	assert.NoError(t, err)
	assert.Len(t, list2, 1)

	// Cleanup
	err = store.Clear(ctx, "ns2")
	assert.NoError(t, err)
}

func TestRedisStorage_ErrorHandling(t *testing.T) {
	store, err := NewRedisStorage(getTestRedisConfig())
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Get non-existent key
	_, err = store.Get(ctx, "test", "nonexistent")
	assert.Error(t, err)

	// Delete non-existent key (should not error)
	err = store.Delete(ctx, "test", "nonexistent")
	assert.NoError(t, err)

	// Empty namespace
	err = store.Store(ctx, "", "key", "value")
	assert.Error(t, err)

	// Empty key
	err = store.Store(ctx, "test", "", "value")
	assert.Error(t, err)
}
