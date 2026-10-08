// SPDX-License-Identifier: LGPL-3.0-or-later
package guardbinding

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"regexp"

	"github.com/sage-x-project/sage-adk/core/toolhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var _ g.IssuancePolicy = (*Operation)(nil)
var _ g.IntentMeasurement = (*Operation)(nil)

func (o *Operation) enter(ctx context.Context) error {
	if o == nil || o.gate == nil || !active(ctx) || o.retiring.Load() {
		return ErrDenied
	}
	select {
	case <-ctx.Done():
		return ErrDenied
	case <-o.gate:
	}
	if !active(ctx) || o.retiring.Load() || o.closed {
		o.leave()
		return ErrDenied
	}
	return nil
}
func (o *Operation) leave() { o.gate <- struct{}{} }
func (o *Operation) recover(err *error) {
	if recover() != nil {
		o.retiring.Store(true)
		*err = ErrDenied
	}
}

// Bindings returns copies of the provisioned descriptors after capture, exact
// current file and same-instance checks. Wire digests cannot install a mapping.
func (o *Operation) Bindings(ctx context.Context, issuer, id string) (original string, policy, manifest []byte, err error) {
	if e := o.enter(ctx); e != nil {
		return "", nil, nil, e
	}
	defer o.leave()
	defer func() {
		if recover() != nil {
			o.retiring.Store(true)
			original = ""
			policy = nil
			manifest = nil
			err = ErrDenied
		}
	}()
	if issuer != o.issuer || id != o.request.ID() || o.check(ctx) != nil {
		return "", nil, nil, ErrDenied
	}
	return o.request.Digest(), append([]byte(nil), o.policy...), append([]byte(nil), o.manifest...), nil
}
func (o *Operation) permitted(issuer, tool string, args []byte) bool {
	b, e := canonicalObject(args)
	return e == nil && issuer == o.issuer && tool == o.rules.Tool && bytes.Equal(b, o.rules.Arguments)
}

// Authorize evaluates the exact locally provisioned tool and arguments. Success
// is a policy snapshot only; it is not approval, a reservation or execution.
func (o *Operation) Authorize(ctx context.Context, issuer, tool string, args []byte) (err error) {
	if e := o.enter(ctx); e != nil {
		return e
	}
	defer o.leave()
	defer o.recover(&err)
	if !o.permitted(issuer, tool, args) || o.check(ctx) != nil {
		return ErrDenied
	}
	return nil
}

// ApproveIntent checks every field of the canonical root or hop intent against fixed
// capture, identity, key, commitments and arguments. Core issuance separately
// checks fresh authority/time and owns one-use approval and durable fencing.
func (o *Operation) ApproveIntent(ctx context.Context, raw []byte) (err error) {
	if e := o.enter(ctx); e != nil {
		return e
	}
	defer o.leave()
	defer o.recover(&err)
	b, e := canonicalObject(raw)
	if e != nil {
		return ErrDenied
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil || len(m) != 17 {
		return ErrDenied
	}
	expected := map[string]string{"version": "0.10.0", "profile": "sage-execution-guard", "issuer": o.issuer, "recipient": o.rules.Recipient, "keyid": o.rules.KeyID, "alg": "ed25519", "request_id": o.request.ID(), "original_digest": o.request.Digest(), "policy_digest": o.policyDigest, "manifest_digest": o.manifestDigest, "tool": o.rules.Tool}
	for field, want := range expected {
		var got string
		if json.Unmarshal(m[field], &got) != nil || got != want {
			return ErrDenied
		}
	}
	var call, nonce string
	var created, expires int64
	if json.Unmarshal(m["call_id"], &call) != nil || !uuid.MatchString(call) || json.Unmarshal(m["nonce"], &nonce) != nil || len(nonce) != 22 {
		return ErrDenied
	}
	n, e := base64.RawURLEncoding.Strict().DecodeString(nonce)
	if e != nil || len(n) != 16 || !o.parentMatches(m["parent_call_id"], call) || json.Unmarshal(m["created"], &created) != nil || json.Unmarshal(m["expires"], &expires) != nil || created < 0 || expires > 9007199254740991 || expires <= created || expires-created > o.rules.Lifetime || !o.permitted(o.issuer, o.rules.Tool, m["arguments"]) || o.check(ctx) != nil {
		return ErrDenied
	}
	return nil
}

func (o *Operation) parentMatches(raw []byte, call string) bool {
	if o.parentID == "" {
		return bytes.Equal(raw, []byte("null"))
	}
	var parent string
	return json.Unmarshal(raw, &parent) == nil && parent == o.parentID && call != parent
}

// Check verifies the approved component name/digest and current captured input,
// exact artifact bytes and the same loaded evaluator/tool instance.
func (o *Operation) Check(ctx context.Context, manifest, tool string) (err error) {
	if e := o.enter(ctx); e != nil {
		return e
	}
	defer o.leave()
	defer o.recover(&err)
	if manifest != o.manifestDigest || tool != o.rules.Tool {
		return ErrDenied
	}
	return o.check(ctx)
}

type loaded struct{ operation *Operation }

func (l *loaded) Check(ctx context.Context, manifest, tool string) error {
	return l.operation.Check(ctx, manifest, tool)
}
func (l *loaded) Execute(ctx context.Context, args []byte) (output []byte, err error) {
	o := l.operation
	if e := o.enter(ctx); e != nil {
		return nil, e
	}
	defer o.leave()
	defer func() {
		if recover() != nil {
			o.retiring.Store(true)
			output = nil
			err = ErrDenied
		}
	}()
	if !o.permitted(o.issuer, o.rules.Tool, args) || o.check(ctx) != nil {
		return nil, ErrDenied
	}
	out, e := o.instance.Execute(ctx, append([]byte(nil), args...))
	if e != nil || !active(ctx) {
		return nil, ErrDenied
	}
	return append([]byte(nil), out...), nil
}

// Binding supplies a private same-instance wrapper for trusted native host
// configuration. Never expose this capability to a model/plugin or invoke it as
// an unsigned dispatch API. Only toolhost's admitted worker may call Execute.
func (o *Operation) Binding(ctx context.Context) (binding toolhost.Binding, err error) {
	if e := o.enter(ctx); e != nil {
		return binding, e
	}
	defer o.leave()
	defer func() {
		if recover() != nil {
			o.retiring.Store(true)
			binding = toolhost.Binding{}
			err = ErrDenied
		}
	}()
	if o.check(ctx) != nil {
		return binding, ErrDenied
	}
	return toolhost.Binding{Name: o.rules.Tool, ManifestDigest: o.manifestDigest, Tool: &loaded{o}}, nil
}

// Close immediately refuses new callbacks, waits for the bounded accepted effect
// and cleanup, then permanently closes artifact custody. Timeout retains ownership
// and refusal; retry with a fresh context. It cannot undo an already accepted effect.
func (o *Operation) Close(ctx context.Context) error {
	if o == nil {
		return nil
	}
	if o.gate == nil || !active(ctx) {
		return ErrDenied
	}
	o.retiring.Store(true)
	select {
	case <-ctx.Done():
		return ErrDenied
	case <-o.gate:
	}
	defer o.leave()
	if o.closed {
		return nil
	}
	if o.root.Close() != nil {
		return ErrDenied
	}
	o.closed = true
	return nil
}
