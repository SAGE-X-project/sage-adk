// SPDX-License-Identifier: LGPL-3.0-or-later
package capture_test

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage-adk/core/capture"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

func TestMCPTransferRequiresOriginalAndCompleteServices(t *testing.T) {
	ctx := context.Background()
	store := &memoryStore{}
	host, _ := capture.NewHost(store)
	r, err := host.Capture(ctx, [][]byte{[]byte("inert")})
	if err != nil {
		t.Fatal(err)
	}
	policy := &fixturePolicy{request: r, store: store}
	services := g.MCPClientServices{IntentAuthority: &g.RegistryAuthority{}, ResultAuthority: &g.RegistryAuthority{}, Policy: policy, Clock: fixtureClock{}}
	for _, mode := range []string{"connection", "intent-authority", "result-authority", "policy", "clock", "typed-nil-policy", "typed-nil-clock", "zero-connection", "changed", "unavailable", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			s := services
			c := &g.MCPConnection{}
			callCtx := ctx
			switch mode {
			case "connection":
				c = nil
			case "intent-authority":
				s.IntentAuthority = nil
			case "result-authority":
				s.ResultAuthority = nil
			case "policy":
				s.Policy = nil
			case "clock":
				s.Clock = nil
			case "typed-nil-policy":
				var p *fixturePolicy
				s.Policy = p
			case "typed-nil-clock":
				var clock *fixtureClock
				s.Clock = clock
			case "changed":
				store.corrupt = true
				defer func() { store.corrupt = false }()
			case "unavailable":
				store.failLoad = true
				defer func() { store.failLoad = false }()
			case "cancelled":
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel()
				callCtx = cancelCtx
			}
			if e := r.OpenMCPClient(callCtx, c, "missing-journal", nil, s); e == nil {
				t.Fatal("incomplete native transfer accepted")
			}
		})
	}
}
