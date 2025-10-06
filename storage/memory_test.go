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
	"sync"
	"testing"

	"github.com/sage-x-project/sage-adk/pkg/errors"
)

func TestNewMemoryStorage(t *testing.T) {
	store := NewMemoryStorage()

	if store == nil {
		t.Fatal("NewMemoryStorage() should not return nil")
	}
}

func TestMemoryStorage_Store_Success(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	value := map[string]string{"key": "value"}
	err := store.Store(ctx, "test", "key1", value)

	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}
}

func TestMemoryStorage_Store_InvalidNamespace(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	err := store.Store(ctx, "", "key1", "value")

	if err == nil {
		t.Error("Store() with empty namespace should return error")
	}
}

func TestMemoryStorage_Store_InvalidKey(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	err := store.Store(ctx, "test", "", "value")

	if err == nil {
		t.Error("Store() with empty key should return error")
	}
}

func TestMemoryStorage_Get_Success(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	expected := "test-value"
	store.Store(ctx, "test", "key1", expected)

	got, err := store.Get(ctx, "test", "key1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != expected {
		t.Errorf("Get() = %v, want %v", got, expected)
	}
}

func TestMemoryStorage_Get_NotFound(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	_, err := store.Get(ctx, "test", "nonexistent")

	if err == nil {
		t.Error("Get() should return ErrNotFound for nonexistent key")
	}

	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryStorage_Get_InvalidNamespace(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	_, err := store.Get(ctx, "", "key1")

	if err == nil {
		t.Error("Get() with empty namespace should return error")
	}
}

func TestMemoryStorage_Get_InvalidKey(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	_, err := store.Get(ctx, "test", "")

	if err == nil {
		t.Error("Get() with empty key should return error")
	}
}

func TestMemoryStorage_List_Success(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Store multiple items
	store.Store(ctx, "test", "key1", "value1")
	store.Store(ctx, "test", "key2", "value2")
	store.Store(ctx, "test", "key3", "value3")

	items, err := store.List(ctx, "test")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 3 {
		t.Errorf("List() returned %d items, want 3", len(items))
	}
}

func TestMemoryStorage_List_Empty(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	items, err := store.List(ctx, "empty-namespace")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("List() returned %d items, want 0", len(items))
	}
}

func TestMemoryStorage_List_InvalidNamespace(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	_, err := store.List(ctx, "")

	if err == nil {
		t.Error("List() with empty namespace should return error")
	}
}

func TestMemoryStorage_Delete_Success(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	store.Store(ctx, "test", "key1", "value1")

	err := store.Delete(ctx, "test", "key1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err = store.Get(ctx, "test", "key1")
	if !errors.Is(err, errors.ErrNotFound) {
		t.Error("Get() after Delete() should return ErrNotFound")
	}
}

func TestMemoryStorage_Delete_NotFound(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	err := store.Delete(ctx, "test", "nonexistent")

	if err == nil {
		t.Error("Delete() should return ErrNotFound for nonexistent key")
	}

	if !errors.Is(err, errors.ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryStorage_Delete_InvalidNamespace(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	err := store.Delete(ctx, "", "key1")

	if err == nil {
		t.Error("Delete() with empty namespace should return error")
	}
}

func TestMemoryStorage_Delete_InvalidKey(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	err := store.Delete(ctx, "test", "")

	if err == nil {
		t.Error("Delete() with empty key should return error")
	}
}

func TestMemoryStorage_Clear_Success(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Store multiple items
	store.Store(ctx, "test", "key1", "value1")
	store.Store(ctx, "test", "key2", "value2")

	err := store.Clear(ctx, "test")
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify clearing
	items, _ := store.List(ctx, "test")
	if len(items) != 0 {
		t.Errorf("List() after Clear() returned %d items, want 0", len(items))
	}
}

func TestMemoryStorage_Clear_EmptyNamespace(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Should not error even if namespace is empty
	err := store.Clear(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Clear() on empty namespace should not error, got: %v", err)
	}
}

func TestMemoryStorage_Clear_InvalidNamespace(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	err := store.Clear(ctx, "")

	if err == nil {
		t.Error("Clear() with empty namespace should return error")
	}
}

func TestMemoryStorage_Exists_True(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	store.Store(ctx, "test", "key1", "value1")

	exists, err := store.Exists(ctx, "test", "key1")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if !exists {
		t.Error("Exists() = false, want true")
	}
}

func TestMemoryStorage_Exists_False(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	exists, err := store.Exists(ctx, "test", "nonexistent")
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}

	if exists {
		t.Error("Exists() = true, want false")
	}
}

func TestMemoryStorage_Exists_InvalidNamespace(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	_, err := store.Exists(ctx, "", "key1")

	if err == nil {
		t.Error("Exists() with empty namespace should return error")
	}
}

func TestMemoryStorage_Exists_InvalidKey(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	_, err := store.Exists(ctx, "test", "")

	if err == nil {
		t.Error("Exists() with empty key should return error")
	}
}

func TestMemoryStorage_NamespaceIsolation(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Store same key in different namespaces
	store.Store(ctx, "namespace1", "key1", "value1")
	store.Store(ctx, "namespace2", "key1", "value2")

	// Verify isolation
	val1, _ := store.Get(ctx, "namespace1", "key1")
	val2, _ := store.Get(ctx, "namespace2", "key1")

	if val1 == val2 {
		t.Error("Namespaces should be isolated")
	}

	if val1 != "value1" {
		t.Errorf("namespace1 value = %v, want value1", val1)
	}

	if val2 != "value2" {
		t.Errorf("namespace2 value = %v, want value2", val2)
	}
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key" + string(rune(n))
			store.Store(ctx, "test", key, n)
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key" + string(rune(n))
			store.Get(ctx, "test", key)
		}(i)
	}

	wg.Wait()

	// Should not panic and should have stored values
	items, _ := store.List(ctx, "test")
	if len(items) == 0 {
		t.Error("Concurrent Store() should have stored items")
	}
}

func TestMemoryStorage_TypePreservation(t *testing.T) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Test string
	store.Store(ctx, "test", "string", "test-string")
	got, err := store.Get(ctx, "test", "string")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if s, ok := got.(string); !ok || s != "test-string" {
		t.Errorf("string: got %v (type %T), want test-string (type string)", got, got)
	}

	// Test int
	store.Store(ctx, "test", "int", 42)
	got, err = store.Get(ctx, "test", "int")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if i, ok := got.(int); !ok || i != 42 {
		t.Errorf("int: got %v (type %T), want 42 (type int)", got, got)
	}

	// Test float
	store.Store(ctx, "test", "float", 3.14)
	got, err = store.Get(ctx, "test", "float")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if f, ok := got.(float64); !ok || f != 3.14 {
		t.Errorf("float: got %v (type %T), want 3.14 (type float64)", got, got)
	}

	// Test bool
	store.Store(ctx, "test", "bool", true)
	got, err = store.Get(ctx, "test", "bool")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if b, ok := got.(bool); !ok || b != true {
		t.Errorf("bool: got %v (type %T), want true (type bool)", got, got)
	}

	// Test struct
	type testStruct struct{ Name string }
	original := testStruct{Name: "test"}
	store.Store(ctx, "test", "struct", original)
	got, err = store.Get(ctx, "test", "struct")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if s, ok := got.(testStruct); !ok || s.Name != "test" {
		t.Errorf("struct: got %v (type %T), want %v (type testStruct)", got, got, original)
	}
}
