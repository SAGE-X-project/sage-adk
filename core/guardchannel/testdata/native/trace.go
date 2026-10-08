// SPDX-License-Identifier: LGPL-3.0-or-later
package main

import (
	"context"
	"fmt"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	"github.com/sage-x-project/sage-adk/core/toolhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	"os"
)

// Nonsecret test diagnostics only; delegate every check without substitution.
type observedPolicy struct{ inner *b.Operation }

func (p observedPolicy) Bindings(ctx context.Context, issuer, id string) (string, []byte, []byte, error) {
	a, b, c, e := p.inner.Bindings(ctx, issuer, id)
	if e != nil {
		fmt.Fprintln(os.Stderr, "fixture policy bindings refused", ctx.Err())
	}
	return a, b, c, e
}
func (p observedPolicy) Authorize(ctx context.Context, issuer, tool string, args []byte) error {
	e := p.inner.Authorize(ctx, issuer, tool, args)
	if e != nil {
		fmt.Fprintln(os.Stderr, "fixture policy authorization refused", ctx.Err())
	}
	return e
}
func (p observedPolicy) ApproveIntent(ctx context.Context, raw []byte) error {
	return p.inner.ApproveIntent(ctx, raw)
}

type observedTool struct{ inner toolhost.LoadedTool }

func (t observedTool) Check(ctx context.Context, md, tool string) error {
	e := t.inner.Check(ctx, md, tool)
	if e != nil {
		fmt.Fprintln(os.Stderr, "fixture final measurement refused", ctx.Err())
	}
	return e
}
func (t observedTool) Execute(ctx context.Context, raw []byte) ([]byte, error) {
	out, e := t.inner.Execute(ctx, raw)
	if e != nil {
		fmt.Fprintln(os.Stderr, "fixture final effect refused", ctx.Err())
	}
	return out, e
}

var _ g.IssuancePolicy = observedPolicy{}
