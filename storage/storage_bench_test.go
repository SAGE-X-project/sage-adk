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
	"fmt"
	"testing"
)

// BenchmarkMemoryStorage_Store benchmarks storing values in memory storage
func BenchmarkMemoryStorage_Store(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()
	value := "test-value-for-benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		if err := store.Store(ctx, "bench", key, value); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryStorage_Get benchmarks retrieving values from memory storage
func BenchmarkMemoryStorage_Get(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()
	value := "test-value-for-benchmark"

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		store.Store(ctx, "bench", key, value)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		if _, err := store.Get(ctx, "bench", key); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryStorage_List benchmarks listing all keys in a namespace
func BenchmarkMemoryStorage_List(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Pre-populate with varying amounts of data
	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			store.Clear(ctx, "bench")
			for i := 0; i < size; i++ {
				key := fmt.Sprintf("key-%d", i)
				store.Store(ctx, "bench", key, "value")
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := store.List(ctx, "bench"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMemoryStorage_Delete benchmarks deleting values
func BenchmarkMemoryStorage_Delete(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := fmt.Sprintf("key-%d", i)
		store.Store(ctx, "bench", key, "value")
		b.StartTimer()

		if err := store.Delete(ctx, "bench", key); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryStorage_Exists benchmarks checking key existence
func BenchmarkMemoryStorage_Exists(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		store.Store(ctx, "bench", key, "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		if _, err := store.Exists(ctx, "bench", key); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryStorage_Clear benchmarks clearing a namespace
func BenchmarkMemoryStorage_Clear(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	sizes := []int{10, 100, 1000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				// Pre-populate
				for j := 0; j < size; j++ {
					key := fmt.Sprintf("key-%d", j)
					store.Store(ctx, "bench", key, "value")
				}
				b.StartTimer()

				if err := store.Clear(ctx, "bench"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMemoryStorage_Concurrent benchmarks concurrent access
func BenchmarkMemoryStorage_Concurrent(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			store.Store(ctx, "bench", key, "value")
			store.Get(ctx, "bench", key)
			i++
		}
	})
}

// BenchmarkMemoryStorage_LargeValue benchmarks storing large values
func BenchmarkMemoryStorage_LargeValue(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	// Test with different value sizes
	sizes := []int{1024, 10240, 102400} // 1KB, 10KB, 100KB
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%dB", size), func(b *testing.B) {
			value := make([]byte, size)
			for i := range value {
				value[i] = byte(i % 256)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i)
				if err := store.Store(ctx, "bench", key, value); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkMemoryStorage_ComplexValue benchmarks storing complex data structures
func BenchmarkMemoryStorage_ComplexValue(b *testing.B) {
	store := NewMemoryStorage()
	ctx := context.Background()

	complexValue := map[string]interface{}{
		"string": "test-value",
		"number": 12345,
		"bool":   true,
		"nested": map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
		"array": []interface{}{1, 2, 3, 4, 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		if err := store.Store(ctx, "bench", key, complexValue); err != nil {
			b.Fatal(err)
		}
	}
}
