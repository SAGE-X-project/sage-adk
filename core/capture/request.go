// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import (
	"context"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// Request is an opaque binding to persisted original input. Its zero value is
// invalid. Keep it with the trusted input owner, never in model/tool arguments.
type Request struct {
	host       *Host
	id, digest string
}

// ID returns the host-generated request identifier, or empty for a nil request.
func (r *Request) ID() string {
	if r == nil {
		return ""
	}
	return r.id
}

// Digest returns the exact-byte commitment. A digest alone grants no authority.
func (r *Request) Digest() string {
	if r == nil {
		return ""
	}
	return r.digest
}

// Inputs reloads and verifies durable input before returning a defensive copy.
// Failure requires refusing processing rather than using a reconstructed input.
func (r *Request) Inputs(ctx context.Context) (inputs [][]byte, err error) {
	defer func() {
		if recover() != nil {
			inputs = nil
			err = ErrCapture
		}
	}()
	if r == nil || r.host == nil || absent(r.host.store) || !active(ctx) {
		return nil, ErrCapture
	}
	b, e := r.host.store.Load(ctx, r.id)
	if e != nil || !active(ctx) {
		return nil, ErrCapture
	}
	if _, e = g.OriginalCommitment(b); e != nil {
		return nil, ErrCapture
	}
	owned := clone(b)
	d, e := g.OriginalCommitment(owned)
	if e != nil || d != r.digest || !active(ctx) {
		return nil, ErrCapture
	}
	return owned, nil
}

// NewIntentIssuer binds retained input to all policy checks and signing custody.
// Services must be independently authoritative, protected and immutable during an
// operation. This method does not provide a permissive policy, registry or signer.
func (r *Request) NewIntentIssuer(ctx context.Context, s g.IssuerServices) (*g.IntentIssuer, error) {
	inputs, e := r.Inputs(ctx)
	if e != nil {
		return nil, e
	}
	if absent(s.Policy) || absent(s.Client.Policy) || absent(s.Signer) ||
		absent(s.Measurement) || absent(s.Client.IntentAuthority) ||
		absent(s.Client.ResultAuthority) || absent(s.Client.Clock) || absent(s.Client.Sender) {
		return nil, ErrCapture
	}
	root, e := g.NewRootCapture(inputs, r.id)
	if e != nil {
		return nil, ErrCapture
	}
	s.Client.Policy = &boundPolicy{request: r, delegate: s.Client.Policy}
	s.Policy = &boundIssuancePolicy{boundPolicy: boundPolicy{request: r, delegate: s.Policy}, delegate: s.Policy}
	s.Signer = &boundSigner{request: r, delegate: s.Signer}
	return g.NewIntentIssuer(root, s)
}
