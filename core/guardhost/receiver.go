//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardhost

import (
	"context"
	"net"

	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	p "github.com/sage-x-project/sage-adk/core/guardservices"
	"github.com/sage-x-project/sage-adk/core/toolhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	h "github.com/sage-x-project/sage/pkg/agent/hpke"
)

// ReceiverConfig assembles a separate tool host. It holds no original request:
// the operation is bound with guardbinding.OpenReceiver and verified through
// the provisioned (issuer, policy_digest) mapping. KEM is the local X25519
// private key registered for this identity; it is not held by custody.
type ReceiverConfig struct {
	Environment
	Identity
	KEM               []byte
	Issuer, IssuerKey string
	Approved
	Bounds     g.MCPHostBounds
	Connection g.MCPConnectionConfig
}

// Receiver owns the native host, the approved operation and its journals.
type Receiver struct {
	state      *state
	operation  *b.Operation
	endpoint   func(context.Context) (*h.CompletionEndpoint010, error)
	kem        []byte
	host       *toolhost.Host
	connection g.MCPConnectionConfig
}

// OpenReceiver records the operator approval, binds the approved operation and
// opens the native host. Any missing port or refused check fails closed.
func OpenReceiver(ctx context.Context, c ReceiverConfig) (receiver *Receiver, err error) {
	if !active(ctx) || !c.Environment.valid() || !c.Identity.valid() || len(c.KEM) != 32 || c.Issuer == "" || c.IssuerKey == "" || c.Connection.Role != g.MCPResponder {
		return nil, ErrDenied
	}
	s := &state{env: c.Environment}
	rcv := &Receiver{state: s, connection: c.Connection}
	defer func() {
		if recover() != nil {
			err = ErrDenied
		}
		if err != nil {
			_ = rcv.Close(context.Background())
			receiver = nil
		}
	}()
	if err = s.approve(c.Approved); err != nil {
		return nil, err
	}
	if rcv.operation, err = b.OpenReceiver(ctx, c.Operation); err != nil {
		return nil, ErrDenied
	}
	policy, err := g.NewReceiverPolicy(rcv.operation)
	if err != nil {
		return nil, ErrDenied
	}
	binding, err := rcv.operation.Binding(ctx)
	if err != nil {
		return nil, ErrDenied
	}
	intent, err := s.authority("intent", c.Issuer, c.IssuerKey)
	if err != nil {
		return nil, err
	}
	result, err := s.authority("result", c.DID, c.KeyID)
	if err != nil {
		return nil, err
	}
	signer, err := p.NewResultSigner(ctx, result, c.DID, c.KeyID, c.Result)
	if err != nil {
		return nil, ErrDenied
	}
	rcv.kem = append([]byte(nil), c.KEM...)
	if rcv.endpoint, err = s.transport(c.Identity, rcv.kem); err != nil {
		return nil, err
	}
	rcv.host, err = toolhost.Open(s.path("ledger"), c.Create, c.DID, toolhost.Services{IntentAuthority: intent, ResultAuthority: result, Policy: policy, Signer: signer, Clock: c.Clock}, []toolhost.Binding{binding}, c.Bounds)
	if err != nil {
		return nil, ErrDenied
	}
	return rcv, nil
}

// Serve accepts bounded native connections until ctx ends. Each connection
// serves admitted calls until the peer closes it.
func (r *Receiver) Serve(ctx context.Context, l net.Listener, workers int) error {
	if r == nil || r.host == nil {
		return ErrDenied
	}
	return r.host.Serve(ctx, l, workers, r.connection, &connection{endpoint: r.endpoint, handle: func(_ context.Context, m *g.MCPConnection) error {
		for {
			if err := m.ServeOne(); err != nil {
				return err
			}
		}
	}})
}

// Close retires the host, the operation, the local KEM copy and the journals. On a
// timeout it keeps ownership; call again with a fresh context.
func (r *Receiver) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if r.host != nil {
		if err := r.host.Close(ctx); err != nil {
			return err
		}
		r.host = nil
	}
	if r.operation != nil {
		if err := r.operation.Close(ctx); err != nil {
			return err
		}
		r.operation = nil
	}
	r.endpoint = nil
	for i := range r.kem {
		r.kem[i] = 0
	}
	if r.state != nil {
		return r.state.close()
	}
	return nil
}
