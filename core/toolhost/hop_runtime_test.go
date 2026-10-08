// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost_test

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/capture"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	p "github.com/sage-x-project/sage-adk/core/guardservices"
	"github.com/sage-x-project/sage-adk/core/toolhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	h "github.com/sage-x-project/sage/pkg/agent/hpke"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

// Fixed safe A->B->A loopback: A is also the separately authorized final receiver.
// Ephemeral keys and fixture Registry/measurement are not deployment evidence.
type hopSource struct{ env *fixtureEnvironment }

func (s hopSource) Read(ctx context.Context, did string) (r.Snapshot, error) {
	v, e := fixtureSource(s).Read(ctx, did)
	if e != nil {
		return v, e
	}
	if did == fixtureAlice {
		kem, e := ecdh.X25519().NewPrivateKey(s.env.kem)
		if e != nil {
			return v, e
		}
		v.Keys = append([]r.Key{{Name: "kem-1", Alg: "x25519", Material: hex.EncodeToString(kem.PublicKey().Bytes()), State: "accepted"}}, v.Keys...)
	}
	return v, nil
}
func hopGate(t *testing.T, e *fixtureEnvironment) *r.Gate {
	t.Helper()
	e.counter++
	journal, err := r.OpenJournal(filepath.Join(e.root, "hop-registry-"+string(rune('A'+e.counter))), true)
	if err != nil {
		t.Fatal(err)
	}
	e.journals = append(e.journals, journal)
	gate, err := r.NewGate(r.Config{Source: "adk-inert-fixture", Registry: "web:agent.example", Network: "local"}, hopSource{e}, e.clock, journal)
	if err != nil {
		t.Fatal(err)
	}
	return gate
}
func hopAuthority(t *testing.T, e *fixtureEnvironment, did string) *g.RegistryAuthority {
	t.Helper()
	a, err := g.NewRegistryAuthority(hopGate(t, e), did, did+"#signing-1")
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func hopEndpoint(t *testing.T, e *fixtureEnvironment, did string, initiator bool) *h.CompletionEndpoint010 {
	t.Helper()
	key := e.bob
	if did == fixtureAlice {
		key = e.alice
	}
	kem := e.kem
	if initiator {
		kem = nil
	}
	e.counter++
	replay, err := h.OpenReplayJournal010(filepath.Join(e.root, "hop-replay-"+string(rune('A'+e.counter))), true, e.clock)
	if err != nil {
		t.Fatal(err)
	}
	e.replays = append(e.replays, replay)
	ep, err := h.NewCompletionEndpoint010(did, did+"#signing-1", key.Seed(), kem, hopGate(t, e), e.clock, replay)
	if err != nil {
		t.Fatal(err)
	}
	return ep
}

type hopBackend struct{ key ed25519.PrivateKey }

func (s hopBackend) PublicKey(ctx context.Context) (ed25519.PublicKey, error) {
	if ctx.Err() != nil {
		return nil, g.ErrInvalid
	}
	return s.key.Public().(ed25519.PublicKey), nil
}
func (s hopBackend) Sign(ctx context.Context, b []byte) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, g.ErrInvalid
	}
	return ed25519.Sign(s.key, b), nil
}
func hopResultSigner(t *testing.T, a *g.RegistryAuthority, did string, key ed25519.PrivateKey) *p.ResultSigner {
	t.Helper()
	signer, err := p.NewResultSigner(context.Background(), a, did, did+"#signing-1", hopBackend{key})
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

type hopIntentBackend struct {
	hopBackend
	path  string
	calls atomic.Int64
}

func (s *hopIntentBackend) Sign(ctx context.Context, b []byte) ([]byte, error) {
	const domain = "sage-execution-intent|0.10.0\x00"
	marker, e := os.ReadFile(s.path + ".issuance")
	if e != nil || !bytes.HasPrefix(b, []byte(domain)) || string(marker) != "sage-intent-issuance|0.10.0\n"+fixtureHash(b[len(domain):])+"\n" {
		return nil, g.ErrInvalid
	}
	s.calls.Add(1)
	return s.hopBackend.Sign(ctx, b)
}

type hopPolicy struct {
	request  atomic.Pointer[capture.HopRequest]
	enabled  atomic.Bool
	approved *approvedHopFixture
}

func (p *hopPolicy) Bindings(ctx context.Context, issuer, id string) (string, []byte, []byte, error) {
	request := p.request.Load()
	if !p.enabled.Load() || request == nil || issuer != fixtureBob || id != request.ID() {
		return "", nil, nil, g.ErrInvalid
	}
	if p.approved != nil {
		op := p.approved.operation.Load()
		if op == nil {
			return "", nil, nil, g.ErrInvalid
		}
		return op.Bindings(ctx, issuer, id)
	}
	manifest := fixtureManifest()
	policy := fixtureJSON(map[string]any{"version": "0.10.0", "issuer": fixtureBob, "epoch": "00000000-0000-4000-8000-000000000011", "engine": "inert-independent-hop-policy/1", "artifacts": json.RawMessage(manifest)})
	return request.Digest(), policy, manifest, nil
}
func (p *hopPolicy) Authorize(ctx context.Context, issuer, tool string, args []byte) error {
	if p.approved != nil {
		op := p.approved.operation.Load()
		if op == nil {
			return g.ErrInvalid
		}
		return op.Authorize(ctx, issuer, tool, args)
	}
	if ctx.Err() != nil || !p.enabled.Load() || issuer != fixtureBob || tool != "sum" || !bytes.Equal(args, []byte(`{"a":2,"b":3}`)) {
		return g.ErrInvalid
	}
	return nil
}
func (p *hopPolicy) ApproveIntent(ctx context.Context, raw []byte) error {
	var i struct {
		Recipient string          `json:"recipient"`
		Parent    string          `json:"parent_call_id"`
		ID        string          `json:"request_id"`
		Tool      string          `json:"tool"`
		Arguments json.RawMessage `json:"arguments"`
	}
	request := p.request.Load()
	if request == nil || json.Unmarshal(raw, &i) != nil || i.Recipient != fixtureAlice || i.Parent != request.ParentCallID() || i.ID != request.ID() {
		return g.ErrInvalid
	}
	return p.Authorize(ctx, fixtureBob, i.Tool, i.Arguments)
}

type hopUpstreamPolicy struct {
	inner   *fixturePolicy
	enabled atomic.Bool
}

func (p *hopUpstreamPolicy) Bindings(ctx context.Context, issuer, id string) (string, []byte, []byte, error) {
	if !p.enabled.Load() {
		return "", nil, nil, g.ErrInvalid
	}
	return p.inner.Bindings(ctx, issuer, id)
}
func (p *hopUpstreamPolicy) Authorize(ctx context.Context, issuer, tool string, args []byte) error {
	if !p.enabled.Load() {
		return g.ErrInvalid
	}
	return p.inner.Authorize(ctx, issuer, tool, args)
}

type hopExecutor struct {
	t                 *testing.T
	env               *fixtureEnvironment
	host              *capture.Host
	downstream        *g.MCPHost
	listener          net.Listener
	endpoint          *h.CompletionEndpoint010
	upstream          *hopUpstreamPolicy
	upstreamAuthority *g.RegistryAuthority
	policy            *hopPolicy
	authority, result *g.RegistryAuthority
	signer            *hopIntentBackend
	protectedSigner   g.IntentSigner
	mode              string
	approved          *approvedHopFixture
	captured          atomic.Pointer[capture.HopRequest]
	calls             atomic.Int64
}

func (e *hopExecutor) Check(ctx context.Context, manifest, tool string) error {
	return (&fixtureLoadedTool{}).Check(ctx, manifest, tool)
}
func (e *hopExecutor) Run(ctx context.Context, i *g.Invocation) ([]byte, error) {
	e.calls.Add(1)
	services := capture.HopServices{Recipient: fixtureBob, Invocation: i, Authority: e.upstreamAuthority, Policy: e.upstream}
	request, err := e.host.CaptureHop(ctx, services)
	if err != nil {
		return nil, err
	}
	e.captured.Store(request)
	e.policy.request.Store(request)
	if e.mode == "restore" {
		restored, err := e.host.RestoreHop(ctx, request.ID(), request.Digest(), services)
		if err != nil || restored.ID() != request.ID() || restored.Digest() != request.Digest() {
			return nil, g.ErrInvalid
		}
		request = restored
		e.captured.Store(request)
		e.policy.request.Store(request)
	}
	inputs, err := request.Inputs(ctx)
	if err != nil || len(inputs) != 1 || !bytes.Equal(inputs[0], i.CanonicalIntent()) {
		return nil, g.ErrInvalid
	}
	if e.mode == "upstream-policy-loss" {
		e.upstream.enabled.Store(false)
		_, err = request.Inputs(ctx)
		e.upstream.enabled.Store(true)
		if err == nil {
			return nil, g.ErrInvalid
		}
		if _, again := request.Inputs(ctx); again == nil {
			return nil, g.ErrInvalid
		}
		return nil, err
	}
	if e.mode == "downstream-policy-denied" {
		e.policy.enabled.Store(false)
	}
	var policy g.IssuancePolicy = e.policy
	var measurement g.IntentMeasurement = &fixtureLoadedTool{}
	arguments := []byte(`{"a":2,"b":3}`)
	tool := "sum"
	if e.approved != nil {
		operation, err := e.approved.open(ctx, request)
		if err != nil {
			return nil, err
		}
		defer func() { _ = operation.Close(context.Background()) }()
		policy, measurement = operation, operation
		if e.mode == "approved-upstream-policy-loss" {
			e.upstream.enabled.Store(false)
			denied := operation.Authorize(ctx, fixtureBob, "calculator", []byte(`{"a":2,"b":3,"operation":"add"}`))
			e.upstream.enabled.Store(true)
			if denied == nil || operation.Authorize(ctx, fixtureBob, "calculator", []byte(`{"a":2,"b":3,"operation":"add"}`)) == nil {
				return nil, g.ErrInvalid
			}
			return nil, denied
		}
		tool = "calculator"
		arguments = []byte(`{"a":2,"b":3,"operation":"add"}`)
		if e.mode == "approved-policy-denied" {
			arguments = []byte(`{"a":2,"b":4,"operation":"add"}`)
		}
	}
	config := g.IssuerServices{Client: g.ClientServices{IntentAuthority: e.authority, ResultAuthority: e.result, Policy: e.policy, Clock: e.env.clock, Sender: fixtureNoSender{}, ExpectedIssuer: fixtureBob, ExpectedRecipient: fixtureAlice}, Policy: policy, Signer: e.protectedSigner, Measurement: measurement, KeyID: fixtureBob + "#signing-1"}
	issuer, err := request.NewIntentIssuer(ctx, config)
	if err != nil {
		return nil, err
	}
	defer func() { _ = issuer.Retire() }()
	approved, err := issuer.Authorize(ctx, g.IntentProposal{Tool: tool, Arguments: arguments, LifetimeSeconds: 300})
	if err != nil {
		return nil, err
	}
	durable, err := issuer.Issue(ctx, e.signer.path, approved)
	if err != nil {
		return nil, err
	}
	intent, err := durable.JournaledIntent()
	closeErr := durable.Close()
	if err != nil || closeErr != nil {
		return nil, g.ErrInvalid
	}
	var wire struct {
		Intent struct {
			Issuer   string `json:"issuer"`
			Parent   string `json:"parent_call_id"`
			Request  string `json:"request_id"`
			Call     string `json:"call_id"`
			Original string `json:"original_digest"`
		} `json:"intent"`
	}
	if json.Unmarshal(intent, &wire) != nil || wire.Intent.Issuer != fixtureBob || wire.Intent.Parent != request.ParentCallID() || wire.Intent.Call == request.ParentCallID() || wire.Intent.Request != request.ID() || wire.Intent.Original != request.Digest() {
		return nil, g.ErrInvalid
	}
	conn, err := net.DialTimeout("tcp", e.listener.Addr().String(), time.Second)
	if err != nil {
		return nil, err
	}
	var delivered *g.ClientDelivery
	connectionConfig := g.MCPConnectionConfig{Role: g.MCPInitiator, Recipient: fixtureAlice, RecipientKey: fixtureAlice + "#signing-1", Name: "fixed-hop", Version: "1", TTLSeconds: 300, Timeout: 2 * time.Second}
	if e.approved != nil {
		connectionConfig.Timeout = 10 * time.Second
	}
	err = e.downstream.Connect(ctx, conn, connectionConfig, &fixtureHandler{endpoint: e.endpoint, handle: func(callCtx context.Context, c *g.MCPConnection) error {
		path := e.signer.path
		if e.mode == "missing-journal" {
			path = filepath.Join(e.env.root, "missing-hop-journal")
		}
		if err := request.OpenMCPClient(callCtx, c, path, intent, g.MCPClientServices{IntentAuthority: e.authority, ResultAuthority: e.result, Policy: e.policy, Clock: e.env.clock}); err != nil {
			e.t.Logf("fixed child handoff refused: %v", err)
			return err
		}
		var err error
		polls := 5
		if e.approved != nil {
			polls = 12
		}
		for n := 0; n < polls; n++ {
			time.Sleep(time.Second)
			if e.approved == nil {
				e.env.clock.mono.Add(1000)
			}
			delivered, err = c.Exchange()
			if err != nil {
				e.t.Logf("fixed child exchange refused: %v; worker context=%v", err, ctx.Err())
				return err
			}
			if delivered != nil && delivered.Status() != "pending" {
				return nil
			}
		}
		return g.ErrInvalid
	}})
	if err != nil {
		e.t.Logf("fixed child connection refused: %v", err)
	}
	if err != nil || delivered == nil || delivered.Status() != "completed" || !delivered.FirstTerminal() || !bytes.Equal(delivered.Output(), e.expectedOutput()) {
		return nil, g.ErrInvalid
	}
	return delivered.Output(), nil
}

func (e *hopExecutor) expectedOutput() []byte {
	if e.approved != nil {
		return []byte(`{"output":5,"success":true}`)
	}
	return []byte(`{"sum":5}`)
}

func TestAdmittedHopCaptureNativeRuntime(t *testing.T) {
	for _, mode := range []string{"allowed", "restore", "upstream-policy-loss", "downstream-policy-denied", "missing-journal"} {
		t.Run(mode, func(t *testing.T) { runAdmittedHop(t, mode) })
	}
}
func runAdmittedHop(t *testing.T, mode string) {
	t.Helper()
	env := newFixtureEnvironment(t)
	defer env.close(t)
	store, err := capture.OpenFileStore(filepath.Join(env.root, "hop-originals"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	}()
	capturer, _ := capture.NewHost(store)
	root, err := capturer.Capture(context.Background(), [][]byte{[]byte("fixed local addition request")})
	if err != nil {
		t.Fatal(err)
	}
	rootPolicy := &fixturePolicy{id: root.ID(), digest: root.Digest()}
	upstream := &hopUpstreamPolicy{inner: rootPolicy}
	upstream.enabled.Store(true)
	downstreamPolicy := &hopPolicy{}
	downstreamPolicy.enabled.Store(true)
	// Construct every provider/endpoint before concurrency; fixture counters are setup-only.
	rootIntent, rootResult := hopAuthority(t, env, fixtureAlice), hopAuthority(t, env, fixtureBob)
	parentIntent, parentResult := hopAuthority(t, env, fixtureAlice), hopAuthority(t, env, fixtureBob)
	childIntent, childResult := hopAuthority(t, env, fixtureBob), hopAuthority(t, env, fixtureAlice)
	finalIntent, finalResult := hopAuthority(t, env, fixtureBob), hopAuthority(t, env, fixtureAlice)
	rootEP, parentEP := hopEndpoint(t, env, fixtureAlice, true), hopEndpoint(t, env, fixtureBob, false)
	childEP, finalEP := hopEndpoint(t, env, fixtureBob, true), hopEndpoint(t, env, fixtureAlice, false)
	bounds := fixtureBounds()
	bounds.Worker = 10 * time.Second
	bounds.Request = 20 * time.Second
	bounds.Client = 20 * time.Second
	lifetime := 20 * time.Second
	if strings.HasPrefix(mode, "approved-") {
		// Includes durable upstream/capture/loader checks under race-enabled CI.
		// Preserve finite request/worker/client bounds and all core validation.
		bounds.Worker = 30 * time.Second
		bounds.Request, bounds.Client = time.Minute, time.Minute
		lifetime = time.Minute
	}
	finalTool := &fixtureLoadedTool{}
	var approved *approvedHopFixture
	var finalLoaded toolhost.LoadedTool = finalTool
	toolName := "sum"
	if strings.HasPrefix(mode, "approved-") {
		approved = newApprovedHopFixture(t, env, mode)
		defer func() {
			if err := approved.factory.Close(context.Background()); err != nil {
				t.Error(err)
			}
		}()
		downstreamPolicy.approved = approved
		finalLoaded = approved
		toolName = "calculator"
	}
	manifest := fixtureManifest()
	if approved != nil {
		manifest = approved.config.Manifest
	}
	md, _ := g.ManifestCommitment(manifest)
	final, err := toolhost.Open(filepath.Join(env.root, "final-ledger"), true, fixtureAlice, toolhost.Services{IntentAuthority: finalIntent, ResultAuthority: finalResult, Policy: downstreamPolicy, Signer: hopResultSigner(t, finalResult, fixtureAlice, env.alice), Clock: env.clock}, []toolhost.Binding{{Name: toolName, ManifestDigest: md, Tool: finalLoaded}}, bounds)
	if err != nil {
		t.Fatal(err)
	}
	nativeServices := func(intent, result *g.RegistryAuthority, policy g.IntentPolicy, exec g.MCPExecutor, did string, key ed25519.PrivateKey) g.MCPHostServices {
		return g.MCPHostServices{IntentAuthority: intent, ResultAuthority: result, Policy: policy, Executor: exec, Signer: hopResultSigner(t, result, did, key), Clock: env.clock}
	}
	child, err := g.OpenMCPHost(filepath.Join(env.root, "child-ledger"), true, fixtureAlice, nativeServices(childIntent, childResult, downstreamPolicy, &hopNoEffect{}, fixtureAlice, env.alice), bounds)
	if err != nil {
		t.Fatal(err)
	}
	signer := &hopIntentBackend{hopBackend: hopBackend{env.bob}, path: filepath.Join(env.root, "child-journal")}
	protectedSigner, err := p.NewIntentSigner(context.Background(), childIntent, fixtureBob, fixtureBob+"#signing-1", signer)
	if err != nil {
		t.Fatal(err)
	}
	// Only issuer services receive signing custody; executor keeps the protected adapter.
	exec := &hopExecutor{t: t, env: env, host: capturer, downstream: child, endpoint: childEP, upstream: upstream, upstreamAuthority: parentIntent, policy: downstreamPolicy, authority: childIntent, result: childResult, signer: signer, protectedSigner: protectedSigner, mode: mode, approved: approved}
	parent, err := g.OpenMCPHost(filepath.Join(env.root, "parent-ledger"), true, fixtureBob, nativeServices(parentIntent, parentResult, rootPolicy, exec, fixtureBob, env.bob), bounds)
	if err != nil {
		t.Fatal(err)
	}
	client, err := g.OpenMCPHost(filepath.Join(env.root, "root-ledger"), true, fixtureBob, nativeServices(rootIntent, rootResult, rootPolicy, &hopNoEffect{}, fixtureBob, env.bob), bounds)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), lifetime)
	defer cancel()
	closeNative := func(host *g.MCPHost) {
		closeCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		if err := host.Close(closeCtx); err != nil {
			t.Error(err)
		}
	}
	defer closeNative(client)
	defer closeNative(parent)
	defer closeNative(child)
	defer func() {
		closeCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		if err := final.Close(closeCtx); err != nil {
			t.Error(err)
		}
	}()
	finalListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if approved != nil {
		finalListener = approvedHopListener{Listener: finalListener, t: t, label: "final"}
	}
	exec.listener = finalListener
	parentListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if approved != nil {
		parentListener = approvedHopListener{Listener: parentListener, t: t, label: "parent"}
	}
	env.clock.mono.Add(360000)
	if approved != nil {
		env.clock.real.Store(&fixtureClockOrigin{base: env.clock.mono.Load(), started: time.Now()})
	}
	rootSigner := &fixtureIntentSigner{env: env, path: filepath.Join(env.root, "root-journal")}
	rootProtected, err := p.NewIntentSigner(context.Background(), rootIntent, fixtureAlice, fixtureAlice+"#signing-1", rootSigner)
	if err != nil {
		t.Fatal(err)
	}
	rootIssuer, err := root.NewIntentIssuer(ctx, g.IssuerServices{Client: g.ClientServices{IntentAuthority: rootIntent, ResultAuthority: rootResult, Policy: rootPolicy, Clock: env.clock, Sender: fixtureNoSender{}, ExpectedIssuer: fixtureAlice, ExpectedRecipient: fixtureBob}, Policy: rootPolicy, Signer: rootProtected, Measurement: &fixtureLoadedTool{}, KeyID: fixtureAlice + "#signing-1"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rootIssuer.Retire() }()
	approval, err := rootIssuer.Authorize(ctx, g.IntentProposal{Tool: "sum", Arguments: []byte(`{"a":2,"b":3}`), LifetimeSeconds: 300})
	if err != nil {
		t.Fatal(err)
	}
	durable, err := rootIssuer.Issue(ctx, rootSigner.path, approval)
	if err != nil {
		t.Fatal(err)
	}
	rootIntentBytes, err := durable.JournaledIntent()
	if err != nil {
		t.Fatal(err)
	}
	if err := durable.Close(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 2)
	handler := func(ep *h.CompletionEndpoint010) *fixtureHandler {
		return &fixtureHandler{endpoint: ep, handle: func(_ context.Context, c *g.MCPConnection) error {
			for {
				if err := c.ServeOne(); err != nil {
					return err
				}
			}
		}}
	}
	finalConfig := g.MCPConnectionConfig{Role: g.MCPResponder, Name: "fixed-final", Version: "1", TTLSeconds: 300, Timeout: 2 * time.Second}
	parentConfig, clientConfig := fixtureConfig(false), fixtureConfig(true)
	if approved != nil {
		finalConfig.Timeout, parentConfig.Timeout, clientConfig.Timeout = 10*time.Second, 10*time.Second, 10*time.Second
	}
	go func() {
		done <- final.Serve(ctx, finalListener, 1, finalConfig, handler(finalEP))
	}()
	go func() { done <- parent.Serve(ctx, parentListener, 1, parentConfig, handler(parentEP)) }()
	conn, err := net.DialTimeout("tcp", parentListener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var delivery *g.ClientDelivery
	err = client.Connect(ctx, conn, clientConfig, &fixtureHandler{endpoint: rootEP, handle: func(callCtx context.Context, c *g.MCPConnection) error {
		if err := root.OpenMCPClient(callCtx, c, rootSigner.path, rootIntentBytes, g.MCPClientServices{IntentAuthority: rootIntent, ResultAuthority: rootResult, Policy: rootPolicy, Clock: env.clock}); err != nil {
			return err
		}
		polls := 8
		if approved != nil {
			polls = 20
		}
		for n := 0; n < polls; n++ {
			time.Sleep(time.Second)
			if approved == nil {
				env.clock.mono.Add(1000)
			}
			var err error
			delivery, err = c.Exchange()
			if err != nil {
				t.Logf("fixed root exchange refused: %v", err)
				return err
			}
			if delivery != nil && delivery.Status() != "pending" {
				return nil
			}
		}
		return g.ErrInvalid
	}})
	cancel()
	for n := 0; n < 2; n++ {
		if e := <-done; e != nil && !errors.Is(e, context.Canceled) {
			t.Errorf("serve lifecycle: %v", e)
		}
	}
	if err != nil {
		t.Fatalf("hop runtime: %v", err)
	}
	allowed := mode == "allowed" || mode == "restore" || mode == "approved-allowed"
	if delivery == nil {
		t.Fatal("missing root verified delivery")
	}
	effects := finalTool.calls.Load()
	if approved != nil {
		effects = approved.effects.Load()
	}
	if allowed {
		if delivery.Status() != "completed" || !bytes.Equal(delivery.Output(), exec.expectedOutput()) || effects != 1 || signer.calls.Load() != 1 {
			t.Fatalf("completed chain: status=%s effects=%d signs=%d", delivery.Status(), effects, signer.calls.Load())
		}
	} else {
		if delivery.Status() != "unknown" || len(delivery.Output()) != 0 || effects != 0 {
			t.Fatal("refused hop reached effect")
		}
		want := int64(0)
		if mode == "missing-journal" {
			want = 1
		}
		if signer.calls.Load() != want {
			t.Fatal("unexpected key use on refusal")
		}
	}
	if exec.calls.Load() != 1 || rootSigner.calls.Load() != 1 {
		t.Fatal("native worker or root issuance repeated")
	}
	retained := exec.captured.Load()
	if retained == nil {
		t.Fatal("parent capture missing")
	}
	if approved != nil {
		checks := approved.measurement.checks.Load()
		unavailable, err := b.OpenHop(context.Background(), retained, approved.config)
		if err == nil || unavailable != nil || approved.measurement.checks.Load() != checks {
			t.Fatal("finished parent reached another approved loader")
		}
	}
	if _, e := retained.Inputs(context.Background()); e == nil {
		t.Fatal("completed parent remained usable")
	}
	if mode == "missing-journal" {
		if _, e := os.Stat(filepath.Join(env.root, "missing-hop-journal")); !os.IsNotExist(e) {
			t.Fatal("missing history recreated")
		}
	}
}

type hopNoEffect struct{}

func (*hopNoEffect) Check(context.Context, string, string) error        { return g.ErrInvalid }
func (*hopNoEffect) Run(context.Context, *g.Invocation) ([]byte, error) { return nil, g.ErrInvalid }
