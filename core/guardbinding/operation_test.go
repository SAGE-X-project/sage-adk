// SPDX-License-Identifier: LGPL-3.0-or-later
package guardbinding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/capture"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

const alice = "did:sage:web:agent.example:alice"
const bob = "did:sage:web:agent.example:bob"

func testJSON(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func testHash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

type testStore struct {
	data   [][]byte
	onLoad func()
}

func (s *testStore) Create(_ context.Context, _ string, b [][]byte) error { s.data = b; return nil }
func (s *testStore) Load(context.Context, string) ([][]byte, error) {
	if s.onLoad != nil {
		s.onLoad()
	}
	return s.data, nil
}

type testInstance struct {
	fail, panicCheck, panicExecute, errorExecute, cancelExecute bool
	onCheck                                                     func()
	effects                                                     atomic.Int64
	started, release                                            chan struct{}
}

func (i *testInstance) Check(context.Context, string, string, string) error {
	if i.onCheck != nil {
		i.onCheck()
	}
	if i.panicCheck {
		panic("fixture")
	}
	if i.fail {
		return errors.New("fixture")
	}
	return nil
}
func (i *testInstance) Execute(ctx context.Context, b []byte) ([]byte, error) {
	i.effects.Add(1)
	if i.panicExecute {
		panic("inert fixture")
	}
	if i.errorExecute {
		return nil, errors.New("inert fixture")
	}
	if i.cancelExecute {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if i.started != nil {
		close(i.started)
		select {
		case <-i.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	b[0] = 'x'
	return []byte(`{"sum":5}`), nil
}

type testFactory struct {
	instance                *testInstance
	fail, panicLoad, mutate bool
}

func (f *testFactory) Load(_ context.Context, s *Snapshot) (Instance, error) {
	if f.panicLoad {
		panic("fixture")
	}
	if f.fail {
		return nil, errors.New("fixture")
	}
	if f.mutate {
		b := s.Artifacts()
		b[0].Bytes[0] = 'x'
		p := s.Policy()
		p[0] = 'x'
		m := s.Manifest()
		m[0] = 'x'
	}
	return f.instance, nil
}

type testEnvironment struct {
	request *capture.Request
	store   *testStore
	config  Config
	factory *testFactory
}

func environment(t *testing.T) *testEnvironment {
	t.Helper()
	store := &testStore{}
	h, e := capture.NewHost(store)
	if e != nil {
		t.Fatal(e)
	}
	r, e := h.Capture(context.Background(), [][]byte{[]byte("sum two and three")})
	if e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	files := map[string][]byte{"evaluator.fixture": []byte("inert evaluator bytes"), "tool.fixture": []byte("inert tool bytes"), "rules.json": testJSON(map[string]any{"version": "0.10.0", "recipient": bob, "keyid": alice + "#signing-1", "tool": "sum", "arguments": map[string]int{"a": 2, "b": 3}, "max_lifetime": 300})}
	for name, b := range files {
		if e = os.WriteFile(filepath.Join(root, name), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	manifest := func(names ...string) []byte {
		rows := []any{}
		for _, n := range names {
			rows = append(rows, map[string]any{"path": n, "sha256": testHash(files[n])})
		}
		return testJSON(map[string]any{"version": "0.10.0", "files": rows})
	}
	factory := &testFactory{instance: &testInstance{}}
	policy := testJSON(map[string]any{"version": "0.10.0", "issuer": alice, "epoch": "00000000-0000-4000-8000-000000000001", "engine": Engine, "artifacts": json.RawMessage(manifest("evaluator.fixture", "rules.json"))})
	return &testEnvironment{r, store, Config{Directory: root, Policy: policy, Manifest: manifest("tool.fixture"), Limits: Limits{FileBytes: 1024, TotalBytes: 4096}, Factory: factory}, factory}
}
func (e *testEnvironment) open(t *testing.T) *Operation {
	t.Helper()
	o, err := Open(context.Background(), e.request, e.config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := o.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return o
}
func unsigned(o *Operation, r *capture.Request) []byte {
	return testJSON(map[string]any{"version": "0.10.0", "profile": "sage-execution-guard", "issuer": alice, "recipient": bob, "keyid": alice + "#signing-1", "alg": "ed25519", "request_id": r.ID(), "original_digest": r.Digest(), "policy_digest": o.policyDigest, "manifest_digest": o.manifestDigest, "tool": "sum", "arguments": json.RawMessage(`{"a":2,"b":3}`), "parent_call_id": nil, "call_id": "00000000-0000-4000-8000-000000000002", "created": 100, "expires": 400, "nonce": "AAAAAAAAAAAAAAAAAAAAAA"})
}
func TestPolicyAndPinnedInstance(t *testing.T) {
	e := environment(t)
	e.factory.mutate = true
	o := e.open(t)
	ctx := context.Background()
	if o.Recipient() != bob || o.KeyID() != alice+"#signing-1" {
		t.Fatal("identity changed")
	}
	digest, p, m, err := o.Bindings(ctx, alice, e.request.ID())
	if err != nil || digest != e.request.Digest() {
		t.Fatal(err)
	}
	p[0] = 'x'
	m[0] = 'x'
	e.config.Policy[0] = 'x'
	e.config.Manifest[0] = 'x'
	raw, _ := g.Canonicalize(unsigned(o, e.request))
	if err = o.ApproveIntent(ctx, raw); err != nil {
		t.Fatal(err)
	}
	binding, err := o.Binding(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = binding.Tool.Check(ctx, binding.ManifestDigest, "sum"); err != nil {
		t.Fatal(err)
	}
	args := []byte(`{"a":2,"b":3}`)
	out, err := binding.Tool.Execute(ctx, args)
	if err != nil || !bytes.Equal(out, []byte(`{"sum":5}`)) || args[0] != '{' || e.factory.instance.effects.Load() != 1 {
		t.Fatal("effect binding", err)
	}
	if o.Authorize(ctx, alice, "sum", []byte(`{"a":2,"b":4}`)) == nil {
		t.Fatal("unapproved args")
	}
	if o.Authorize(ctx, bob, "sum", args) == nil || o.Authorize(ctx, alice, "other", args) == nil {
		t.Fatal("unapproved identity/tool")
	}
	if _, _, _, err = o.Bindings(ctx, bob, e.request.ID()); err == nil {
		t.Fatal("issuer mapping")
	}
	if _, _, _, err = o.Bindings(ctx, alice, "missing"); err == nil {
		t.Fatal("request mapping")
	}
}
func TestOpenRefusesUnavailableOrChangedApproval(t *testing.T) {
	for _, mode := range []string{"nil-request", "nil-factory", "typed-nil", "bad-policy", "bad-manifest", "unknown-engine", "wrong-issuer", "empty-manifest", "bad-rules", "extra-rules-field", "wrong-key", "wrong-recipient", "wrong-tool", "array-args", "long-lifetime", "zero-limits", "large-limits", "missing", "symlink", "directory", "changed", "file-limit", "total-limit", "conflicting-path", "loader-error", "loader-panic", "check-error", "check-panic", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			e := environment(t)
			ctx := context.Background()
			rulesPath := filepath.Join(e.config.Directory, "rules.json")
			rewriteRules := func(key string, v any) {
				var m map[string]any
				b, err := os.ReadFile(rulesPath)
				if err != nil {
					t.Fatal(err)
				}
				if json.Unmarshal(b, &m) != nil {
					t.Fatal("rules")
				}
				m[key] = v
				b = testJSON(m)
				if err = os.WriteFile(rulesPath, b, 0600); err != nil {
					t.Fatal(err)
				}
				var p map[string]any
				must(t, json.Unmarshal(e.config.Policy, &p))
				a := p["artifacts"].(map[string]any)
				for _, row := range a["files"].([]any) {
					f := row.(map[string]any)
					if f["path"] == "rules.json" {
						f["sha256"] = testHash(b)
					}
				}
				e.config.Policy = testJSON(p)
			}
			switch mode {
			case "nil-request":
				e.request = nil
			case "nil-factory":
				e.config.Factory = nil
			case "typed-nil":
				var f *testFactory
				e.config.Factory = f
			case "bad-policy":
				e.config.Policy = []byte(`{}`)
			case "bad-manifest":
				e.config.Manifest = []byte(`{}`)
			case "unknown-engine", "wrong-issuer":
				var p map[string]any
				must(t, json.Unmarshal(e.config.Policy, &p))
				if mode == "unknown-engine" {
					p["engine"] = "unknown/1"
				} else {
					p["issuer"] = "invalid"
				}
				e.config.Policy = testJSON(p)
			case "empty-manifest":
				e.config.Manifest = []byte(`{"version":"0.10.0","files":[]}`)
			case "bad-rules":
				must(t, os.WriteFile(rulesPath, []byte(`{`), 0600))
			case "extra-rules-field":
				rewriteRules("extra", true)
			case "wrong-key":
				rewriteRules("keyid", bob+"#signing-1")
			case "wrong-recipient":
				rewriteRules("recipient", "invalid")
			case "wrong-tool":
				rewriteRules("tool", "sage_secure_call")
			case "array-args":
				rewriteRules("arguments", []int{})
			case "long-lifetime":
				rewriteRules("max_lifetime", 301)
			case "zero-limits":
				e.config.Limits = Limits{}
			case "large-limits":
				e.config.Limits.FileBytes = 1 << 40
			case "missing":
				must(t, os.Remove(filepath.Join(e.config.Directory, "tool.fixture")))
			case "symlink":
				path := filepath.Join(e.config.Directory, "tool.fixture")
				must(t, os.Remove(path))
				must(t, os.Symlink("evaluator.fixture", path))
			case "directory":
				path := filepath.Join(e.config.Directory, "tool.fixture")
				must(t, os.Remove(path))
				must(t, os.Mkdir(path, 0700))
			case "changed":
				must(t, os.WriteFile(filepath.Join(e.config.Directory, "tool.fixture"), []byte("changed"), 0600))
			case "file-limit":
				e.config.Limits.FileBytes = 1
			case "total-limit":
				e.config.Limits.TotalBytes = 1
			case "conflicting-path":
				var m map[string]any
				must(t, json.Unmarshal(e.config.Manifest, &m))
				m["files"] = []any{map[string]any{"path": "rules.json", "sha256": testHash([]byte("different"))}}
				e.config.Manifest = testJSON(m)
			case "loader-error":
				e.factory.fail = true
			case "loader-panic":
				e.factory.panicLoad = true
			case "check-error":
				e.factory.instance.fail = true
			case "check-panic":
				e.factory.instance.panicCheck = true
			case "canceled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			if o, err := Open(ctx, e.request, e.config); err == nil || o != nil {
				if o != nil {
					must(t, o.Close(context.Background()))
				}
				t.Fatal("accepted invalid config")
			}
		})
	}
}
func TestLaterDriftAndClosedIntentDenial(t *testing.T) {
	for _, mode := range []string{"capture", "file", "instance", "instance-panic", "canceled", "recipient", "key", "parent", "args", "extra", "noncanonical", "call", "nonce", "lifetime"} {
		t.Run(mode, func(t *testing.T) {
			e := environment(t)
			o := e.open(t)
			ctx := context.Background()
			raw, _ := g.Canonicalize(unsigned(o, e.request))
			var body map[string]any
			must(t, json.Unmarshal(raw, &body))
			switch mode {
			case "capture":
				e.store.data[0] = []byte("changed")
			case "file":
				must(t, os.WriteFile(filepath.Join(e.config.Directory, "tool.fixture"), []byte("changed"), 0600))
			case "instance":
				e.factory.instance.fail = true
			case "instance-panic":
				e.factory.instance.panicCheck = true
			case "canceled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			case "recipient":
				body["recipient"] = alice
			case "key":
				body["keyid"] = alice + "#other"
			case "parent":
				body["parent_call_id"] = "00000000-0000-4000-8000-000000000002"
			case "args":
				body["arguments"] = map[string]int{"a": 2, "b": 4}
			case "extra":
				body["extra"] = true
			case "noncanonical":
				raw = append([]byte(" "), raw...)
			case "call":
				body["call_id"] = "invalid"
			case "nonce":
				body["nonce"] = "invalid"
			case "lifetime":
				body["expires"] = 401
			}
			if mode != "noncanonical" {
				raw, _ = g.Canonicalize(testJSON(body))
			}
			if o.ApproveIntent(ctx, raw) == nil {
				t.Fatal("invalid approval")
			}
			if e.factory.instance.effects.Load() != 0 {
				t.Fatal("denial effect")
			}
		})
	}
}
func TestRetirementWaitsForBoundedEffect(t *testing.T) {
	e := environment(t)
	i := e.factory.instance
	i.started = make(chan struct{})
	i.release = make(chan struct{})
	o := e.open(t)
	b, err := o.Binding(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, e := b.Tool.Execute(context.Background(), []byte(`{"a":2,"b":3}`)); done <- e }()
	select {
	case <-i.started:
	case <-time.After(time.Second):
		t.Fatal("effect not started")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if o.Close(ctx) == nil {
		t.Fatal("closed while effect held ownership")
	}
	if o.Authorize(context.Background(), alice, "sum", []byte(`{"a":2,"b":3}`)) == nil {
		t.Fatal("admission after closing")
	}
	close(i.release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if err = o.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = b.Tool.Execute(context.Background(), []byte(`{"a":2,"b":3}`)); err == nil || i.effects.Load() != 1 {
		t.Fatal("execution after retirement")
	}
}

func TestUnavailableBindingsAndFinalExecutionRefuse(t *testing.T) {
	for _, method := range []string{"bindings", "binding", "check", "authorize", "execute"} {
		for _, mode := range []string{"panic", "file", "canceled"} {
			t.Run(method+"/"+mode, func(t *testing.T) {
				e := environment(t)
				o := e.open(t)
				ctx := context.Background()
				b, err := o.Binding(ctx)
				if err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "panic":
					e.factory.instance.panicCheck = true
				case "file":
					if err = os.WriteFile(filepath.Join(e.config.Directory, "tool.fixture"), []byte("changed"), 0600); err != nil {
						t.Fatal(err)
					}
				case "canceled":
					c, cancel := context.WithCancel(ctx)
					cancel()
					ctx = c
				}
				switch method {
				case "bindings":
					_, _, _, err = o.Bindings(ctx, alice, e.request.ID())
				case "binding":
					_, err = o.Binding(ctx)
				case "check":
					err = o.Check(ctx, b.ManifestDigest, "sum")
				case "authorize":
					err = o.Authorize(ctx, alice, "sum", []byte(`{"a":2,"b":3}`))
				case "execute":
					_, err = b.Tool.Execute(ctx, []byte(`{"a":2,"b":3}`))
				}
				if err == nil || e.factory.instance.effects.Load() != 0 {
					t.Fatal("missing denial before effect")
				}
			})
		}
	}
	for _, mode := range []string{"panic", "error", "cancel"} {
		t.Run("effect-"+mode, func(t *testing.T) {
			e := environment(t)
			o := e.open(t)
			b, err := o.Binding(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			switch mode {
			case "panic":
				e.factory.instance.panicExecute = true
			case "error":
				e.factory.instance.errorExecute = true
			case "cancel":
				e.factory.instance.cancelExecute = true
				c, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
				defer cancel()
				ctx = c
			}
			out, err := b.Tool.Execute(ctx, []byte(`{"a":2,"b":3}`))
			if err == nil || len(out) != 0 || e.factory.instance.effects.Load() != 1 {
				t.Fatal("uncertain effect exposed output")
			}
		})
	}
	e := environment(t)
	o := e.open(t)
	b, err := o.Binding(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if o.Check(context.Background(), "wrong", "sum") == nil || o.Check(context.Background(), b.ManifestDigest, "other") == nil {
		t.Fatal("unchecked measurement identity")
	}
	if _, err = b.Tool.Execute(context.Background(), []byte(`{"a":2,"b":4}`)); err == nil || e.factory.instance.effects.Load() != 0 {
		t.Fatal("unapproved effect")
	}
}
func TestCanceledGateWaitAndInvalidHandles(t *testing.T) {
	e := environment(t)
	o := e.open(t)
	<-o.gate
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if o.Check(ctx, o.manifestDigest, "sum") == nil {
		t.Fatal("uncanceled wait")
	}
	o.leave()
	if o.Check(context.Background(), o.manifestDigest, "sum") != nil {
		t.Fatal("canceled wait poisoned healthy owner")
	}
	var missing *Operation
	if missing.Recipient() != "" || missing.KeyID() != "" || missing.Close(context.Background()) != nil {
		t.Fatal("nil handle")
	}
	var snapshot *Snapshot
	if snapshot.Artifacts() != nil || snapshot.Policy() != nil || snapshot.Manifest() != nil {
		t.Fatal("nil snapshot")
	}
	zero := &Operation{}
	if _, err := zero.Binding(context.Background()); err == nil {
		t.Fatal("zero operation")
	}
	if zero.Close(context.Background()) == nil || o.Close(nil) == nil { //nolint:staticcheck // Exercise the documented nil-context refusal.
		t.Fatal("invalid close")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCanceledCaptureReadDoesNotRetireHealthyOperation(t *testing.T) {
	e := environment(t)
	o := e.open(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.store.onLoad = cancel
	if o.Authorize(ctx, alice, "sum", []byte(`{"a":2,"b":3}`)) == nil {
		t.Fatal("canceled capture read")
	}
	e.store.onLoad = nil
	if o.Authorize(context.Background(), alice, "sum", []byte(`{"a":2,"b":3}`)) != nil {
		t.Fatal("cancellation retired healthy operation")
	}
}

func TestAttestationCallbackCannotHideLaterCaptureOrFileDrift(t *testing.T) {
	for _, which := range []string{"capture", "file"} {
		t.Run(which, func(t *testing.T) {
			e := environment(t)
			o := e.open(t)
			b, err := o.Binding(context.Background())
			must(t, err)
			e.factory.instance.onCheck = func() {
				if which == "capture" {
					e.store.data[0] = []byte("changed")
				} else {
					must(t, os.WriteFile(filepath.Join(e.config.Directory, "tool.fixture"), []byte("changed"), 0600))
				}
			}
			if _, err = b.Tool.Execute(context.Background(), []byte(`{"a":2,"b":3}`)); err == nil || e.factory.instance.effects.Load() != 0 {
				t.Fatal("drift during attestation reached execution")
			}
		})
	}
}
