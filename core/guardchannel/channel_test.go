// SPDX-License-Identifier: LGPL-3.0-or-later
package guardchannel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/capture"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	image "github.com/sage-x-project/sage-adk/core/guardimage"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

const alice = "did:sage:web:agent.example:alice"
const bob = "did:sage:web:agent.example:bob"

func encoded(v any) []byte {
	raw, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return raw
}
func hashed(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func fixtureConfig(raw []byte) Config {
	rule := encoded(map[string]any{"version": "0.10.0", "recipient": bob, "keyid": alice + "#signing-1", "tool": "calculator", "arguments": map[string]any{"a": 2, "b": 3, "operation": "add"}, "max_lifetime": 300})
	config := encoded(map[string]any{"version": "0.10.0", "tool": "calculator", "image_path": "image.fixture", "operations": []string{"add"}, "absolute_operand_limit": 100})
	files := map[string][]byte{"image.fixture": raw, "rules.json": rule, "calculator.json": config}
	descriptor := func(names ...string) []byte {
		entries := []any{}
		for _, name := range names {
			entries = append(entries, map[string]any{"path": name, "sha256": hashed(files[name])})
		}
		return encoded(map[string]any{"version": "0.10.0", "files": entries})
	}
	policy := encoded(map[string]any{"version": "0.10.0", "issuer": alice, "epoch": "00000000-0000-4000-8000-000000000001", "engine": b.Engine, "artifacts": json.RawMessage(descriptor("image.fixture", "rules.json"))})
	return Config{Image: image.Config{Image: raw, SHA256: hashed(raw), Limit: image.MaxImageBytes}, ImagePath: "image.fixture", Policy: policy, Manifest: descriptor("calculator.json", "image.fixture"), Artifacts: []g.Artifact{{Path: "calculator.json", Bytes: config}, {Path: "image.fixture", Bytes: raw}, {Path: "rules.json", Bytes: rule}}, Timeout: time.Second, MaxChecks: 1000}
}

type fixtureFactory struct{ s *b.Snapshot }
type fixtureInstance struct{}

func (f *fixtureFactory) Load(_ context.Context, s *b.Snapshot) (b.Instance, error) {
	f.s = s
	return fixtureInstance{}, nil
}
func (fixtureInstance) Check(context.Context, string, string, string) error { return nil }
func (fixtureInstance) Execute(context.Context, []byte) ([]byte, error)     { return []byte(`{}`), nil }
func snapshot(t *testing.T, c Config) *b.Snapshot {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "artifacts")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}
	for _, a := range c.Artifacts {
		if e := os.WriteFile(filepath.Join(dir, a.Path), a.Bytes, 0600); e != nil {
			t.Fatal(e)
		}
	}
	store, e := capture.OpenFileStore(filepath.Join(root, "original"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := store.Close(); e != nil {
			t.Error(e)
		}
	})
	host, e := capture.NewHost(store)
	if e != nil {
		t.Fatal(e)
	}
	request, e := host.Capture(context.Background(), [][]byte{[]byte("fixed harmless arithmetic")})
	if e != nil {
		t.Fatal(e)
	}
	factory := &fixtureFactory{}
	op, e := b.Open(context.Background(), request, b.Config{Directory: dir, Policy: c.Policy, Manifest: c.Manifest, Limits: b.Limits{FileBytes: image.MaxImageBytes, TotalBytes: 256 << 20}, Factory: factory})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := op.Close(context.Background()); e != nil {
			t.Error(e)
		}
	})
	return factory.s
}

// Controlled providers below are unit-only assertions, not runtime attestation.
type localFixture struct {
	calls        atomic.Int64
	fail, panics bool
	callback     func(context.Context)
}

func (l *localFixture) Check(ctx context.Context, _ *b.Snapshot) error {
	l.calls.Add(1)
	if l.callback != nil {
		l.callback(ctx)
	}
	if l.panics {
		panic("controlled unit provider")
	}
	if l.fail {
		return b.ErrDenied
	}
	return nil
}

type endpointFixture struct {
	hello                 *record
	last                  record
	failSend, failReceive bool
	mutate                func(*record)
	closes                int
	failClose             bool
}

func (f *endpointFixture) send(_ context.Context, r record) error {
	f.last = r
	if f.failSend {
		return b.ErrDenied
	}
	return nil
}
func (f *endpointFixture) receive(context.Context) (record, error) {
	r := f.last
	r.kind = 'A'
	if f.hello != nil {
		r = *f.hello
	}
	if f.mutate != nil {
		f.mutate(&r)
	}
	if f.failReceive {
		return record{}, b.ErrDenied
	}
	return r, nil
}
func (f *endpointFixture) bindPeer(int) error { return nil }
func (f *endpointFixture) close() error {
	f.closes++
	if f.failClose {
		return b.ErrDenied
	}
	return nil
}
func measurement(t *testing.T) (*Measurement, *endpointFixture, *localFixture, *b.Snapshot) {
	t.Helper()
	c := fixtureConfig([]byte("inert unit image"))
	base, e := approved(c)
	if e != nil {
		t.Fatal(e)
	}
	ep := &endpointFixture{}
	local := &localFixture{}
	return &Measurement{endpoint: ep, baseline: base, local: local, timeout: time.Second, gate: gate()}, ep, local, snapshot(t, c)
}
func TestApprovedBaselineAndFreshGeneration(t *testing.T) {
	c := fixtureConfig([]byte("inert unit image"))
	one, e := approved(c)
	if e != nil {
		t.Fatal(e)
	}
	two, e := approved(c)
	if e != nil || one.epoch == two.epoch {
		t.Fatal("reused session generation", e)
	}
	if one.policy != two.policy || one.manifest != two.manifest {
		t.Fatal("baseline unstable")
	}
	variants := []func(*Config){
		func(c *Config) { c.MaxChecks = 0 }, func(c *Config) { c.MaxChecks = 1000001 }, func(c *Config) { c.Timeout = 0 }, func(c *Config) { c.Timeout = 6 * time.Second },
		func(c *Config) { c.ImagePath = "missing" }, func(c *Config) { c.Image.SHA256 = "bad" }, func(c *Config) { c.Policy = []byte(`{}`) }, func(c *Config) { c.Manifest = []byte(`{}`) },
		func(c *Config) { c.Artifacts = nil }, func(c *Config) { c.Artifacts = append(c.Artifacts, c.Artifacts[0]) }, func(c *Config) { c.Artifacts[1].Bytes = []byte("different image") },
		func(c *Config) { c.Artifacts[0].Bytes = []byte("changed configuration") }, func(c *Config) { c.Artifacts[2].Bytes = []byte("changed policy rules") },
		func(c *Config) { c.Manifest = c.Policy }, func(c *Config) { c.Policy = append(c.Policy, 0) },
	}
	for n, change := range variants {
		t.Run(string(rune('A'+n)), func(t *testing.T) {
			bad := fixtureConfig([]byte("inert unit image"))
			change(&bad)
			if _, e := approved(bad); e == nil {
				t.Fatal("approved altered baseline")
			}
			if s, e := Start(context.Background(), bad); e == nil || s != nil {
				t.Fatal("launched refused baseline")
			}
		})
	}
	var noContext context.Context // Deliberate invalid-input boundary.
	if s, e := Start(noContext, c); e == nil || s != nil {
		t.Fatal("nil context")
	}
	if runtime.GOOS != "linux" {
		if s, e := Start(context.Background(), c); !errors.Is(e, image.ErrUnsupported) || s != nil {
			t.Fatal("unsupported fallback", e)
		}
	}
	for _, v := range []string{"bad", string(make([]byte, 64)), "AA" + hashed(nil)[2:]} {
		if _, e := digest(v); e == nil {
			t.Fatal("accepted digest")
		}
	}
}
func TestChildChecksFreshSequenceAndExactSnapshot(t *testing.T) {
	m, ep, local, s := measurement(t)
	for n := uint64(1); n <= 3; n++ {
		if e := m.Check(context.Background(), s); e != nil {
			t.Fatal(e)
		}
		if ep.last.sequence != n || ep.last.kind != 'Q' || ep.last.epoch != m.baseline.epoch {
			t.Fatal("connection generation/sequence")
		}
	}
	if local.calls.Load() != 3 {
		t.Fatal("skipped local assurance")
	}
	if e := m.Close(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e := m.Check(context.Background(), s); e == nil {
		t.Fatal("retired capability")
	}
	if e := m.Close(context.Background()); e != nil || ep.closes != 1 {
		t.Fatal("repeated endpoint cleanup")
	}
}
func TestUnitOnlyRefusalScenariosPermanentlyRetire(t *testing.T) {
	changes := []func(*Measurement, *endpointFixture, *localFixture){
		func(_ *Measurement, e *endpointFixture, _ *localFixture) { e.failSend = true }, func(_ *Measurement, e *endpointFixture, _ *localFixture) { e.failReceive = true },
		func(_ *Measurement, e *endpointFixture, _ *localFixture) { e.mutate = func(r *record) { r.sequence-- } },
		func(_ *Measurement, e *endpointFixture, _ *localFixture) {
			e.mutate = func(r *record) { r.epoch[0] ^= 1 }
		},
		func(_ *Measurement, e *endpointFixture, _ *localFixture) {
			e.mutate = func(r *record) { r.policy[0] ^= 1 }
		},
		func(_ *Measurement, e *endpointFixture, _ *localFixture) {
			e.mutate = func(r *record) { r.manifest[0] ^= 1 }
		},
		func(_ *Measurement, e *endpointFixture, _ *localFixture) { e.mutate = func(r *record) { r.kind = 'Q' } },
		func(_ *Measurement, _ *endpointFixture, l *localFixture) { l.fail = true }, func(_ *Measurement, _ *endpointFixture, l *localFixture) { l.panics = true },
		func(m *Measurement, _ *endpointFixture, _ *localFixture) { m.local = nil }, func(m *Measurement, _ *endpointFixture, _ *localFixture) { m.baseline.policy[0] ^= 1 },
		func(m *Measurement, _ *endpointFixture, _ *localFixture) { m.sequence = 1000000 },
		func(m *Measurement, _ *endpointFixture, l *localFixture) {
			l.callback = func(context.Context) { m.retired.Store(true) }
		},
		func(m *Measurement, _ *endpointFixture, l *localFixture) {
			m.timeout = time.Millisecond
			l.callback = func(ctx context.Context) { <-ctx.Done() }
		},
	}
	for n, change := range changes {
		t.Run(string(rune('A'+n)), func(t *testing.T) {
			m, ep, local, s := measurement(t)
			change(m, ep, local)
			if e := m.Check(context.Background(), s); e == nil || !m.retired.Load() {
				t.Fatal("accepted inconsistent provider/ack")
			}
			ep.failSend = false
			ep.failReceive = false
			ep.mutate = nil
			local.fail = false
			local.panics = false
			if e := m.Check(context.Background(), s); e == nil {
				t.Fatal("reused failed session")
			}
		})
	}
	m, _, _, _ := measurement(t)
	if m.Check(context.Background(), nil) == nil || !m.retired.Load() {
		t.Fatal("nil snapshot")
	}
	var nilM *Measurement
	if nilM.Check(context.Background(), nil) == nil || nilM.Close(context.Background()) != nil {
		t.Fatal("nil measurement")
	}
}
func TestCancellationAndCleanupRetry(t *testing.T) {
	m, ep, _, s := measurement(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if m.Check(ctx, s) == nil {
		t.Fatal("cancelled check")
	}
	if m.Close(ctx) == nil || ep.closes != 0 || !m.retired.Load() {
		t.Fatal("lost cleanup ownership")
	}
	ep.failClose = true
	if m.Close(context.Background()) == nil || m.closed {
		t.Fatal("false cleanup success")
	}
	ep.failClose = false
	if m.Close(context.Background()) != nil || !m.closed {
		t.Fatal("cleanup retry")
	}
	m, _, local, s := measurement(t)
	entered, release := make(chan struct{}), make(chan struct{})
	local.callback = func(context.Context) { close(entered); <-release }
	done := make(chan error, 1)
	go func() { done <- m.Check(context.Background(), s) }()
	<-entered
	if m.Close(ctx) == nil {
		t.Fatal("cancelled concurrent cleanup")
	}
	close(release)
	if <-done == nil {
		t.Fatal("accepted response after retirement")
	}
	if m.Close(context.Background()) != nil {
		t.Fatal("final cleanup")
	}
	if (&Measurement{}).Close(context.Background()) == nil {
		t.Fatal("zero capability")
	}
}
func TestClosedRecordAndOpenChildRefusals(t *testing.T) {
	r := record{kind: 'Q', sequence: 3}
	r.epoch[0] = 1
	r.policy[0] = 2
	r.manifest[0] = 3
	if got, e := decode(encode(r)); e != nil || got != r {
		t.Fatal("record roundtrip")
	}
	for _, raw := range [][]byte{nil, encode(r)[:111], append(encode(r), 0)} {
		if _, e := decode(raw); e == nil {
			t.Fatal("open record length")
		}
	}
	raw := encode(r)
	raw[0] = 'X'
	if _, e := decode(raw); e == nil {
		t.Fatal("open kind")
	}
	raw = encode(r)
	raw[1] = 'X'
	if _, e := decode(raw); e == nil {
		t.Fatal("open version")
	}
	var nilLocal *localFixture
	if m, e := OpenChild(context.Background(), nil, nilLocal, time.Second); e == nil || m != nil {
		t.Fatal("missing local assurance")
	}
	file, e := os.CreateTemp(t.TempDir(), "inert")
	if e != nil {
		t.Fatal(e)
	}
	var noContext context.Context // Deliberate invalid-input boundary.
	if m, e := OpenChild(noContext, file, &localFixture{}, time.Second); e == nil || m != nil {
		t.Fatal("nil context")
	}
	if _, e = file.Stat(); e == nil {
		t.Fatal("failed startup retained supplied file")
	}
	var s *Supervisor
	if s.Close(context.Background()) != nil {
		t.Fatal("nil supervisor")
	}
	if (&Supervisor{}).Close(context.Background()) == nil {
		t.Fatal("zero supervisor")
	}
	ep := &endpointFixture{}
	done := make(chan struct{})
	close(done)
	sup := &Supervisor{endpoint: ep, done: done, gate: gate()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if sup.Close(ctx) == nil || ep.closes != 0 {
		t.Fatal("cancelled supervisor cleanup")
	}
	if sup.Close(context.Background()) != nil {
		t.Fatal("supervisor cleanup retry")
	}
	if sup.Close(context.Background()) != nil || ep.closes != 1 {
		t.Fatal("supervisor cleanup retry")
	}
}

func TestBootstrapRefusesMalformedGenerationAndSnapshotUnits(t *testing.T) {
	base, e := approved(fixtureConfig([]byte("inert unit image")))
	if e != nil {
		t.Fatal(e)
	}
	for _, mutation := range []string{"valid", "kind", "epoch", "sequence", "policy", "manifest", "receive"} {
		t.Run(mutation, func(t *testing.T) {
			r := base
			ep := &endpointFixture{hello: &r}
			switch mutation {
			case "kind":
				r.kind = 'A'
			case "epoch":
				r.epoch = [32]byte{}
			case "sequence":
				r.sequence = 1
			case "policy":
				r.policy = [32]byte{}
			case "manifest":
				r.manifest = [32]byte{}
			case "receive":
				ep.failReceive = true
			}
			m, e := establish(context.Background(), ep, &localFixture{}, time.Second)
			if mutation == "valid" {
				if e != nil || m == nil {
					t.Fatal(e)
				}
				if m.Close(context.Background()) != nil {
					t.Fatal("bootstrap cleanup")
				}
			} else if e == nil || m != nil || ep.closes != 1 {
				t.Fatal("retained malformed bootstrap")
			}
		})
	}
	c := fixtureConfig([]byte("inert unit image"))
	c.Artifacts = append(c.Artifacts, g.Artifact{Path: "undeclared", Bytes: []byte("unused")})
	if _, e := approved(c); e == nil {
		t.Fatal("undeclared artifact union")
	}
	file, e := os.CreateTemp(t.TempDir(), "caller-owned")
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = file.Close() }()
	c = fixtureConfig([]byte("inert unit image"))
	c.Image.MeasurementChannel = file
	if _, e := approved(c); e == nil {
		t.Fatal("caller-provided replacement channel")
	}
	if _, e := file.Stat(); e != nil {
		t.Fatal("refusal stole caller ownership")
	}
}
