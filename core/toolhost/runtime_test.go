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
	"github.com/sage-x-project/sage-adk/core/toolhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// Only private files, loopback, ephemeral keys and inert arithmetic are used.
// Fixture registry/measurement services are not deployment attestation.
func TestNativeGuardedToolRuntime(t *testing.T) { runNativeFixture(t, "allowed") }
func runNativeFixture(t *testing.T, mode string) {
	t.Helper()
	env := newFixtureEnvironment(t)
	defer env.close(t)
	store, err := capture.OpenFileStore(filepath.Join(env.root, "originals"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
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
	clientIntent, clientResult := env.authority(t, fixtureAlice), env.authority(t, fixtureBob)
	serverIntent, serverResult := env.authority(t, fixtureAlice), env.authority(t, fixtureBob)
	hostServices := func(intent, result *g.RegistryAuthority, p *fixturePolicy) toolhost.Services {
		return toolhost.Services{IntentAuthority: intent, ResultAuthority: result, Policy: p, Signer: fixtureResultSigner{env}, Clock: env.clock}
	}
	serverPath := filepath.Join(env.root, "server-ledger")
	server, err := toolhost.Open(serverPath, true, fixtureBob, hostServices(serverIntent, serverResult, serverPolicy), bindings, fixtureBounds())
	if err != nil {
		t.Fatal(err)
	}
	client, err := toolhost.Open(filepath.Join(env.root, "client-ledger"), true, fixtureBob, hostServices(clientIntent, clientResult, policy), bindings, fixtureBounds())
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
	issuer, err := request.NewIntentIssuer(context.Background(), g.IssuerServices{Client: g.ClientServices{IntentAuthority: clientIntent, ResultAuthority: clientResult, Policy: policy, Clock: env.clock, Sender: fixtureNoSender{}, ExpectedIssuer: fixtureAlice, ExpectedRecipient: fixtureBob}, Policy: policy, Signer: signer, Measurement: loaded, KeyID: fixtureAlice + "#signing-1"})
	if err != nil {
		t.Fatal(err)
	}
	defer issuer.Retire()
	approved, err := issuer.Authorize(context.Background(), g.IntentProposal{Tool: "sum", Arguments: []byte(`{"a":2,"b":3}`), LifetimeSeconds: 300})
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
		if e := request.OpenMCPClient(callCtx, c, journal, intent, g.MCPClientServices{IntentAuthority: clientIntent, ResultAuthority: clientResult, Policy: policy, Clock: env.clock}); e != nil {
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
	if mode == "allowed" || mode == "terminal" {
		if delivery == nil || delivery.Status() != "completed" || !delivery.FirstTerminal() || !bytes.Equal(delivery.Output(), []byte(`{"sum":5}`)) || loaded.calls.Load() != 1 {
			t.Fatalf("verified delivery/effect: %+v calls=%d", delivery, loaded.calls.Load())
		}
		if args := <-loaded.seen; !bytes.Equal(args, []byte(`{"a":2,"b":3}`)) {
			t.Fatal("arguments changed")
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
	if _, e := toolhost.Open(serverPath, true, fixtureBob, hostServices(serverIntent, serverResult, serverPolicy), bindings, fixtureBounds()); e == nil {
		t.Fatal("existing ledger recreated")
	}
	recovered, e := toolhost.Open(serverPath, false, fixtureBob, hostServices(serverIntent, serverResult, serverPolicy), bindings, fixtureBounds())
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
	if mode != "allowed" && mode != "terminal" {
		expected = "UNKNOWN"
	}
	if last["state"] != expected {
		t.Fatalf("recovery changed terminal: %v", last["state"])
	}
	if loaded.calls.Load() > 1 {
		t.Fatal("recovery executed again")
	}
}
