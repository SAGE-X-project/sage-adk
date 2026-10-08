//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardhost_test

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
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

	"github.com/sage-x-project/sage-adk/core/guardapproval"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	calc "github.com/sage-x-project/sage-adk/core/guardcalculator"
	"github.com/sage-x-project/sage-adk/core/guardhost"
	"github.com/sage-x-project/sage-adk/core/guardsigner"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

const alice = "did:sage:web:agent.example:alice"
const bob = "did:sage:web:agent.example:bob"

// testClock follows real elapsed time plus an offset, so tests can skip the
// six-minute replay quarantine while polling still sees time pass.
type testClock struct {
	mono    atomic.Int64
	started time.Time
}

func (c *testClock) now() int64 { return c.mono.Load() + time.Since(c.started).Milliseconds() }
func (c *testClock) Now() (r.Stamp, error) {
	m := c.now()
	return r.Stamp{MonoMS: m, Unix: 100 + m/1000}, nil
}
func (c *testClock) Sample(context.Context) (int64, int64, error) {
	m := c.now()
	return 100000 + m, m, nil
}

// syntheticSource is a test-only Registry. Its Ready/Validated/Finalized flags
// are asserted, not observed; it is not an authoritative Source.
type syntheticSource struct {
	clock      *testClock
	alice, bob ed25519.PublicKey
	kem        []byte
	revoked    atomic.Bool
}

func (s *syntheticSource) Read(_ context.Context, did string) (r.Snapshot, error) {
	key := s.alice
	if did == bob {
		key = s.bob
	} else if did != alice {
		return r.Snapshot{}, errors.New("unknown")
	}
	state := "active"
	if s.revoked.Load() && did == bob {
		state = "deactivated"
	}
	keys := []r.Key{{Name: "signing-1", Alg: "ed25519", Material: hex.EncodeToString(key), State: "accepted"}}
	if did == bob {
		keys = append([]r.Key{{Name: "kem-1", Alg: "x25519", Material: hex.EncodeToString(s.kem), State: "accepted"}}, keys...)
	}
	stamp, _ := s.clock.Now()
	return r.Snapshot{Source: "synthetic-test-fixture", Registry: "web:agent.example", Network: "local", DID: did, Version: "1", State: state, Digest: strings.Repeat("a", 64), Ready: true, Validated: true, Finalized: true, AcquiredMS: stamp.MonoMS, Keys: keys}, nil
}

// syntheticMeasurement accepts any snapshot. It is not a loaded-code or
// isolation measurement and must never be used by a deployment.
type syntheticMeasurement struct{}

func (syntheticMeasurement) Check(ctx context.Context, s *b.Snapshot) error {
	if ctx.Err() != nil || s == nil {
		return errors.New("unavailable")
	}
	return nil
}

func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func mustJSON(v any) []byte {
	out, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return out
}

type world struct {
	clock                  *testClock
	source                 *syntheticSource
	aliceKey, bobKey       ed25519.PrivateKey
	kemPrivate             []byte
	operator               ed25519.PrivateKey
	policy, manifest       []byte
	artifacts              map[string][]byte
	aliceSigner, bobSigner *guardsigner.Client
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{clock: &testClock{started: time.Now()}}
	var err error
	if _, w.aliceKey, err = ed25519.GenerateKey(rand.Reader); err != nil {
		t.Fatal(err)
	}
	if _, w.bobKey, err = ed25519.GenerateKey(rand.Reader); err != nil {
		t.Fatal(err)
	}
	if _, w.operator, err = ed25519.GenerateKey(rand.Reader); err != nil {
		t.Fatal(err)
	}
	kem, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	w.kemPrivate = kem.Bytes()
	w.source = &syntheticSource{clock: w.clock, alice: w.aliceKey.Public().(ed25519.PublicKey), bob: w.bobKey.Public().(ed25519.PublicKey), kem: kem.PublicKey().Bytes()}
	args := json.RawMessage(`{"a":2,"b":3,"operation":"add"}`)
	rule := mustJSON(map[string]any{"version": "0.10.0", "recipient": bob, "keyid": alice + "#signing-1", "tool": "calculator", "arguments": args, "max_lifetime": 300})
	configuration := mustJSON(map[string]any{"version": "0.10.0", "tool": "calculator", "image_path": "image.fixture", "operations": []string{"add"}, "absolute_operand_limit": 100})
	w.artifacts = map[string][]byte{"image.fixture": []byte("synthetic runtime image observation"), "rules.json": rule, calc.ConfigurationPath: configuration}
	files := func(names ...string) []byte {
		rows := []any{}
		for _, n := range names {
			rows = append(rows, map[string]any{"path": n, "sha256": hash(w.artifacts[n])})
		}
		return mustJSON(map[string]any{"version": "0.10.0", "files": rows})
	}
	w.policy = mustJSON(map[string]any{"version": "0.10.0", "issuer": alice, "epoch": "00000000-0000-4000-8000-000000000001", "engine": b.Engine, "artifacts": json.RawMessage(files("image.fixture", "rules.json"))})
	w.manifest = files(calc.ConfigurationPath, "image.fixture")
	dir, err := os.MkdirTemp("/tmp", "ghost")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	w.aliceSigner = serveSigner(t, filepath.Join(dir, "a"), w.aliceKey, "intent", "result", "transport")
	w.bobSigner = serveSigner(t, filepath.Join(dir, "b"), w.bobKey, "result", "transport")
	return w
}

func serveSigner(t *testing.T, path string, key ed25519.PrivateKey, roles ...string) *guardsigner.Client {
	t.Helper()
	s, err := guardsigner.NewServer(guardsigner.Config{Key: key, Roles: roles, AllowedUIDs: []uint32{uint32(os.Getuid())}, Timeout: 2 * time.Second, MaxConnections: 8})
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, l) }()
	t.Cleanup(func() { cancel(); <-done; s.Close() })
	c, err := guardsigner.NewClient(path, uint32(os.Getuid()), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// hostDir creates one host's protected state directory and its own artifact copy.
func (w *world) hostDir(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	state, artifacts := filepath.Join(root, "state"), filepath.Join(root, "artifacts")
	for _, d := range []string{state, artifacts} {
		if err := os.Mkdir(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range w.artifacts {
		if err := os.WriteFile(filepath.Join(artifacts, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return state, artifacts
}

func (w *world) approved(t *testing.T, artifacts string, sequence uint64) guardhost.Approved {
	t.Helper()
	factory, err := calc.NewFactory(calc.Config{Measurement: syntheticMeasurement{}, MaxInstances: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = factory.Close(context.Background()) })
	approval, err := guardapproval.Sign(w.operator, w.policy, w.manifest, sequence)
	if err != nil {
		t.Fatal(err)
	}
	return guardhost.Approved{Operation: b.Config{Directory: artifacts, Policy: w.policy, Manifest: w.manifest, Limits: b.Limits{FileBytes: 4096, TotalBytes: 16384}, Factory: factory}, Approval: approval, Approvers: []ed25519.PublicKey{w.operator.Public().(ed25519.PublicKey)}}
}

func bounds() g.MCPHostBounds {
	return g.MCPHostBounds{Capacity: 2, Preparations: 2, Clients: 2, Owners: 4, Workers: 1, Request: 20 * time.Second, Claim: 10 * time.Second, Worker: 5 * time.Second, Client: 20 * time.Second, Tick: time.Millisecond}
}

func (w *world) env(dir string) guardhost.Environment {
	return guardhost.Environment{Dir: dir, Create: true, Registry: r.Config{Source: "synthetic-test-fixture", Registry: "web:agent.example", Network: "local"}, Source: w.source, Clock: w.clock}
}

func (w *world) receiverConfig(t *testing.T) guardhost.ReceiverConfig {
	state, artifacts := w.hostDir(t)
	return guardhost.ReceiverConfig{Environment: w.env(state), Identity: guardhost.Identity{DID: bob, KeyID: bob + "#signing-1", Transport: w.bobSigner, Result: w.bobSigner}, KEM: w.kemPrivate, Issuer: alice, IssuerKey: alice + "#signing-1", Approved: w.approved(t, artifacts, 1), Bounds: bounds(), Connection: g.MCPConnectionConfig{Role: g.MCPResponder, Name: "adk guarded receiver", Version: "1", TTLSeconds: 300, Timeout: 3 * time.Second}}
}

func (w *world) callerConfig(t *testing.T) guardhost.CallerConfig {
	state, artifacts := w.hostDir(t)
	return guardhost.CallerConfig{Environment: w.env(state), Identity: guardhost.Identity{DID: alice, KeyID: alice + "#signing-1", Transport: w.aliceSigner, Result: w.aliceSigner}, Intent: w.aliceSigner, Recipient: bob, RecipientKey: bob + "#signing-1", Approved: w.approved(t, artifacts, 1), Bounds: bounds(), Connection: g.MCPConnectionConfig{Role: g.MCPInitiator, Recipient: bob, RecipientKey: bob + "#signing-1", Name: "adk guarded caller", Version: "1", TTLSeconds: 300, Timeout: 3 * time.Second}}
}

func proposal() g.IntentProposal {
	return g.IntentProposal{Tool: "calculator", Arguments: []byte(`{"a":2,"b":3,"operation":"add"}`), LifetimeSeconds: 300}
}

// Two hosts with separate state, approvals, artifact copies and signer
// processes complete one approved 2+3 call. The receiver never sees the
// caller's original input.
func TestSeparateHostsCompleteApprovedCall(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	receiver, err := guardhost.OpenReceiver(ctx, w.receiverConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	caller, err := guardhost.OpenCaller(ctx, w.callerConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	w.clock.mono.Add(361000) // replay quarantine of both new journals
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveCtx, stop := context.WithCancel(ctx)
	served := make(chan error, 1)
	go func() { served <- receiver.Serve(serveCtx, listener, 1) }()
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	output, err := caller.Call(callCtx, [][]byte{[]byte("please add two and three")}, proposal(), conn)
	stop()
	if e := <-served; e != nil && !errors.Is(e, context.Canceled) {
		t.Errorf("serve: %v", e)
	}
	if err != nil || string(output) != `{"output":5,"success":true}` {
		t.Fatalf("call: %s %v", output, err)
	}
	if e := receiver.Close(ctx); e != nil {
		t.Error(e)
	}
	if e := caller.Close(); e != nil {
		t.Error(e)
	}
}

// A call during the replay quarantine and a call with arguments outside the
// approved rules both fail closed without a result.
func TestCallerRefusesBeforeQuarantineAndOutsidePolicy(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	receiver, err := guardhost.OpenReceiver(ctx, w.receiverConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = receiver.Close(ctx) }()
	caller, err := guardhost.OpenCaller(ctx, w.callerConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caller.Close() }()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveCtx, stop := context.WithCancel(ctx)
	defer stop()
	go func() { _ = receiver.Serve(serveCtx, listener, 1) }()
	call := func(p g.IntentProposal) ([]byte, error) {
		conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return caller.Call(callCtx, [][]byte{[]byte("add")}, p, conn)
	}
	if out, err := call(proposal()); err == nil || out != nil {
		t.Fatal("call accepted during replay quarantine")
	}
	w.clock.mono.Add(361000)
	other := proposal()
	other.Arguments = []byte(`{"a":2,"b":4,"operation":"add"}`)
	if out, err := call(other); err == nil || out != nil {
		t.Fatal("arguments outside the approved rules were called")
	}
	// The same hosts still complete the approved call, so both refusals came
	// from the quarantine and the policy rather than a broken environment.
	if out, err := call(proposal()); err != nil || string(out) != `{"output":5,"success":true}` {
		t.Fatalf("approved call after refusals: %s %v", out, err)
	}
}

func TestHostAssemblyRefusesMissingPorts(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	for name, change := range map[string]func(*guardhost.ReceiverConfig){
		"no-source":    func(c *guardhost.ReceiverConfig) { c.Source = nil },
		"no-clock":     func(c *guardhost.ReceiverConfig) { c.Clock = nil },
		"no-custody":   func(c *guardhost.ReceiverConfig) { c.Transport = nil },
		"no-kem":       func(c *guardhost.ReceiverConfig) { c.KEM = nil },
		"relative-dir": func(c *guardhost.ReceiverConfig) { c.Dir = "state" },
		"wrong-role":   func(c *guardhost.ReceiverConfig) { c.Connection.Role = g.MCPInitiator },
		"unsigned":     func(c *guardhost.ReceiverConfig) { c.Approval = nil },
		"unpinned": func(c *guardhost.ReceiverConfig) {
			c.Approvers = []ed25519.PublicKey{w.aliceKey.Public().(ed25519.PublicKey)}
		},
		"other-policy": func(c *guardhost.ReceiverConfig) {
			c.Operation.Policy = []byte(strings.Replace(string(w.policy), "000000000001", "000000000002", 1))
		},
		"wrong-result-key": func(c *guardhost.ReceiverConfig) { c.Result = w.aliceSigner },
	} {
		t.Run(name, func(t *testing.T) {
			c := w.receiverConfig(t)
			change(&c)
			if r, err := guardhost.OpenReceiver(ctx, c); err == nil || r != nil {
				t.Fatal("receiver opened")
			}
		})
	}
	open := w.receiverConfig(t)
	if err := os.Chmod(open.Dir, 0755); err != nil {
		t.Fatal(err)
	}
	if r, err := guardhost.OpenReceiver(ctx, open); err == nil || r != nil {
		t.Fatal("shared state directory accepted")
	}
	for name, change := range map[string]func(*guardhost.CallerConfig){
		"no-intent-custody": func(c *guardhost.CallerConfig) { c.Intent = nil },
		"peer-mismatch":     func(c *guardhost.CallerConfig) { c.Connection.Recipient = alice },
		"wrong-intent-key":  func(c *guardhost.CallerConfig) { c.Intent = w.bobSigner },
	} {
		t.Run(name, func(t *testing.T) {
			c := w.callerConfig(t)
			change(&c)
			if cl, err := guardhost.OpenCaller(ctx, c); err == nil || cl != nil {
				t.Fatal("caller opened")
			}
		})
	}
}
