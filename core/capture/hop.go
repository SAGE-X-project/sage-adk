// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import (
	"bytes"
	"context"
	"encoding/json"
	"sync/atomic"

	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// HopServices is protected upstream verification configuration. Invocation must
// come from the core's actually admitted, currently running native MCP worker.
// Recipient is this host's independently configured DID. None is a model input.
// Providers must be bounded, concurrent-safe and non-reentrant; keep them alive
// until all accepted downstream work has finished. No authority is inherited.
type HopServices struct {
	Recipient  string
	Invocation *g.Invocation
	Authority  g.Authority
	Policy     g.IntentPolicy
}

// HopRequest retains one fresh local capture of the exact authenticated inbound
// envelope and its live parent admission. It exports no root Request or raw
// admission handle. Zero values are invalid. Parent completion permanently
// prevents new issuance or downstream handoff through this capability.
type HopRequest struct {
	request                         *Request
	incoming                        []byte
	parentID, upstreamID, recipient string
	invocation                      *g.Invocation
	authority                       g.Authority
	policy                          g.IntentPolicy
	retired                         atomic.Bool
}

func (r *HopRequest) ID() string {
	if r == nil {
		return ""
	}
	return r.request.ID()
}
func (r *HopRequest) Digest() string {
	if r == nil {
		return ""
	}
	return r.request.Digest()
}

// ParentCallID is causal metadata only; it grants no authority.
func (r *HopRequest) ParentCallID() string {
	if r == nil {
		return ""
	}
	return r.parentID
}

func hop(s HopServices) (*HopRequest, error) {
	if s.Invocation == nil || absent(s.Authority) || absent(s.Policy) || s.Invocation.ParentAdmission() == nil {
		return nil, ErrCapture
	}
	raw := s.Invocation.CanonicalIntent()
	if len(raw) == 0 || len(raw) > g.MaxBytes {
		return nil, ErrCapture
	}
	var envelope struct {
		Intent struct {
			CallID    string `json:"call_id"`
			RequestID string `json:"request_id"`
		} `json:"intent"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return nil, ErrCapture
	}
	return &HopRequest{incoming: raw, parentID: envelope.Intent.CallID, upstreamID: envelope.Intent.RequestID,
		recipient: s.Recipient, invocation: s.Invocation, authority: s.Authority, policy: s.Policy}, nil
}
func (r *HopRequest) check(ctx context.Context) (err error) {
	if r == nil || !active(ctx) || r.retired.Load() {
		return ErrCapture
	}
	defer func() {
		if recover() != nil {
			err = ErrCapture
		}
		if err != nil {
			r.retired.Store(true)
		}
	}()
	if r.invocation == nil || absent(r.authority) || absent(r.policy) || r.invocation.ParentAdmission() == nil {
		return ErrCapture
	}
	admission := r.invocation.ParentAdmission()
	if admission.Authorized(ctx, append([]byte(nil), r.incoming...)) != nil {
		return ErrCapture
	}
	v, e := g.VerifyIntent(ctx, append([]byte(nil), r.incoming...), r.recipient, r.authority, r.policy)
	if e != nil || !bytes.Equal(v.Canonical(), r.incoming) || v.Digest() != r.invocation.IntentDigest() ||
		!bytes.Equal(r.invocation.CanonicalIntent(), r.incoming) || admission.Authorized(ctx, append([]byte(nil), r.incoming...)) != nil || !active(ctx) {
		return ErrCapture
	}
	return nil
}

// CaptureHop verifies a real admitted parent before storing its exact envelope
// as a new local original. It checks again after persistence. Failed persistence
// or parent loss grants no capability; do not automatically retry with a new ID.
func (h *Host) CaptureHop(ctx context.Context, s HopServices) (request *HopRequest, err error) {
	defer func() {
		if recover() != nil {
			request = nil
			err = ErrCapture
		}
	}()
	r, e := hop(s)
	if e != nil || r.check(ctx) != nil {
		return nil, ErrCapture
	}
	r.request, e = h.Capture(ctx, [][]byte{r.incoming})
	if e != nil || r.request.ID() == r.upstreamID {
		return nil, ErrCapture
	}
	if _, e = r.Inputs(ctx); e != nil {
		return nil, ErrCapture
	}
	return r, nil
}

// RestoreHop binds a protected checkpoint to the exact same currently admitted
// parent. A finished/UNKNOWN parent cannot be resurrected by restoring storage.
// ID/digest are trusted checkpoint values, never selected from wire/model input.
func (h *Host) RestoreHop(ctx context.Context, id, digest string, s HopServices) (request *HopRequest, err error) {
	defer func() {
		if recover() != nil {
			request = nil
			err = ErrCapture
		}
	}()
	r, e := hop(s)
	if e != nil || r.check(ctx) != nil || id == r.upstreamID {
		return nil, ErrCapture
	}
	r.request, e = h.Restore(ctx, id, digest)
	if e != nil {
		return nil, ErrCapture
	}
	if _, e = r.Inputs(ctx); e != nil {
		return nil, ErrCapture
	}
	return r, nil
}

// Inputs verifies upstream authority/current native admission before and after
// exact durable capture readback. Any observed inconsistency retires permanently.
func (r *HopRequest) Inputs(ctx context.Context) (inputs [][]byte, err error) {
	if r == nil || r.request == nil || r.check(ctx) != nil {
		return nil, ErrCapture
	}
	defer func() {
		if recover() != nil {
			inputs = nil
			err = ErrCapture
		}
		if err != nil {
			r.retired.Store(true)
		}
	}()
	inputs, err = r.request.Inputs(ctx)
	if err != nil || len(inputs) != 1 || !bytes.Equal(inputs[0], r.incoming) || r.ID() == r.upstreamID || r.check(ctx) != nil {
		return nil, ErrCapture
	}
	return inputs, nil
}

// NewIntentIssuer applies this host's independently approved downstream policy,
// key custody and actual loaded-code measurement. It wraps capture checks around
// policy and signing and delegates one-use approval/durable fencing to the core.
func (r *HopRequest) NewIntentIssuer(ctx context.Context, s g.IssuerServices) (*g.IntentIssuer, error) {
	inputs, e := r.Inputs(ctx)
	if e != nil || s.Client.ExpectedIssuer != r.recipient {
		return nil, ErrCapture
	}
	if absent(s.Policy) || absent(s.Client.Policy) || absent(s.Signer) || absent(s.Measurement) || absent(s.Client.IntentAuthority) || absent(s.Client.ResultAuthority) || absent(s.Client.Clock) || absent(s.Client.Sender) {
		return nil, ErrCapture
	}
	root, e := g.NewRootCapture(inputs, r.ID())
	if e != nil {
		return nil, ErrCapture
	}
	s.Client.Policy = &boundPolicy{request: r, delegate: s.Client.Policy}
	s.Policy = &boundIssuancePolicy{boundPolicy: boundPolicy{request: r, delegate: s.Policy}, delegate: s.Policy}
	s.Signer = &boundSigner{request: r, delegate: s.Signer}
	return g.NewHopIntentIssuer(root, s, append([]byte(nil), r.incoming...), g.HopServices{Authority: r.authority, Policy: r.policy, Parent: r.invocation.ParentAdmission()})
}

// OpenMCPClient transfers an already issued, successfully closed Client journal
// to this authenticated native connection. It never recreates missing history.
// Preserve path, issuance fence and exact signed bytes in protected custody.
func (r *HopRequest) OpenMCPClient(ctx context.Context, c *g.MCPConnection, path string, intent []byte, s g.MCPClientServices) error {
	if _, e := r.Inputs(ctx); e != nil {
		return e
	}
	if c == nil || s.IntentAuthority == nil || s.ResultAuthority == nil || absent(s.Policy) || absent(s.Clock) {
		return ErrCapture
	}
	s.Policy = &boundPolicy{request: r, delegate: s.Policy}
	return c.OpenHopClient(path, false, append([]byte(nil), intent...), s, g.MCPHopServices{Parent: r.invocation, Authority: r.authority, Policy: r.policy})
}
