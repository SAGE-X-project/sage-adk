// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost

import (
	"bytes"
	"context"
	"reflect"
	"regexp"

	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

var toolName = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)
var digestName = regexp.MustCompile(`^[0-9a-f]{64}$`)

type executor struct {
	recipient string
	authority g.Authority
	policy    g.IntentPolicy
	bindings  map[string]Binding
}

func absent(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
func active(ctx context.Context) bool { return ctx != nil && ctx.Err() == nil }
func newExecutor(recipient string, authority g.Authority, policy g.IntentPolicy, bindings []Binding) (*executor, error) {
	if len(bindings) == 0 || len(bindings) > 1024 {
		return nil, ErrDenied
	}
	e := &executor{recipient: recipient, authority: authority, policy: policy, bindings: make(map[string]Binding, len(bindings))}
	for _, b := range bindings {
		if !toolName.MatchString(b.Name) || b.Name == "sage_secure_call" || !digestName.MatchString(b.ManifestDigest) || absent(b.Tool) {
			return nil, ErrDenied
		}
		if _, exists := e.bindings[b.Name]; exists {
			return nil, ErrDenied
		}
		e.bindings[b.Name] = b
	}
	return e, nil
}
func (e *executor) Check(ctx context.Context, manifest, name string) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrDenied
		}
	}()
	if e == nil || !active(ctx) {
		return ErrDenied
	}
	b, ok := e.bindings[name]
	if !ok || b.ManifestDigest != manifest {
		return ErrDenied
	}
	if b.Tool.Check(ctx, manifest, name) != nil || !active(ctx) {
		return ErrDenied
	}
	return nil
}
func (e *executor) Run(ctx context.Context, i *g.Invocation) (output []byte, err error) {
	defer func() {
		if recover() != nil {
			output = nil
			err = ErrDenied
		}
	}()
	if e == nil || !active(ctx) || i == nil || i.ParentAdmission() == nil {
		return nil, ErrDenied
	}
	incoming := i.CanonicalIntent()
	admission := i.ParentAdmission()
	if admission.Authorized(ctx, incoming) != nil {
		return nil, ErrDenied
	}
	verified, err := g.VerifyIntent(ctx, incoming, e.recipient, e.authority, e.policy)
	if err != nil || !bytes.Equal(verified.Canonical(), incoming) || verified.Digest() != i.IntentDigest() {
		return nil, ErrDenied
	}
	b, ok := e.bindings[i.Tool()]
	if !ok || b.ManifestDigest != i.ManifestDigest() {
		return nil, ErrDenied
	}
	if e.Check(ctx, i.ManifestDigest(), i.Tool()) != nil || admission.Authorized(ctx, incoming) != nil || !active(ctx) {
		return nil, ErrDenied
	}
	// The same registered instance owns measurement and execution. Pass only a
	// byte snapshot; never the invocation, completion token, signer or host owner.
	raw, err := b.Tool.Execute(ctx, i.Arguments())
	if err != nil || !active(ctx) {
		return nil, ErrDenied
	}
	canonical, err := g.Canonicalize(raw)
	if err != nil || len(canonical) == 0 || canonical[0] != '{' || !active(ctx) {
		return nil, ErrDenied
	}
	return canonical, nil
}
