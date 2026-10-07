// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/capture"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	calc "github.com/sage-x-project/sage-adk/core/guardcalculator"
	p "github.com/sage-x-project/sage-adk/core/guardservices"
	"github.com/sage-x-project/sage-adk/core/toolhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// Only private files, loopback, ephemeral keys and inert arithmetic are used.
// Fixture registry/measurement services are not deployment attestation.
func TestNativeGuardedToolRuntime(t *testing.T)        { runNativeFixture(t, "allowed") }
func TestApprovedOperationNativeRuntime(t *testing.T)  { runNativeFixture(t, "approved-binding") }
func TestCompiledCalculatorNativeRuntime(t *testing.T) { runNativeFixture(t, "calculator-binding") }
func runNativeFixture(t *testing.T, mode string) {
	t.Helper()
	env := newFixtureEnvironment(t)
	defer env.close(t)
	store, err := capture.OpenFileStore(filepath.Join(env.root, "originals"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := store.Close(); e != nil {
			t.Error(e)
		}
	}()
	capturer, err := capture.NewHost(store)
	if err != nil {
		t.Fatal(err)
	}
	request, err := capturer.Capture(context.Background(), [][]byte{[]byte(" 계산 원본\n"), []byte("e\u0301")})
	if err != nil {
		t.Fatal(err)
	}
	policy := &fixturePolicy{id: request.ID(), digest: request.Digest()}
	serverPolicy := &fixturePolicy{id: request.ID(), digest: request.Digest()}
	loaded := &fixtureLoadedTool{seen: make(chan []byte, 1)}
	switch mode {
	case "policy":
		serverPolicy.failAt = 3
	case "panic", "error", "array", "invalid", "oversize", "cancel":
		loaded.mode = mode
	}
	manifest, err := g.ManifestCommitment(fixtureManifest())
	if err != nil {
		t.Fatal(err)
	}
	bindings := []toolhost.Binding{{Name: "sum", ManifestDigest: manifest, Tool: loaded}}
	var clientPolicy g.IssuancePolicy = policy
	var receiverPolicy g.IntentPolicy = serverPolicy
	var measurement g.IntentMeasurement = loaded
	if mode == "approved-binding" {
		directory := filepath.Join(env.root, "approved-artifacts")
		if err = os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
		rule := fixtureJSON(map[string]any{"version": "0.10.0", "recipient": fixtureBob, "keyid": fixtureAlice + "#signing-1", "tool": "sum", "arguments": map[string]int{"a": 2, "b": 3}, "max_lifetime": 300})
		artifacts := map[string][]byte{"arithmetic.fixture": []byte(fixtureMaterial), "evaluator.fixture": []byte("inert evaluator fixture"), "rules.json": rule}
		for path, data := range artifacts {
			if err = os.WriteFile(filepath.Join(directory, path), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		policyArtifacts := fixtureJSON(map[string]any{"version": "0.10.0", "files": []any{map[string]any{"path": "evaluator.fixture", "sha256": fixtureHash(artifacts["evaluator.fixture"])}, map[string]any{"path": "rules.json", "sha256": fixtureHash(rule)}}})
		descriptor := fixtureJSON(map[string]any{"version": "0.10.0", "issuer": fixtureAlice, "epoch": "00000000-0000-4000-8000-000000000001", "engine": b.Engine, "artifacts": json.RawMessage(policyArtifacts)})
		open := func(instance *fixtureLoadedTool) *b.Operation {
			operation, e := b.Open(context.Background(), request, b.Config{Directory: directory, Policy: descriptor, Manifest: fixtureManifest(), Limits: b.Limits{FileBytes: 1024, TotalBytes: 4096}, Factory: &fixtureOperationFactory{instance: instance}})
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() {
				if e := operation.Close(context.Background()); e != nil {
					t.Error(e)
				}
			})
			return operation
		}
		// Separate local issuer and receiver bindings; loaded-instance and Registry
		// attestation are explicitly inert fixture providers, not deployment evidence.
		issuerOperation := open(&fixtureLoadedTool{})
		receiverOperation := open(loaded)
		clientPolicy, receiverPolicy, measurement = issuerOperation, receiverOperation, issuerOperation
		binding, e := receiverOperation.Binding(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		bindings = []toolhost.Binding{binding}
	}
	toolName := "sum"
	proposalArguments := []byte(`{"a":2,"b":3}`)
	expectedOutput := []byte(`{"sum":5}`)
	if mode == "calculator-binding" {
		toolName = "calculator"
		proposalArguments = []byte(`{"a":2,"b":3,"operation":"add"}`)
		expectedOutput = []byte(`{"output":5,"success":true}`)
		directory := filepath.Join(env.root, "calculator-artifacts")
		if err = os.Mkdir(directory, 0700); err != nil {
			t.Fatal(err)
		}
		rule := fixtureJSON(map[string]any{"version": "0.10.0", "recipient": fixtureBob, "keyid": fixtureAlice + "#signing-1", "tool": toolName, "arguments": json.RawMessage(proposalArguments), "max_lifetime": 300})
		configuration := fixtureJSON(map[string]any{"version": "0.10.0", "tool": toolName, "image_path": "image.fixture", "operations": []string{"add"}, "absolute_operand_limit": 100})
		artifacts := map[string][]byte{"image.fixture": []byte("synthetic runtime image observation"), "rules.json": rule, calc.ConfigurationPath: configuration}
		for path, data := range artifacts {
			if err = os.WriteFile(filepath.Join(directory, path), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		files := func(names ...string) []byte {
			rows := []any{}
			for _, name := range names {
				rows = append(rows, map[string]any{"path": name, "sha256": fixtureHash(artifacts[name])})
			}
			return fixtureJSON(map[string]any{"version": "0.10.0", "files": rows})
		}
		descriptor := fixtureJSON(map[string]any{"version": "0.10.0", "issuer": fixtureAlice, "epoch": "00000000-0000-4000-8000-000000000001", "engine": b.Engine, "artifacts": json.RawMessage(files("image.fixture", "rules.json"))})
		component := files(calc.ConfigurationPath, "image.fixture")
		// Real compiled calculator instances, but synthetic runtime-image appraisal
		// and Registry providers. This is safe local integration, not deployment evidence.
		factory, e := calc.NewFactory(calc.Config{Measurement: fixtureCalculatorMeasurement{}, MaxInstances: 2})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			if e := factory.Close(context.Background()); e != nil {
				t.Error(e)
			}
		})
		open := func() *b.Operation {
			op, e := b.Open(context.Background(), request, b.Config{Directory: directory, Policy: descriptor, Manifest: component, Limits: b.Limits{FileBytes: 4096, TotalBytes: 16384}, Factory: factory})
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() {
				if e := op.Close(context.Background()); e != nil {
					t.Error(e)
				}
			})
			return op
		}
		issuerOperation, receiverOperation := open(), open()
		clientPolicy, receiverPolicy, measurement = issuerOperation, receiverOperation, issuerOperation
		binding, e := receiverOperation.Binding(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		bindings = []toolhost.Binding{binding}
	}
	clientIntent, clientResult := env.authority(t, fixtureAlice), env.authority(t, fixtureBob)
	serverIntent, serverResult := env.authority(t, fixtureAlice), env.authority(t, fixtureBob)
	hostServices := func(intent, result *g.RegistryAuthority, p g.IntentPolicy) toolhost.Services {
		return toolhost.Services{IntentAuthority: intent, ResultAuthority: result, Policy: p, Signer: env.resultSigner(t, result), Clock: env.clock}
	}
	serverPath := filepath.Join(env.root, "server-ledger")
	server, err := toolhost.Open(serverPath, true, fixtureBob, hostServices(serverIntent, serverResult, receiverPolicy), bindings, fixtureBounds())
	if err != nil {
		t.Fatal(err)
	}
	client, err := toolhost.Open(filepath.Join(env.root, "client-ledger"), true, fixtureBob, hostServices(clientIntent, clientResult, clientPolicy), bindings, fixtureBounds())
	if err != nil {
		t.Fatal(err)
	}
	closeHost := func(host *toolhost.Host) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if e := host.Close(ctx); e != nil {
			t.Error(e)
		}
	}
	defer closeHost(server)
	defer closeHost(client)
	serverEndpoint, clientEndpoint := env.endpoint(t, false), env.endpoint(t, true)
	// New replay journals require the core's full UTC/monotonic quarantine.
	// Advance only this injected test clock before issuing or connecting.
	env.clock.mono.Add(360000)
	journal := filepath.Join(env.root, "client-journal")
	signer := &fixtureIntentSigner{env: env, path: journal}
	protectedSigner, err := p.NewIntentSigner(context.Background(), clientIntent, fixtureAlice, fixtureAlice+"#signing-1", signer)
	if err != nil {
		t.Fatal(err)
	}
	issuer, err := request.NewIntentIssuer(context.Background(), g.IssuerServices{Client: g.ClientServices{IntentAuthority: clientIntent, ResultAuthority: clientResult, Policy: clientPolicy, Clock: env.clock, Sender: fixtureNoSender{}, ExpectedIssuer: fixtureAlice, ExpectedRecipient: fixtureBob}, Policy: clientPolicy, Signer: protectedSigner, Measurement: measurement, KeyID: fixtureAlice + "#signing-1"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if e := issuer.Retire(); e != nil {
			t.Error(e)
		}
	}()
	approved, err := issuer.Authorize(context.Background(), g.IntentProposal{Tool: toolName, Arguments: proposalArguments, LifetimeSeconds: 300})
	if err != nil {
		t.Fatal(err)
	}
	durable, err := issuer.Issue(context.Background(), journal, approved)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := durable.JournaledIntent()
	if err != nil {
		t.Fatal(err)
	}
	if err = durable.Close(); err != nil {
		t.Fatal(err)
	}
	// Issuer and receiver measurements are independent. Receiver failure counters
	// are reset after issuing, before its actual admitted worker is exercised.
	loaded.checks.Store(0)
	if mode == "measurement" {
		loaded.failAt = 3
	}
	before, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Serve(ctx, listener, 1, fixtureConfig(false), &fixtureHandler{endpoint: serverEndpoint, handle: func(_ context.Context, c *g.MCPConnection) error {
			for {
				if e := c.ServeOne(); e != nil {
					return e
				}
			}
		}})
	}()
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	stage := "connecting"
	var delivery *g.ClientDelivery
	terminalRefused := false
	err = client.Connect(ctx, conn, fixtureConfig(true), &fixtureHandler{endpoint: clientEndpoint, handle: func(callCtx context.Context, c *g.MCPConnection) error {
		stage = "opening"
		if e := request.OpenMCPClient(callCtx, c, journal, intent, g.MCPClientServices{IntentAuthority: clientIntent, ResultAuthority: clientResult, Policy: clientPolicy, Clock: env.clock}); e != nil {
			return e
		}
		for n := 0; n < 12; n++ {
			time.Sleep(20 * time.Millisecond)
			env.clock.mono.Add(1000)
			var e error
			stage = "exchanging"
			delivery, e = c.Exchange()
			if e != nil {
				return e
			}
			if delivery.Status() == "completed" || delivery.Status() == "unknown" {
				if mode == "terminal" {
					stage = "terminal-refusal"
					// Refusal retires this native connection. Connect must fail too;
					// it cannot be treated as a successful session termination.
					if repeated, e := c.Exchange(); e == nil || repeated != nil {
						return errors.New("terminal client dispatched again")
					}
					terminalRefused = true
					return errors.New("expected terminal handoff refusal")
				}
				return nil
			}
		}
		return errors.New("fixture terminal result unavailable")
	}})
	cancel()
	if e := <-serverDone; e != nil && !errors.Is(e, context.Canceled) {
		t.Errorf("server lifecycle: %v", e)
	}
	if mode == "terminal" && terminalRefused && err != nil {
		err = nil // Expected native connection retirement after the refused handoff.
	}
	if err != nil {
		t.Fatalf("%s: %v; receiver policy=%d checks=%d effects=%d reads=%d", stage, err, serverPolicy.calls.Load(), loaded.checks.Load(), loaded.calls.Load(), env.reads.Load())
	}
	if signer.calls.Load() != 1 {
		t.Fatal("intent re-signed during transfer")
	}
	after, err := os.ReadFile(journal)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(after, before) {
		t.Fatal("journal identity replaced")
	}
	if mode == "allowed" || mode == "terminal" || mode == "approved-binding" || mode == "calculator-binding" {
		if delivery == nil || delivery.Status() != "completed" || !delivery.FirstTerminal() || !bytes.Equal(delivery.Output(), expectedOutput) || (mode != "calculator-binding" && loaded.calls.Load() != 1) || (mode == "calculator-binding" && loaded.calls.Load() != 0) {
			t.Fatalf("verified delivery/effect: %+v calls=%d", delivery, loaded.calls.Load())
		}
		if mode != "calculator-binding" {
			if args := <-loaded.seen; !bytes.Equal(args, proposalArguments) {
				t.Fatal("arguments changed")
			}
		}
	} else {
		if delivery == nil || delivery.Status() != "unknown" || len(delivery.Output()) != 0 {
			t.Fatal("uncertain tool exposed completed output")
		}
		want := int64(1)
		if mode == "measurement" || mode == "policy" {
			want = 0
		}
		if loaded.calls.Load() != want {
			t.Fatalf("denial effects: %d", loaded.calls.Load())
		}
	}
	if mode == "terminal" && !terminalRefused {
		t.Fatal("terminal refusal was not reached")
	}
	// Completed/uncertain replay fences survive a clean host restart; neither
	// terminal may be reset by choosing create=true on the existing ledger.
	closeHost(server)
	if _, e := toolhost.Open(serverPath, true, fixtureBob, hostServices(serverIntent, serverResult, receiverPolicy), bindings, fixtureBounds()); e == nil {
		t.Fatal("existing ledger recreated")
	}
	recovered, e := toolhost.Open(serverPath, false, fixtureBob, hostServices(serverIntent, serverResult, receiverPolicy), bindings, fixtureBounds())
	if e != nil {
		t.Fatal(e)
	}
	closeHost(recovered)
	records, e := os.ReadFile(serverPath)
	if e != nil {
		t.Fatal(e)
	}
	var last map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(records), []byte("\n"))[1:] {
		if json.Unmarshal(line, &last) != nil {
			t.Fatal("ledger record")
		}
	}
	expected := "COMPLETED"
	if mode != "allowed" && mode != "terminal" && mode != "approved-binding" && mode != "calculator-binding" {
		expected = "UNKNOWN"
	}
	if last["state"] != expected {
		t.Fatalf("recovery changed terminal: %v", last["state"])
	}
	if loaded.calls.Load() > 1 {
		t.Fatal("recovery executed again")
	}
}

// Deliberately synthetic image appraisal. Do not use this provider in deployments.
type fixtureCalculatorMeasurement struct{}

func (fixtureCalculatorMeasurement) Check(ctx context.Context, snapshot *b.Snapshot) error {
	if ctx.Err() != nil || snapshot == nil {
		return errors.New("fixture image observation unavailable")
	}
	return nil
}
