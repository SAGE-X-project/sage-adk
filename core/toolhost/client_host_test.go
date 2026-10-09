// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/toolhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// An initiator-only host refuses to serve and closes without a ledger.
func TestOpenClientIsInitiatorOnly(t *testing.T) {
	b := g.MCPClientHostBounds{Clients: 1, Owners: 2, Client: time.Second, Tick: time.Millisecond}
	if h, err := toolhost.OpenClient(nil, b); err == nil || h != nil {
		t.Fatal("client host opened without a clock")
	}
	if h, err := toolhost.OpenClient(&fixtureClock{}, g.MCPClientHostBounds{}); err == nil || h != nil {
		t.Fatal("client host opened with invalid bounds")
	}
	h, err := toolhost.OpenClient(&fixtureClock{}, b)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Serve(ctx, listener, 1, g.MCPConnectionConfig{Role: g.MCPResponder}, nil); err == nil {
		t.Fatal("initiator-only host served")
	}
	if err := h.Close(ctx); err != nil {
		t.Fatal(err)
	}
}
