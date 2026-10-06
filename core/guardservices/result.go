// SPDX-License-Identifier: LGPL-3.0-or-later
package guardservices

import (
	"context"
	"crypto/ed25519"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// ResultSigner binds result-key custody and fresh authority to one executor. Its
// zero value is invalid. It does not authorize execution or manufacture outcomes.
type ResultSigner struct{ binding *signingBinding }

// NewResultSigner validates the fixed Ed25519 backend against current registered
// bytes. The core owns result identity, state, exact proof validation and storage.
func NewResultSigner(ctx context.Context, a *g.RegistryAuthority, issuer, keyid string, b Ed25519Backend) (*ResultSigner, error) {
	binding, e := newBinding(ctx, a, issuer, keyid, b)
	if e != nil {
		return nil, e
	}
	return &ResultSigner{binding}, nil
}

// ActiveKey freshly checks the exact configured executor and signing key.
func (s *ResultSigner) ActiveKey(ctx context.Context, issuer, kid string) (ed25519.PublicKey, error) {
	if s == nil || s.binding == nil || issuer != s.binding.issuer || kid != s.binding.keyid {
		return nil, ErrDenied
	}
	return s.binding.fresh(ctx)
}

// Now returns time from a fresh observation of the current configured authority.
func (s *ResultSigner) Now(ctx context.Context) (now int64, err error) {
	defer func() {
		if recover() != nil {
			now, err = 0, ErrDenied
		}
	}()
	if s == nil || s.binding == nil {
		return 0, ErrDenied
	}
	if _, e := s.binding.fresh(ctx); e != nil {
		return 0, e
	}
	now, err = s.binding.authority.Now(ctx)
	if err != nil || !active(ctx) {
		return 0, ErrDenied
	}
	return now, nil
}

// KeyID returns only the fixed key after a fresh activation/custody check.
func (s *ResultSigner) KeyID(ctx context.Context) (string, error) {
	if s == nil || s.binding == nil {
		return "", ErrDenied
	}
	if _, e := s.binding.fresh(ctx); e != nil {
		return "", e
	}
	return s.binding.keyid, nil
}

// Sign uses only the result signing domain, with fresh key checks before and after
// key use. It cannot sign an intent, HPKE handshake or an X25519 KEM operation.
func (s *ResultSigner) Sign(ctx context.Context, kid string, raw []byte) ([]byte, error) {
	if s == nil {
		return nil, ErrDenied
	}
	return s.binding.sign(ctx, kid, "sage-tool-result|0.10.0\x00", raw)
}
