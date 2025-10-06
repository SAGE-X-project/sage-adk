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

	"github.com/sage-x-project/sage-adk/pkg/errors"
)

// MemoryStorage is an in-memory implementation of the Storage interface.
//
// MemoryStorage provides a thread-safe, in-memory key-value store
// organized by namespaces. Data is not persisted and will be lost
// when the process exits.
//
// This implementation is suitable for:
//   - Testing and development
//   - Temporary caching
//   - Single-instance deployments
//
// For production use with persistence, use Redis or PostgreSQL backends.
type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string]map[string]interface{} // namespace -> key -> value
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]map[string]interface{}),
	}
}

// Store stores an item with the given key in a namespace.
func (m *MemoryStorage) Store(ctx context.Context, namespace, key string, value interface{}) error {
	if namespace == "" {
		return errors.ErrInvalidInput.WithMessage("namespace cannot be empty")
	}
	if key == "" {
		return errors.ErrInvalidInput.WithMessage("key cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create namespace if it doesn't exist
	if m.data[namespace] == nil {
		m.data[namespace] = make(map[string]interface{})
	}

	m.data[namespace][key] = value
	return nil
}

// Get retrieves an item by key from a namespace.
func (m *MemoryStorage) Get(ctx context.Context, namespace, key string) (interface{}, error) {
	if namespace == "" {
		return nil, errors.ErrInvalidInput.WithMessage("namespace cannot be empty")
	}
	if key == "" {
		return nil, errors.ErrInvalidInput.WithMessage("key cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	ns, ok := m.data[namespace]
	if !ok {
		return nil, errors.ErrNotFound.WithDetail("namespace", namespace).WithDetail("key", key)
	}

	value, ok := ns[key]
	if !ok {
		return nil, errors.ErrNotFound.WithDetail("namespace", namespace).WithDetail("key", key)
	}

	return value, nil
}

// List retrieves all items in a namespace.
func (m *MemoryStorage) List(ctx context.Context, namespace string) ([]interface{}, error) {
	if namespace == "" {
		return nil, errors.ErrInvalidInput.WithMessage("namespace cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	ns, ok := m.data[namespace]
	if !ok {
		// Return empty slice for non-existent namespace
		return []interface{}{}, nil
	}

	items := make([]interface{}, 0, len(ns))
	for _, value := range ns {
		items = append(items, value)
	}

	return items, nil
}

// Delete removes an item by key from a namespace.
func (m *MemoryStorage) Delete(ctx context.Context, namespace, key string) error {
	if namespace == "" {
		return errors.ErrInvalidInput.WithMessage("namespace cannot be empty")
	}
	if key == "" {
		return errors.ErrInvalidInput.WithMessage("key cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	ns, ok := m.data[namespace]
	if !ok {
		return errors.ErrNotFound.WithDetail("namespace", namespace).WithDetail("key", key)
	}

	if _, ok := ns[key]; !ok {
		return errors.ErrNotFound.WithDetail("namespace", namespace).WithDetail("key", key)
	}

	delete(ns, key)
	return nil
}

// Clear removes all items in a namespace.
func (m *MemoryStorage) Clear(ctx context.Context, namespace string) error {
	if namespace == "" {
		return errors.ErrInvalidInput.WithMessage("namespace cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Delete the entire namespace
	delete(m.data, namespace)
	return nil
}

// Exists checks if a key exists in a namespace.
func (m *MemoryStorage) Exists(ctx context.Context, namespace, key string) (bool, error) {
	if namespace == "" {
		return false, errors.ErrInvalidInput.WithMessage("namespace cannot be empty")
	}
	if key == "" {
		return false, errors.ErrInvalidInput.WithMessage("key cannot be empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	ns, ok := m.data[namespace]
	if !ok {
		return false, nil
	}

	_, ok = ns[key]
	return ok, nil
}
