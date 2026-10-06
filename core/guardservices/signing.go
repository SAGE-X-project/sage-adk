// SPDX-License-Identifier: LGPL-3.0-or-later
package guardservices

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	adkerrors "github.com/sage-x-project/sage-adk/pkg/errors"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	"reflect"
)

// ErrDenied refuses unavailable, inconsistent or out-of-scope provider calls.
// It does not reveal backend errors, signing bytes or original request content.
var ErrDenied = &adkerrors.Error{Category: adkerrors.CategorySecurity, Code: "GUARD_PROVIDER_DENIED", Message: "guard provider operation denied"}

func active(ctx context.Context) bool { return ctx != nil && ctx.Err() == nil }
func missing(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}

type signingBinding struct {
	authority     *g.RegistryAuthority
	issuer, keyid string
	public        ed25519.PublicKey
	backend       Ed25519Backend
}

func newBinding(ctx context.Context, a *g.RegistryAuthority, issuer, keyid string, b Ed25519Backend) (binding *signingBinding, err error) {
	defer func() {
		if recover() != nil {
			binding = nil
			err = ErrDenied
		}
	}()
	if !active(ctx) || a == nil || missing(b) {
		return nil, ErrDenied
	}
	pub, e := b.PublicKey(ctx)
	if e != nil || len(pub) != ed25519.PublicKeySize || !active(ctx) {
		return nil, ErrDenied
	}
	owned := append(ed25519.PublicKey(nil), pub...)
	current, e := a.ActiveKey(ctx, issuer, keyid)
	if e != nil || !bytes.Equal(owned, current) || !active(ctx) {
		return nil, ErrDenied
	}
	return &signingBinding{authority: a, issuer: issuer, keyid: keyid, public: owned, backend: b}, nil
}
func (b *signingBinding) fresh(ctx context.Context) (pub ed25519.PublicKey, err error) {
	defer func() {
		if recover() != nil {
			pub = nil
			err = ErrDenied
		}
	}()
	if b == nil || !active(ctx) {
		return nil, ErrDenied
	}
	actual, e := b.backend.PublicKey(ctx)
	if e != nil || !bytes.Equal(actual, b.public) || !active(ctx) {
		return nil, ErrDenied
	}
	current, e := b.authority.ActiveKey(ctx, b.issuer, b.keyid)
	if e != nil || !bytes.Equal(current, b.public) || !active(ctx) {
		return nil, ErrDenied
	}
	return append(ed25519.PublicKey(nil), current...), nil
}
func (b *signingBinding) sign(ctx context.Context, kid, domain string, raw []byte) (proof []byte, err error) {
	defer func() {
		if recover() != nil {
			proof = nil
			err = ErrDenied
		}
	}()
	if b == nil || !active(ctx) || kid != b.keyid || len(raw) > g.MaxBytes+len(domain) || !bytes.HasPrefix(raw, []byte(domain)) {
		return nil, ErrDenied
	}
	owned := append([]byte(nil), raw...)
	body := owned[len(domain):]
	canonical, e := g.Canonicalize(body)
	if e != nil || len(canonical) == 0 || canonical[0] != '{' || !bytes.Equal(canonical, body) {
		return nil, ErrDenied
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil, ErrDenied
	}
	for name, want := range map[string]string{"issuer": b.issuer, "keyid": b.keyid, "version": "0.10.0", "alg": "ed25519"} {
		var got string
		if json.Unmarshal(fields[name], &got) != nil || got != want {
			return nil, ErrDenied
		}
	}
	pub, e := b.fresh(ctx)
	if e != nil {
		return nil, ErrDenied
	}
	signed, e := b.backend.Sign(ctx, append([]byte(nil), owned...))
	if e != nil || !active(ctx) {
		return nil, ErrDenied
	}
	proof = append([]byte(nil), signed...)
	if len(proof) != ed25519.SignatureSize || !ed25519.Verify(pub, owned, proof) {
		return nil, ErrDenied
	}
	if _, e = b.fresh(ctx); e != nil {
		return nil, ErrDenied
	}
	return proof, nil
}
