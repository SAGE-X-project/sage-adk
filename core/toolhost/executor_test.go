// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost

import (
	"context"
	"strings"
	"testing"

	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

type countTool struct {
	checks, calls    int
	fail, panicCheck bool
}

func (t *countTool) Check(context.Context, string, string) error {
	t.checks++
	if t.panicCheck {
		panic("fixture check")
	}
	if t.fail {
		return ErrDenied
	}
	return nil
}
func (t *countTool) Execute(context.Context, []byte) ([]byte, error) {
	t.calls++
	return []byte(`{}`), nil
}
func TestCheckPinsRegistrationAndRejectsFailure(t *testing.T) {
	tool := &countTool{}
	digest := strings.Repeat("a", 64)
	e, err := newExecutor("did:sage:web:agent.example:bob", nil, nil, []Binding{{"sum", digest, tool}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err = e.Check(ctx, digest, "sum"); err != nil {
		t.Fatal(err)
	}
	if tool.checks != 1 {
		t.Fatal("measurement not called")
	}
	for _, test := range []struct{ digest, name string }{{strings.Repeat("b", 64), "sum"}, {digest, "unknown"}} {
		if err = e.Check(ctx, test.digest, test.name); err == nil {
			t.Fatal("unbound instance accepted")
		}
	}
	tool.fail = true
	if err = e.Check(ctx, digest, "sum"); err == nil {
		t.Fatal("failed measurement accepted")
	}
	tool.fail = false
	tool.panicCheck = true
	if err = e.Check(ctx, digest, "sum"); err == nil {
		t.Fatal("panicking measurement accepted")
	}
	if err = e.Check(nil, digest, "sum"); err == nil {
		t.Fatal("nil context")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err = e.Check(cancelled, digest, "sum"); err == nil {
		t.Fatal("cancelled check")
	}
}
func TestNoAdmittedInvocationCannotRun(t *testing.T) {
	tool := &countTool{}
	e, err := newExecutor("did:sage:web:agent.example:bob", nil, nil, []Binding{{"sum", strings.Repeat("a", 64), tool}})
	if err != nil {
		t.Fatal(err)
	}
	for _, invocation := range []*g.Invocation{nil, {}} {
		if output, err := e.Run(context.Background(), invocation); err == nil || output != nil {
			t.Fatal("unadmitted invocation executed")
		}
	}
	if _, err = e.Run(nil, &g.Invocation{}); err == nil {
		t.Fatal("nil context")
	}
	if tool.calls != 0 {
		t.Fatal("tool invoked without admission")
	}
}

func TestBindingsAreCopied(t *testing.T) {
	first, replacement := &countTool{}, &countTool{}
	digest := strings.Repeat("a", 64)
	bindings := []Binding{{"sum", digest, first}}
	e, err := newExecutor("did:sage:web:agent.example:bob", nil, nil, bindings)
	if err != nil {
		t.Fatal(err)
	}
	bindings[0].Tool = replacement
	bindings[0].ManifestDigest = strings.Repeat("b", 64)
	if e.Check(context.Background(), digest, "sum") != nil || first.checks != 1 || replacement.checks != 0 {
		t.Fatal("registration changed through caller slice")
	}
}
