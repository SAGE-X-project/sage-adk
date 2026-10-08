// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import (
	"context"
	"testing"

	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

func TestHopRequiresNativeAdmissionBeforePersistence(t *testing.T) {
	store := &boundaryStore{}
	host, _ := NewHost(store)
	for _, s := range []HopServices{{}, {Recipient: "did:sage:web:agent.example:b", Invocation: &g.Invocation{}}} {
		if r, e := host.CaptureHop(context.Background(), s); e == nil || r != nil {
			t.Fatal("unadmitted input stored")
		}
		if r, e := host.RestoreHop(context.Background(), "00000000-0000-4000-8000-000000000001", "", s); e == nil || r != nil {
			t.Fatal("unadmitted checkpoint restored")
		}
	}
	if store.inputs != nil {
		t.Fatal("unadmitted parent reached storage")
	}
}
func TestAbsentOrRetiredHopCannotUseProviders(t *testing.T) {
	var nilHop *HopRequest
	for _, r := range []*HopRequest{nilHop, {}, {request: &Request{}, invocation: &g.Invocation{}}} {
		if _, e := r.Inputs(context.Background()); e == nil {
			t.Fatal("invalid hop input")
		}
		if _, e := r.NewIntentIssuer(context.Background(), g.IssuerServices{}); e == nil {
			t.Fatal("invalid issuer")
		}
		if e := r.OpenMCPClient(context.Background(), nil, "", nil, g.MCPClientServices{}); e == nil {
			t.Fatal("invalid handoff")
		}
	}
	if nilHop.ID() != "" || nilHop.Digest() != "" || nilHop.ParentCallID() != "" {
		t.Fatal("nil metadata")
	}
	r := &HopRequest{request: &Request{id: "fixture", digest: "fixture"}}
	if e := r.check(context.Background()); e == nil || !r.retired.Load() {
		t.Fatal("failed check did not retire")
	}
	if e := r.check(context.Background()); e == nil {
		t.Fatal("retired hop reused")
	}
	canceled, stop := context.WithCancel(context.Background())
	stop()
	if e := (&HopRequest{}).check(canceled); e == nil {
		t.Fatal("canceled hop accepted")
	}
}
