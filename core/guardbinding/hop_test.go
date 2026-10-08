// SPDX-License-Identifier: LGPL-3.0-or-later
package guardbinding

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sage-x-project/sage-adk/core/capture"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

const testParent = "00000000-0000-4000-8000-000000000009"

func TestHopConstructorRequiresRealRetainedParent(t *testing.T) {
	for _, request := range []*capture.HopRequest{nil, {}} {
		e := environment(t)
		for _, ctx := range []context.Context{context.Background(), nil} {
			if o, err := OpenHop(ctx, request, e.config); err == nil || o != nil {
				t.Fatal("missing actual admitted parent accepted")
			}
		}
		if e.factory.instance.effects.Load() != 0 {
			t.Fatal("invalid parent reached effect")
		}
	}
}

func TestHopLocalPolicyDoesNotInheritRootApproval(t *testing.T) {
	// A private synthetic capture exercises exact local policy fields only.
	// It supplies no real admission; the public OpenHop path is tested over
	// actual native loopback in core/toolhost.
	for _, mode := range []string{"allowed", "null-parent", "wrong-parent", "parent-type", "same-call", "issuer", "recipient", "keyid", "request_id", "original_digest", "policy_digest", "manifest_digest", "arguments", "tool", "profile", "version", "alg", "extra"} {
		t.Run(mode, func(t *testing.T) {
			e := environment(t)
			o, err := open(context.Background(), e.request, e.config, testParent, alice)
			must(t, err)
			t.Cleanup(func() { must(t, o.Close(context.Background())) })
			var fields map[string]any
			must(t, json.Unmarshal(unsigned(o, e.request), &fields))
			fields["parent_call_id"] = testParent
			switch mode {
			case "null-parent":
				fields["parent_call_id"] = nil
			case "wrong-parent":
				fields["parent_call_id"] = "00000000-0000-4000-8000-000000000008"
			case "parent-type":
				fields["parent_call_id"] = 9
			case "same-call":
				fields["call_id"] = testParent
			case "arguments":
				fields["arguments"] = map[string]int{"a": 2, "b": 4}
			case "extra":
				fields["extra"] = true
			default:
				if mode != "allowed" {
					fields[mode] = "unapproved"
				}
			}
			raw, err := g.Canonicalize(testJSON(fields))
			must(t, err)
			err = o.ApproveIntent(context.Background(), raw)
			if (err == nil) != (mode == "allowed") {
				t.Fatalf("approval %s: %v", mode, err)
			}
			if e.factory.instance.effects.Load() != 0 {
				t.Fatal("policy snapshot executed an effect")
			}
		})
	}
	e := environment(t)
	if o, err := open(context.Background(), e.request, e.config, testParent, bob); err == nil || o != nil {
		t.Fatal("policy issuer differed from authenticated local recipient")
	}
}
