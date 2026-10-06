// SPDX-License-Identifier: LGPL-3.0-or-later
package guardservices

import (
	"context"
	"crypto/ed25519"
)

// Ed25519Backend is one fixed host-owned signing key. Implement it with protected
// local or remote custody; PublicKey and Sign must be bounded, cancellation-aware,
// concurrency-safe, non-reentrant and immutable for the binding's lifetime.
// Return owned bytes and sign the supplied bytes exactly. No key export or KEM
// role is required. The backend itself must never reach model/plugin capabilities.
type Ed25519Backend interface {
	PublicKey(context.Context) (ed25519.PublicKey, error)
	Sign(context.Context, []byte) ([]byte, error)
}
