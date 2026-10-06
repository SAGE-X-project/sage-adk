// SPDX-License-Identifier: LGPL-3.0-or-later
package guardservices_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	p "github.com/sage-x-project/sage-adk/core/guardservices"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

const principal = "did:sage:web:agent.example:alice"
const keyid = principal + "#signing-1"

type source struct {
	key                  ed25519.PublicKey
	clock                *p.SystemClock
	unavailable, revoked atomic.Bool
	alg                  string
	panicAt              int64
	reads                atomic.Int64
}

func (s *source) Read(ctx context.Context, did string) (r.Snapshot, error) {
	if s.reads.Add(1) == s.panicAt {
		panic("inert registry failure")
	}
	if ctx.Err() != nil || s.unavailable.Load() {
		return r.Snapshot{}, g.ErrInvalid
	}
	stamp, e := s.clock.Now()
	if e != nil {
		return r.Snapshot{}, e
	}
	state := "active"
	if s.revoked.Load() {
		state = "deactivated"
	}
	alg := "ed25519"
	if s.alg != "" {
		alg = s.alg
	}
	return r.Snapshot{Source: "owned-fixture", Registry: "web:agent.example", Network: "local", DID: did, Version: "1", Digest: strings.Repeat("a", 64), State: state, Ready: true, Validated: true, Finalized: true, AcquiredMS: stamp.MonoMS, Keys: []r.Key{{Name: "signing-1", Alg: alg, Material: hex.EncodeToString(s.key), State: "accepted"}}}, nil
}

type backend struct {
	key                                                 ed25519.PrivateKey
	calls                                               int
	fail, pubFail, panicSign, panicPub, invalid, mutate bool
	before                                              func()
}

func (b *backend) PublicKey(context.Context) (ed25519.PublicKey, error) {
	if b.panicPub {
		panic("inert custody failure")
	}
	if b.pubFail {
		return nil, g.ErrInvalid
	}
	return b.key.Public().(ed25519.PublicKey), nil
}
func (b *backend) Sign(_ context.Context, m []byte) ([]byte, error) {
	b.calls++
	if b.before != nil {
		b.before()
	}
	if b.panicSign {
		panic("inert signing failure")
	}
	if b.fail {
		return nil, g.ErrInvalid
	}
	if b.invalid {
		return make([]byte, 64), nil
	}
	if b.mutate {
		m[0] = 'x'
	}
	return ed25519.Sign(b.key, m), nil
}
func fixture(t *testing.T) (*g.RegistryAuthority, *source, *backend) {
	t.Helper()
	_, k, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	clock := p.NewSystemClock()
	s := &source{key: k.Public().(ed25519.PublicKey), clock: clock}
	j, e := r.OpenJournal(filepath.Join(t.TempDir(), "registry"), true)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := j.Close(); e != nil {
			t.Error(e)
		}
	})
	gate, e := r.NewGate(r.Config{Source: "owned-fixture", Registry: "web:agent.example", Network: "local"}, s, clock, j)
	if e != nil {
		t.Fatal(e)
	}
	a, e := g.NewRegistryAuthority(gate, principal, keyid)
	if e != nil {
		t.Fatal(e)
	}
	return a, s, &backend{key: k}
}
func body() []byte {
	return []byte(fmt.Sprintf(`{"alg":"ed25519","issuer":%q,"keyid":%q,"version":"0.10.0"}`, principal, keyid))
}
func TestSignerDomainsAndCurrentAuthority(t *testing.T) {
	a, _, b := fixture(t)
	ctx := context.Background()
	i, e := p.NewIntentSigner(ctx, a, principal, keyid, b)
	if e != nil {
		t.Fatal(e)
	}
	result, e := p.NewResultSigner(ctx, a, principal, keyid, b)
	if e != nil {
		t.Fatal(e)
	}
	for _, x := range []struct {
		sign   g.IntentSigner
		domain string
	}{{i, "sage-execution-intent|0.10.0\x00"}, {result, "sage-tool-result|0.10.0\x00"}} {
		msg := append([]byte(x.domain), body()...)
		before := append([]byte(nil), msg...)
		proof, e := x.sign.Sign(ctx, keyid, msg)
		if e != nil || !ed25519.Verify(b.key.Public().(ed25519.PublicKey), msg, proof) || !bytes.Equal(before, msg) {
			t.Fatal(e)
		}
		other := "sage-tool-result|0.10.0\x00"
		if x.domain == other {
			other = "sage-execution-intent|0.10.0\x00"
		}
		calls := b.calls
		for _, m := range [][]byte{append([]byte(other), body()...), []byte(x.domain), append([]byte(x.domain), []byte(`[]`)...), append([]byte(x.domain), []byte(`{ "alg":"ed25519" }`)...), append([]byte(x.domain), bytes.ReplaceAll(body(), []byte("ed25519"), []byte("x25519"))...)} {
			if output, e := x.sign.Sign(ctx, keyid, m); e == nil || output != nil {
				t.Fatal("unscoped signing")
			}
		}
		if b.calls != calls {
			t.Fatal("key used for rejected input")
		}
	}
	if kid, e := result.KeyID(ctx); e != nil || kid != keyid {
		t.Fatal(kid, e)
	}
	if _, e := result.Now(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e := result.ActiveKey(ctx, principal, keyid); e != nil {
		t.Fatal(e)
	}
	if _, e := result.ActiveKey(ctx, principal, "other"); e == nil {
		t.Fatal("other key")
	}
}
func TestSigningFailuresNeverPublishProof(t *testing.T) {
	for _, mode := range []string{"revoked", "unavailable", "mismatched", "public-error", "public-panic", "sign-error", "sign-panic", "invalid-proof", "mutated-input", "revoked-after-use", "cancel-after-use", "cancelled", "wrong-key"} {
		t.Run(mode, func(t *testing.T) {
			a, s, b := fixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			signer, e := p.NewIntentSigner(ctx, a, principal, keyid, b)
			if e != nil {
				t.Fatal(e)
			}
			kid := keyid
			want := 0
			switch mode {
			case "revoked":
				s.revoked.Store(true)
			case "unavailable":
				s.unavailable.Store(true)
			case "mismatched":
				_, b.key, _ = ed25519.GenerateKey(rand.Reader)
			case "public-error":
				b.pubFail = true
			case "public-panic":
				b.panicPub = true
			case "sign-error":
				b.fail = true
				want = 1
			case "sign-panic":
				b.panicSign = true
				want = 1
			case "invalid-proof":
				b.invalid = true
				want = 1
			case "mutated-input":
				b.mutate = true
				want = 1
			case "revoked-after-use":
				b.before = func() { s.revoked.Store(true) }
				want = 1
			case "cancel-after-use":
				b.before = cancel
				want = 1
			case "cancelled":
				cancel()
			case "wrong-key":
				kid = "other"
			}
			if proof, e := signer.Sign(ctx, kid, append([]byte("sage-execution-intent|0.10.0\x00"), body()...)); e == nil || proof != nil {
				t.Fatal("failed signing published proof")
			}
			if b.calls != want {
				t.Fatalf("key uses %d want %d", b.calls, want)
			}
		})
	}
}

func TestConstructorsRequireCurrentSigningCustody(t *testing.T) {
	for _, mode := range []string{"nil-authority", "nil-backend", "typed-nil-backend", "nil-context", "cancelled", "wrong-issuer", "wrong-key", "revoked", "kem-only", "public-error", "public-panic", "mismatched"} {
		t.Run(mode, func(t *testing.T) {
			a, s, b := fixture(t)
			ctx := context.Background()
			issuer, kid := principal, keyid
			var custody p.Ed25519Backend = b
			switch mode {
			case "nil-authority":
				a = nil
			case "nil-backend":
				custody = nil
			case "typed-nil-backend":
				var nilBackend *backend
				custody = nilBackend
			case "nil-context":
				ctx = nil
			case "cancelled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			case "wrong-issuer":
				issuer = principal + "other"
			case "wrong-key":
				kid = "other"
			case "revoked":
				s.revoked.Store(true)
			case "kem-only":
				s.alg = "x25519"
			case "public-error":
				b.pubFail = true
			case "public-panic":
				b.panicPub = true
			case "mismatched":
				_, b.key, _ = ed25519.GenerateKey(rand.Reader)
			}
			if signer, e := p.NewIntentSigner(ctx, a, issuer, kid, custody); e == nil || signer != nil {
				t.Fatal("invalid intent custody accepted")
			}
			if signer, e := p.NewResultSigner(ctx, a, issuer, kid, custody); e == nil || signer != nil {
				t.Fatal("invalid result custody accepted")
			}
			if b.calls != 0 {
				t.Fatal("constructor used signing key")
			}
		})
	}
}
func TestResultProviderRefusesUnavailableAuthority(t *testing.T) {
	a, s, b := fixture(t)
	ctx := context.Background()
	result, e := p.NewResultSigner(ctx, a, principal, keyid, b)
	if e != nil {
		t.Fatal(e)
	}
	s.revoked.Store(true)
	if _, e = result.KeyID(ctx); e == nil {
		t.Fatal("revoked keyid")
	}
	if _, e = result.Now(ctx); e == nil {
		t.Fatal("revoked time")
	}
	if _, e = result.ActiveKey(ctx, principal, keyid); e == nil {
		t.Fatal("revoked key")
	}
	s.revoked.Store(false)
	s.panicAt = s.reads.Load() + 2
	if _, e = result.Now(ctx); e == nil {
		t.Fatal("authority panic escaped")
	}
}
func TestInvalidSignerInstancesRefuseAllCalls(t *testing.T) {
	ctx := context.Background()
	var i *p.IntentSigner
	var r *p.ResultSigner
	for _, x := range []g.IntentSigner{i, &p.IntentSigner{}, r, &p.ResultSigner{}} {
		if sig, e := x.Sign(ctx, keyid, nil); e == nil || sig != nil {
			t.Fatal("invalid signer")
		}
	}
	for _, x := range []*p.ResultSigner{r, {}} {
		if _, e := x.Now(ctx); e == nil {
			t.Fatal("invalid now")
		}
		if _, e := x.KeyID(ctx); e == nil {
			t.Fatal("invalid keyid")
		}
		if _, e := x.ActiveKey(ctx, principal, keyid); e == nil {
			t.Fatal("invalid key")
		}
	}
}
