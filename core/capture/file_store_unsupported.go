//go:build !linux && !darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import "context"

// FileStore is unavailable on platforms without the audited file custody path.
type FileStore struct{}

// OpenFileStore refuses unsupported platforms; no weaker storage fallback exists.
func OpenFileStore(string) (*FileStore, error) { return nil, ErrCapture }

// Close refuses an unavailable store.
func (*FileStore) Close() error { return ErrCapture }

// Create refuses an unavailable store.
func (*FileStore) Create(context.Context, string, [][]byte) error { return ErrCapture }

// Load refuses an unavailable store.
func (*FileStore) Load(context.Context, string) ([][]byte, error) { return nil, ErrCapture }
