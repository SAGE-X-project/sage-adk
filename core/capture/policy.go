// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import (
	"context"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

type retainedInput interface {
	ID() string
	Digest() string
	Inputs(context.Context) ([][]byte, error)
}

type boundPolicy struct {
	request  retainedInput
	delegate g.IntentPolicy
}

func (p *boundPolicy) Bindings(ctx context.Context, issuer, id string) (string, []byte, []byte, error) {
	if id != p.request.ID() {
		return "", nil, nil, ErrCapture
	}
	if _, e := p.request.Inputs(ctx); e != nil {
		return "", nil, nil, e
	}
	d, policy, manifest, e := p.delegate.Bindings(ctx, issuer, id)
	if e != nil {
		return "", nil, nil, e
	}
	if _, e = p.request.Inputs(ctx); e != nil || d != p.request.Digest() {
		return "", nil, nil, ErrCapture
	}
	return d, policy, manifest, nil
}
func (p *boundPolicy) Authorize(ctx context.Context, issuer, tool string, args []byte) error {
	if _, e := p.request.Inputs(ctx); e != nil {
		return e
	}
	if e := p.delegate.Authorize(ctx, issuer, tool, args); e != nil {
		return e
	}
	_, e := p.request.Inputs(ctx)
	return e
}

type boundIssuancePolicy struct {
	boundPolicy
	delegate g.IssuancePolicy
}

func (p *boundIssuancePolicy) ApproveIntent(ctx context.Context, raw []byte) error {
	if _, e := p.request.Inputs(ctx); e != nil {
		return e
	}
	if e := p.delegate.ApproveIntent(ctx, raw); e != nil {
		return e
	}
	_, e := p.request.Inputs(ctx)
	return e
}

type boundSigner struct {
	request  retainedInput
	delegate g.IntentSigner
}

func (s *boundSigner) Sign(ctx context.Context, id string, raw []byte) ([]byte, error) {
	if _, e := s.request.Inputs(ctx); e != nil {
		return nil, e
	}
	return s.delegate.Sign(ctx, id, raw)
}
