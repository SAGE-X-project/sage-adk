// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost

import (
	"context"
	"net"

	adkerrors "github.com/sage-x-project/sage-adk/pkg/errors"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

// ErrDenied refuses unavailable, inconsistent or unadmitted tool execution.
// It exposes neither original input nor provider error details.
var ErrDenied = &adkerrors.Error{Category: adkerrors.CategorySecurity, Code: "GUARDED_TOOL_DENIED", Message: "guarded tool operation denied"}

// Services are protected native host providers, never model/plugin capabilities.
// Registry sources, policy, clocks and signing custody must be independently
// authoritative, bounded, concurrency-safe and immutable until Close succeeds.
// Clocks share one monotonic origin as required by the core host.
type Services struct {
	IntentAuthority, ResultAuthority *g.RegistryAuthority
	Policy                           g.IntentPolicy
	Signer                           g.ResultSigner
	Clock                            r.Clock
}

// Host owns the native MCP lifecycle and a fixed set of pinned tool instances.
// It exports no direct executor, unsigned dispatch, tool getter or replacement.
// Supported carriage is private length-framed MCP, not HTTP, A2A or stdio.
type Host struct{ native *g.MCPHost }

// Open validates protected configuration before opening an exclusive native
// ledger. create is only for a new scope; recovery never recreates missing history.
func Open(path string, create bool, recipient string, s Services, bindings []Binding, b g.MCPHostBounds) (*Host, error) {
	executor, err := newExecutor(recipient, s.IntentAuthority, s.Policy, bindings)
	if err != nil {
		return nil, err
	}
	if s.IntentAuthority == nil || s.ResultAuthority == nil || absent(s.Policy) || absent(s.Signer) || absent(s.Clock) {
		return nil, ErrDenied
	}
	native, err := g.OpenMCPHost(path, create, recipient, g.MCPHostServices{IntentAuthority: s.IntentAuthority, ResultAuthority: s.ResultAuthority, Policy: s.Policy, Executor: executor, Signer: s.Signer, Clock: s.Clock}, b)
	if err != nil {
		return nil, err
	}
	return &Host{native: native}, nil
}

// Connect hands an owned connection to the core's authenticated native lifecycle.
// The handler is trusted host code; never give its connection to model/plugins.
func (h *Host) Connect(ctx context.Context, conn net.Conn, c g.MCPConnectionConfig, handler g.MCPConnectionHandler) error {
	if h == nil || h.native == nil {
		return ErrDenied
	}
	return h.native.Connect(ctx, conn, c, handler)
}

// Serve accepts bounded native connections using the core's protected admission.
// The host supplies actual endpoint/key ownership through the trusted handler.
func (h *Host) Serve(ctx context.Context, listener net.Listener, workers int, c g.MCPConnectionConfig, handler g.MCPConnectionHandler) error {
	if h == nil || h.native == nil {
		return ErrDenied
	}
	return h.native.Serve(ctx, listener, workers, c, handler)
}

// Close retires admission and waits for native workers/providers/cleanup. Timeout
// retains ownership and storage locks; call Close again with a fresh context.
// Do not replace configuration or reopen its ledger before Close succeeds.
func (h *Host) Close(ctx context.Context) error {
	if h == nil || h.native == nil {
		return ErrDenied
	}
	return h.native.Close(ctx)
}
