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
)

// Storage defines the interface for data storage.
//
// Storage provides a simple key-value store with namespace support.
// Namespaces help organize different types of data:
//   - history:<agent-id>: Message history
//   - metadata:<agent-id>: Agent metadata
//   - context:<context-id>: Conversation context
//   - state:<agent-id>: Agent state
type Storage interface {
	// Store stores an item with the given key in a namespace.
	Store(ctx context.Context, namespace, key string, value interface{}) error

	// Get retrieves an item by key from a namespace.
	// Returns ErrNotFound if the key does not exist.
	Get(ctx context.Context, namespace, key string) (interface{}, error)

	// List retrieves all items in a namespace.
	// Returns an empty slice if the namespace is empty.
	List(ctx context.Context, namespace string) ([]interface{}, error)

	// Delete removes an item by key from a namespace.
	// Returns ErrNotFound if the key does not exist.
	Delete(ctx context.Context, namespace, key string) error

	// Clear removes all items in a namespace.
	// Does not return an error if the namespace is empty.
	Clear(ctx context.Context, namespace string) error

	// Exists checks if a key exists in a namespace.
	Exists(ctx context.Context, namespace, key string) (bool, error)
}
