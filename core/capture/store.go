// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import "context"

// Store is protected host custody for original input. Create must atomically
// reserve the ID, be write-once and be durable before success. Load must return
// the exact ordered bytes or refuse incomplete records. The host publishes a
// Request or recovery checkpoint only after Create and verified readback succeed.
// Implementations must be concurrent-safe, bounded and honor cancellation; the
// host protects them from untrusted access, concurrent mutation and rollback.
// Provider errors are not exposed with input bytes by this package.
type Store interface {
	Create(context.Context, string, [][]byte) error
	Load(context.Context, string) ([][]byte, error)
}
