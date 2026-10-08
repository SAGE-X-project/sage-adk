//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardhost

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/sage-x-project/sage-adk/core/capture"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	p "github.com/sage-x-project/sage-adk/core/guardservices"
	"github.com/sage-x-project/sage-adk/core/toolhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	h "github.com/sage-x-project/sage/pkg/agent/hpke"
)

// CallerConfig assembles a root issuer host. Intent is the custody for this
// identity's intent signatures. The core MCP host also requires this host's
// own receiver services; they use this identity and the same approved
// operation, whose rules name only Recipient, so the host issues no call to
// itself.
type CallerConfig struct {
	Environment
	Identity
	Intent                  h.Ed25519Custody010
	Recipient, RecipientKey string
	Approved
	Bounds     g.MCPHostBounds
	Connection g.MCPConnectionConfig
}

// Caller owns the capture store, journals, endpoint factory and signers. A
// new replay journal stays in its 360-second quarantine after OpenCaller;
// calls made earlier fail closed.
type Caller struct {
	config     CallerConfig
	state      *state
	store      *capture.FileStore
	capturer   *capture.Host
	endpoint   func(context.Context) (*h.CompletionEndpoint010, error)
	intent     *g.RegistryAuthority
	peer       *g.RegistryAuthority
	own        *g.RegistryAuthority
	signer     *p.IntentSigner
	ownResults *p.ResultSigner
}

// OpenCaller records the operator approval and opens persistent caller state.
func OpenCaller(ctx context.Context, c CallerConfig) (caller *Caller, err error) {
	if !active(ctx) || !c.Environment.valid() || !c.Identity.valid() || absent(c.Intent) || c.Recipient == "" || c.RecipientKey == "" || c.Connection.Role != g.MCPInitiator || c.Connection.Recipient != c.Recipient || c.Connection.RecipientKey != c.RecipientKey {
		return nil, ErrDenied
	}
	s := &state{env: c.Environment}
	cl := &Caller{config: c, state: s}
	defer func() {
		if recover() != nil {
			err = ErrDenied
		}
		if err != nil {
			_ = cl.Close()
			caller = nil
		}
	}()
	if err = s.approve(c.Approved); err != nil {
		return nil, err
	}
	if cl.store, err = capture.OpenFileStore(s.path("originals")); err != nil {
		return nil, ErrDenied
	}
	if cl.capturer, err = capture.NewHost(cl.store); err != nil {
		return nil, ErrDenied
	}
	if cl.intent, err = s.authority("intent", c.DID, c.KeyID); err != nil {
		return nil, err
	}
	if cl.peer, err = s.authority("result", c.Recipient, c.RecipientKey); err != nil {
		return nil, err
	}
	if cl.own, err = s.authority("own-result", c.DID, c.KeyID); err != nil {
		return nil, err
	}
	if cl.signer, err = p.NewIntentSigner(ctx, cl.intent, c.DID, c.KeyID, c.Intent); err != nil {
		return nil, ErrDenied
	}
	if cl.ownResults, err = p.NewResultSigner(ctx, cl.own, c.DID, c.KeyID, c.Result); err != nil {
		return nil, ErrDenied
	}
	if cl.endpoint, err = s.transport(c.Identity, nil); err != nil {
		return nil, err
	}
	return cl, nil
}

const pollInterval = 1100 * time.Millisecond

type noSender struct{}

func (noSender) Commit(context.Context, string, []byte) error { return ErrDenied }

// Call captures the original inputs before any expansion, binds the approved
// operation to them, issues one intent under the approved policy, sends it over
// conn and returns the verified terminal output. conn is owned by the call.
func (c *Caller) Call(ctx context.Context, inputs [][]byte, proposal g.IntentProposal, conn net.Conn) (output []byte, err error) {
	defer func() {
		if recover() != nil {
			output, err = nil, ErrDenied
		}
	}()
	if c == nil || c.endpoint == nil || !active(ctx) || conn == nil {
		if conn != nil {
			_ = conn.Close()
		}
		return nil, ErrDenied
	}
	request, err := c.capturer.Capture(ctx, inputs)
	if err != nil {
		_ = conn.Close()
		return nil, ErrDenied
	}
	operation, err := b.Open(ctx, request, c.config.Operation)
	if err != nil {
		_ = conn.Close()
		return nil, ErrDenied
	}
	defer func() { err = errors.Join(err, closeWith(operation.Close)) }()
	intent, journal, err := c.issue(ctx, request, operation, proposal)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	binding, err := operation.Binding(ctx)
	if err != nil {
		_ = conn.Close()
		return nil, ErrDenied
	}
	host, err := toolhost.Open(c.state.path("client-"+request.ID()), true, c.config.DID, toolhost.Services{IntentAuthority: c.intent, ResultAuthority: c.own, Policy: operation, Signer: c.ownResults, Clock: c.config.Clock}, []toolhost.Binding{binding}, c.config.Bounds)
	if err != nil {
		_ = conn.Close()
		return nil, ErrDenied
	}
	defer func() { err = errors.Join(err, closeWith(host.Close)) }()
	var delivery *g.ClientDelivery
	err = host.Connect(ctx, conn, c.config.Connection, &connection{endpoint: c.endpoint, handle: func(callCtx context.Context, m *g.MCPConnection) error {
		if e := request.OpenMCPClient(callCtx, m, journal, intent, g.MCPClientServices{IntentAuthority: c.intent, ResultAuthority: c.peer, Policy: operation, Clock: c.config.Clock}); e != nil {
			return e
		}
		for callCtx.Err() == nil {
			d, e := m.Exchange()
			if e != nil {
				return e
			}
			if d.Status() == "completed" || d.Status() == "unknown" {
				delivery = d
				return nil
			}
			// The core Client requires at least one second of UTC and monotonic
			// time between handoffs.
			select {
			case <-callCtx.Done():
			case <-time.After(pollInterval):
			}
		}
		return ErrDenied
	}})
	if err != nil || delivery == nil || delivery.Status() != "completed" || !delivery.FirstTerminal() {
		return nil, ErrDenied
	}
	return delivery.Output(), nil
}

func (c *Caller) issue(ctx context.Context, request *capture.Request, operation *b.Operation, proposal g.IntentProposal) ([]byte, string, error) {
	issuer, err := request.NewIntentIssuer(ctx, g.IssuerServices{Client: g.ClientServices{IntentAuthority: c.intent, ResultAuthority: c.peer, Policy: operation, Clock: c.config.Clock, Sender: noSender{}, ExpectedIssuer: c.config.DID, ExpectedRecipient: c.config.Recipient}, Policy: operation, Signer: c.signer, Measurement: operation, KeyID: c.config.KeyID})
	if err != nil {
		return nil, "", ErrDenied
	}
	defer func() { _ = issuer.Retire() }()
	approved, err := issuer.Authorize(ctx, proposal)
	if err != nil {
		return nil, "", ErrDenied
	}
	journal := c.state.path("journal-" + request.ID())
	durable, err := issuer.Issue(ctx, journal, approved)
	if err != nil {
		return nil, "", ErrDenied
	}
	intent, err := durable.JournaledIntent()
	if closeErr := durable.Close(); err != nil || closeErr != nil {
		return nil, "", ErrDenied
	}
	return intent, journal, nil
}

func closeWith(close func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return close(ctx)
}

// Close retires the capture store and journals.
func (c *Caller) Close() error {
	if c == nil {
		return nil
	}
	var errs []error
	c.endpoint = nil
	if c.store != nil {
		errs = append(errs, c.store.Close())
		c.store = nil
	}
	if c.state != nil {
		errs = append(errs, c.state.close())
	}
	return errors.Join(errs...)
}
