//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later

// Package guardhost assembles the protected native ADK host for one approved
// exact operation from the opt-in Guard libraries: original capture, operator
// approval, the approved operation binding, Registry-bound signing through
// external custody, the native MCP owner and captured-Client result delivery.
//
// The authoritative Registry Source, the loaded-component measurement inside
// the operation factory, signing custody and the protected state directory are
// required ports. This package supplies no default for any of them and no
// fixture. Supplying a port does not make it authoritative or isolated; that
// remains a deployment property. Only private length-framed MCP over a caller
// owned connection is assembled; HTTP, A2A, stdio and ordinary ADK Agent or
// Tool Registry routes are outside this host.
package guardhost

import (
	"context"
	"crypto/ed25519"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"

	"github.com/sage-x-project/sage-adk/core/guardapproval"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	adkerrors "github.com/sage-x-project/sage-adk/pkg/errors"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	h "github.com/sage-x-project/sage/pkg/agent/hpke"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

// ErrDenied refuses missing, inconsistent or unavailable host assembly.
var ErrDenied = &adkerrors.Error{Category: adkerrors.CategorySecurity, Code: "GUARD_HOST_DENIED", Message: "guarded host assembly denied"}

// Clock is the one shared clock of the host: Registry, replay, native host and
// Client services must observe the same monotonic origin.
// guardservices.SystemClock satisfies it.
type Clock interface {
	r.Clock
	g.ClientClock
}

// Environment is protected host administration. Dir must already exist with
// mode 0700 and be owned by the host account. Create is true only for a new
// deployment; recovery never recreates missing state.
type Environment struct {
	Dir      string
	Create   bool
	Registry r.Config
	Source   r.Source
	Clock    Clock
}

// Identity is this host's DID, its signing key ID and the custody that signs
// transport envelopes and tool results with that key.
type Identity struct {
	DID, KeyID string
	Transport  h.Ed25519Custody010
	Result     h.Ed25519Custody010
}

// Approved is one operator-approved exact operation. The approval must cover
// exactly Operation.Policy and Operation.Manifest and be signed by a pinned key.
type Approved struct {
	Operation b.Config
	Approval  []byte
	Approvers []ed25519.PublicKey
}

func absent(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	}
	return false
}

func active(ctx context.Context) bool { return ctx != nil && ctx.Err() == nil }

func (e Environment) valid() bool {
	if !filepath.IsAbs(e.Dir) || filepath.Clean(e.Dir) != e.Dir || absent(e.Source) || absent(e.Clock) {
		return false
	}
	info, err := os.Lstat(e.Dir)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && st.Uid == uint32(os.Getuid())
}

func (i Identity) valid() bool {
	return i.DID != "" && i.KeyID != "" && !absent(i.Transport) && !absent(i.Result)
}

// state owns the journals opened for one host and closes them together.
type state struct {
	env      Environment
	journals []*r.Journal
	replays  []*h.ReplayJournal010
}

func (s *state) path(name string) string { return filepath.Join(s.env.Dir, name) }

func (s *state) gate(name string) (*r.Gate, error) {
	j, err := r.OpenJournal(s.path("registry-"+name), s.env.Create)
	if err != nil {
		return nil, ErrDenied
	}
	s.journals = append(s.journals, j)
	gate, err := r.NewGate(s.env.Registry, s.env.Source, s.env.Clock, j)
	if err != nil {
		return nil, ErrDenied
	}
	return gate, nil
}

func (s *state) authority(name, did, kid string) (*g.RegistryAuthority, error) {
	gate, err := s.gate(name)
	if err != nil {
		return nil, err
	}
	a, err := g.NewRegistryAuthority(gate, did, kid)
	if err != nil {
		return nil, ErrDenied
	}
	return a, nil
}

// transport opens the shared transport gate and durable replay journal and
// returns a factory for per-connection endpoints whose signing and KEM keys
// both stay in custody. The core owns and closes each endpoint when its
// connection ends; every endpoint shares the same replay store. kem is the
// responder's X25519 custody and nil for an initiator.
func (s *state) transport(id Identity, kem h.X25519Custody010) (func(context.Context) (*h.CompletionEndpoint010, error), error) {
	gate, err := s.gate("transport")
	if err != nil {
		return nil, err
	}
	replay, err := h.OpenReplayJournal010(s.path("replay"), s.env.Create, s.env.Clock)
	if err != nil {
		return nil, ErrDenied
	}
	s.replays = append(s.replays, replay)
	return func(ctx context.Context) (*h.CompletionEndpoint010, error) {
		e, err := h.NewProtectedCompletionEndpoint010(ctx, id.DID, id.KeyID, id.Transport, kem, gate, s.env.Clock, replay)
		if err != nil {
			return nil, ErrDenied
		}
		return e, nil
	}, nil
}

// approve records the operator approval in the host's durable ledger.
func (s *state) approve(a Approved) error {
	ledger, err := guardapproval.OpenLedger(s.path("approvals"), s.env.Create)
	if err != nil {
		return ErrDenied
	}
	if _, err = ledger.Accept(a.Approval, a.Operation.Policy, a.Operation.Manifest, a.Approvers); err != nil {
		return ErrDenied
	}
	return nil
}

func (s *state) close() error {
	var errs []error
	for _, j := range s.replays {
		errs = append(errs, j.Close())
	}
	for _, j := range s.journals {
		errs = append(errs, j.Close())
	}
	s.replays, s.journals = nil, nil
	return errors.Join(errs...)
}

// connection is the trusted handler for native connections. Endpoint returns
// a fresh endpoint for each connection.
type connection struct {
	endpoint func(context.Context) (*h.CompletionEndpoint010, error)
	handle   func(context.Context, *g.MCPConnection) error
}

func (c *connection) Endpoint(ctx context.Context) (*h.CompletionEndpoint010, error) {
	return c.endpoint(ctx)
}
func (*connection) Prepare(context.Context) error { return nil }
func (c *connection) Handle(ctx context.Context, m *g.MCPConnection) error {
	return c.handle(ctx, m)
}
