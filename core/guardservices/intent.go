// SPDX-License-Identifier: LGPL-3.0-or-later
package guardservices

import (
	"context"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// IntentSigner binds protected custody to one current role-bound Ed25519 key and
// the Guard intent signing domain. It is not approval or an issuance fence. Use
// only through the protected core IntentIssuer, never as a model-facing tool.
type IntentSigner struct{ binding *signingBinding }

// NewIntentSigner validates current registered bytes against fixed key custody.
// Changing the key requires trusted reconfiguration, not an algorithm fallback.
func NewIntentSigner(ctx context.Context, a *g.RegistryAuthority, issuer, keyid string, b Ed25519Backend) (*IntentSigner, error) {
	binding, e := newBinding(ctx, a, issuer, keyid, b)
	if e != nil {
		return nil, e
	}
	return &IntentSigner{binding}, nil
}

// Sign rechecks current authority around bounded backend key use. Core issuance
// still owns full intent validation, policy approval and durable one-use fencing.
func (s *IntentSigner) Sign(ctx context.Context, kid string, raw []byte) ([]byte, error) {
	if s == nil {
		return nil, ErrDenied
	}
	return s.binding.sign(ctx, kid, "sage-execution-intent|0.10.0\x00", raw)
}
