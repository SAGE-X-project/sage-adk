// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost_test

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	h "github.com/sage-x-project/sage/pkg/agent/hpke"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

const fixtureAlice = "did:sage:web:agent.example:alice"
const fixtureBob = "did:sage:web:agent.example:bob"
const fixtureMaterial = "inert-arithmetic-instance/1"

type fixtureClock struct{ mono atomic.Int64 }

func (c *fixtureClock) Now() (r.Stamp, error) {
	m := c.mono.Load()
	return r.Stamp{MonoMS: m, Unix: 100 + m/1000}, nil
}
func (c *fixtureClock) Sample(context.Context) (int64, int64, error) {
	m := c.mono.Load()
	return 100000 + m, m, nil
}
func fixtureHash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func fixtureJSON(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func fixtureManifest() []byte {
	return fixtureJSON(map[string]any{"version": "0.10.0", "files": []any{map[string]any{"path": "arithmetic.fixture", "sha256": fixtureHash([]byte(fixtureMaterial))}}})
}

type fixtureSource struct{ env *fixtureEnvironment }

func (s fixtureSource) Read(_ context.Context, did string) (r.Snapshot, error) {
	e := s.env
	e.reads.Add(1)
	var key ed25519.PrivateKey
	switch did {
	case fixtureAlice:
		key = e.alice
	case fixtureBob:
		key = e.bob
	default:
		return r.Snapshot{}, g.ErrInvalid
	}
	keys := []r.Key{{Name: "signing-1", Alg: "ed25519", Material: hex.EncodeToString(key.Public().(ed25519.PublicKey)), State: "accepted"}}
	if did == fixtureBob {
		kem, err := ecdh.X25519().NewPrivateKey(e.kem)
		if err != nil {
			return r.Snapshot{}, err
		}
		keys = append([]r.Key{{Name: "kem-1", Alg: "x25519", Material: hex.EncodeToString(kem.PublicKey().Bytes()), State: "accepted"}}, keys...)
	}
	stamp, _ := e.clock.Now()
	return r.Snapshot{Source: "adk-inert-fixture", Registry: "web:agent.example", Network: "local", DID: did, Version: "1", State: "active", Digest: fixtureHash([]byte("fixed-registry-fixture")), Ready: true, Validated: true, Finalized: true, AcquiredMS: stamp.MonoMS, Keys: keys}, nil
}

type fixtureEnvironment struct {
	reads      atomic.Int64
	root       string
	clock      *fixtureClock
	alice, bob ed25519.PrivateKey
	kem        []byte
	journals   []*r.Journal
	replays    []*h.ReplayJournal010
	counter    int
}

func newFixtureEnvironment(t *testing.T) *fixtureEnvironment {
	t.Helper()
	_, alice, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, bob, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	kem, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	e := &fixtureEnvironment{root: t.TempDir(), clock: &fixtureClock{}, alice: alice, bob: bob, kem: kem.Bytes()}
	return e
}
func (e *fixtureEnvironment) gate(t *testing.T) *r.Gate {
	t.Helper()
	e.counter++
	path := filepath.Join(e.root, "registry-"+string(rune('A'+e.counter)))
	journal, err := r.OpenJournal(path, true)
	if err != nil {
		t.Fatal(err)
	}
	e.journals = append(e.journals, journal)
	gate, err := r.NewGate(r.Config{Source: "adk-inert-fixture", Registry: "web:agent.example", Network: "local"}, fixtureSource{e}, e.clock, journal)
	if err != nil {
		t.Fatal(err)
	}
	return gate
}
func (e *fixtureEnvironment) authority(t *testing.T, did string) *g.RegistryAuthority {
	t.Helper()
	a, err := g.NewRegistryAuthority(e.gate(t), did, did+"#signing-1")
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func (e *fixtureEnvironment) endpoint(t *testing.T, client bool) *h.CompletionEndpoint010 {
	t.Helper()
	did, key, kem := fixtureBob, e.bob, e.kem
	if client {
		did, key, kem = fixtureAlice, e.alice, nil
	}
	e.counter++
	replay, err := h.OpenReplayJournal010(filepath.Join(e.root, "replay-"+string(rune('A'+e.counter))), true, e.clock)
	if err != nil {
		t.Fatal(err)
	}
	e.replays = append(e.replays, replay)
	endpoint, err := h.NewCompletionEndpoint010(did, did+"#signing-1", key.Seed(), kem, e.gate(t), e.clock, replay)
	if err != nil {
		t.Fatal(err)
	}
	return endpoint
}
func (e *fixtureEnvironment) close(t *testing.T) {
	t.Helper()
	for _, x := range e.journals {
		if err := x.Close(); err != nil {
			t.Error(err)
		}
	}
	for _, x := range e.replays {
		if err := x.Close(); err != nil {
			t.Error(err)
		}
	}
}

type fixturePolicy struct {
	id, digest string
	calls      atomic.Int64
	failAt     int64
}

func (p *fixturePolicy) Bindings(_ context.Context, issuer, id string) (string, []byte, []byte, error) {
	if issuer != fixtureAlice || id != p.id {
		return "", nil, nil, g.ErrInvalid
	}
	m := fixtureManifest()
	policy := fixtureJSON(map[string]any{"version": "0.10.0", "issuer": fixtureAlice, "epoch": "00000000-0000-4000-8000-000000000001", "engine": "inert-arithmetic-policy/1", "artifacts": json.RawMessage(m)})
	return p.digest, policy, m, nil
}
func (p *fixturePolicy) Authorize(_ context.Context, issuer, tool string, args []byte) error {
	n := p.calls.Add(1)
	if n == p.failAt || issuer != fixtureAlice || tool != "sum" || !bytes.Equal(args, []byte(`{"a":2,"b":3}`)) {
		return g.ErrInvalid
	}
	return nil
}
func (p *fixturePolicy) ApproveIntent(ctx context.Context, raw []byte) error {
	var i struct {
		Recipient string          `json:"recipient"`
		ID        string          `json:"request_id"`
		Tool      string          `json:"tool"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if json.Unmarshal(raw, &i) != nil || i.Recipient != fixtureBob || i.ID != p.id {
		return g.ErrInvalid
	}
	return p.Authorize(ctx, fixtureAlice, i.Tool, i.Arguments)
}

type fixtureLoadedTool struct {
	calls, checks atomic.Int64
	failAt        int64
	mode          string
	seen          chan []byte
}

func (l *fixtureLoadedTool) Check(ctx context.Context, manifest, tool string) error {
	n := l.checks.Add(1)
	if n == l.failAt {
		return g.ErrInvalid
	}
	digest, err := g.VerifyManifest(fixtureManifest(), []g.Artifact{{Path: "arithmetic.fixture", Bytes: []byte(fixtureMaterial)}})
	if err != nil || !bytes.Equal([]byte(digest), []byte(manifest)) || tool != "sum" || ctx.Err() != nil {
		return g.ErrInvalid
	}
	return nil
}
func (l *fixtureLoadedTool) Execute(ctx context.Context, args []byte) ([]byte, error) {
	if ctx.Err() != nil || !bytes.Equal(args, []byte(`{"a":2,"b":3}`)) {
		return nil, g.ErrInvalid
	}
	l.calls.Add(1)
	if l.seen != nil {
		l.seen <- append([]byte(nil), args...)
	}
	switch l.mode {
	case "panic":
		panic("inert execution fixture")
	case "error":
		return nil, errors.New("fixture unavailable")
	case "array":
		return []byte(`[]`), nil
	case "invalid":
		return []byte(`{"sum":`), nil
	case "oversize":
		return make([]byte, g.MaxBytes+1), nil
	case "cancel":
		<-ctx.Done()
		return nil, ctx.Err()
	}
	args[0] = 'x'
	return []byte(`{ "sum": 5 }`), nil
}

type fixtureResultSigner struct{ env *fixtureEnvironment }

func (s fixtureResultSigner) Now(context.Context) (int64, error) {
	stamp, err := s.env.clock.Now()
	return stamp.Unix, err
}
func (s fixtureResultSigner) ActiveKey(_ context.Context, did, kid string) (ed25519.PublicKey, error) {
	if did != fixtureBob || kid != fixtureBob+"#signing-1" {
		return nil, g.ErrInvalid
	}
	return s.env.bob.Public().(ed25519.PublicKey), nil
}
func (fixtureResultSigner) KeyID(context.Context) (string, error) {
	return fixtureBob + "#signing-1", nil
}
func (s fixtureResultSigner) Sign(ctx context.Context, kid string, b []byte) ([]byte, error) {
	if ctx.Err() != nil || kid != fixtureBob+"#signing-1" {
		return nil, g.ErrInvalid
	}
	return ed25519.Sign(s.env.bob, b), nil
}

type fixtureIntentSigner struct {
	env   *fixtureEnvironment
	path  string
	calls atomic.Int64
}

func (s *fixtureIntentSigner) Sign(_ context.Context, kid string, b []byte) ([]byte, error) {
	const domain = "sage-execution-intent|0.10.0\x00"
	if kid != fixtureAlice+"#signing-1" || !bytes.HasPrefix(b, []byte(domain)) {
		return nil, g.ErrInvalid
	}
	fence, err := os.ReadFile(s.path + ".issuance")
	if err != nil || string(fence) != "sage-intent-issuance|0.10.0\n"+fixtureHash(b[len(domain):])+"\n" {
		return nil, g.ErrInvalid
	}
	s.calls.Add(1)
	return ed25519.Sign(s.env.alice, b), nil
}

type fixtureNoSender struct{}

func (fixtureNoSender) Commit(context.Context, string, []byte) error { return g.ErrInvalid }

type fixtureHandler struct {
	endpoint *h.CompletionEndpoint010
	handle   func(context.Context, *g.MCPConnection) error
}

func (x *fixtureHandler) Endpoint(context.Context) (*h.CompletionEndpoint010, error) {
	return x.endpoint, nil
}
func (*fixtureHandler) Prepare(context.Context) error { return nil }
func (x *fixtureHandler) Handle(ctx context.Context, c *g.MCPConnection) error {
	return x.handle(ctx, c)
}
func fixtureBounds() g.MCPHostBounds {
	return g.MCPHostBounds{Capacity: 2, Preparations: 2, Clients: 2, Owners: 4, Workers: 1, Request: 20 * time.Second, Claim: 10 * time.Second, Worker: time.Second, Client: 20 * time.Second, Tick: time.Millisecond}
}
func fixtureConfig(client bool) g.MCPConnectionConfig {
	c := g.MCPConnectionConfig{Role: g.MCPResponder, Name: "inert ADK fixture", Version: "1", TTLSeconds: 300, Timeout: 3 * time.Second}
	if client {
		c.Role = g.MCPInitiator
		c.Recipient = fixtureBob
		c.RecipientKey = fixtureBob + "#signing-1"
	}
	return c
}
