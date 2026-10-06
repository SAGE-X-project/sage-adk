// SPDX-License-Identifier: LGPL-3.0-or-later
package capture_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sage-x-project/sage-adk/core/capture"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

const alice = "did:sage:web:agent.example:alice"
const bob = "did:sage:web:agent.example:bob"

type fixturePolicy struct {
	request      *capture.Request
	store        *memoryStore
	fail, change string
}

func (p *fixturePolicy) Bindings(_ context.Context, issuer, id string) (string, []byte, []byte, error) {
	if p.fail == "bindings" || issuer != alice || id != p.request.ID() {
		return "", nil, nil, g.ErrInvalid
	}
	original := p.request.Digest()
	if p.fail == "digest" {
		original = strings.Repeat("0", 64)
	}
	if p.change == "bindings" {
		p.store.corrupt = true
	}
	manifest := []byte(`{"version":"0.10.0","files":[{"path":"fixture.txt","sha256":"` + strings.Repeat("a", 64) + `"}]}`)
	policy := []byte(`{"version":"0.10.0","issuer":"` + alice + `","epoch":"00000000-0000-4000-8000-000000000001","engine":"inert-capture-fixture/1","artifacts":` + string(manifest) + `}`)
	return original, policy, manifest, nil
}
func (p *fixturePolicy) Authorize(_ context.Context, issuer, tool string, args []byte) error {
	if p.fail == "authorize" || issuer != alice || tool != "read" || !bytes.Equal(args, []byte(`{"path":"public.txt"}`)) {
		return g.ErrInvalid
	}
	if p.change == "authorize" {
		p.store.corrupt = true
	}
	return nil
}
func (p *fixturePolicy) ApproveIntent(_ context.Context, raw []byte) error {
	if p.fail == "approve" {
		return g.ErrInvalid
	}
	var v struct {
		RequestID string `json:"request_id"`
		Original  string `json:"original_digest"`
		Recipient string `json:"recipient"`
	}
	if json.Unmarshal(raw, &v) != nil || v.RequestID != p.request.ID() || v.Original != p.request.Digest() || v.Recipient != bob {
		return g.ErrInvalid
	}
	if p.change == "approve" {
		p.store.corrupt = true
	}
	return nil
}

type fixtureAuthority struct{ public ed25519.PublicKey }

func (a fixtureAuthority) ActiveKey(_ context.Context, issuer, key string) (ed25519.PublicKey, error) {
	if issuer != alice || key != alice+"#signing-1" {
		return nil, g.ErrInvalid
	}
	return a.public, nil
}
func (fixtureAuthority) Now(context.Context) (int64, error) { return 100, nil }

type fixtureClock struct{}

func (fixtureClock) Sample(context.Context) (int64, int64, error) { return 100000, 0, nil }

type noSender struct{}

func (noSender) Commit(context.Context, string, []byte) error { return g.ErrInvalid }

type fixtureMeasurement struct {
	store  *memoryStore
	change bool
}

func (m fixtureMeasurement) Check(context.Context, string, string) error {
	if m.change {
		m.store.corrupt = true
	}
	return nil
}

type fixtureSigner struct {
	key   ed25519.PrivateKey
	path  string
	calls int
}

func (s *fixtureSigner) Sign(_ context.Context, id string, body []byte) ([]byte, error) {
	const domain = "sage-execution-intent|0.10.0\x00"
	if id != alice+"#signing-1" || !bytes.HasPrefix(body, []byte(domain)) {
		return nil, g.ErrInvalid
	}
	sum := sha256.Sum256(body[len(domain):])
	raw, e := os.ReadFile(s.path + ".issuance")
	if e != nil || string(raw) != "sage-intent-issuance|0.10.0\n"+hex.EncodeToString(sum[:])+"\n" {
		return nil, g.ErrInvalid
	}
	s.calls++
	return ed25519.Sign(s.key, body), nil
}
func issuerFixture(t *testing.T) (*capture.Request, *memoryStore, *fixturePolicy, *fixtureSigner, g.IssuerServices) {
	t.Helper()
	store := &memoryStore{}
	host, _ := capture.NewHost(store)
	r, e := host.Capture(context.Background(), [][]byte{[]byte(" 요청\n"), []byte("e\u0301")})
	if e != nil {
		t.Fatal(e)
	}
	pub, priv, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	p := &fixturePolicy{request: r, store: store}
	s := &fixtureSigner{key: priv, path: filepath.Join(t.TempDir(), "intent")}
	authority := fixtureAuthority{pub}
	services := g.IssuerServices{Client: g.ClientServices{IntentAuthority: authority, ResultAuthority: authority, Policy: p, Clock: fixtureClock{}, Sender: noSender{}, ExpectedIssuer: alice, ExpectedRecipient: bob}, Policy: p, Signer: s, Measurement: fixtureMeasurement{store: store}, KeyID: alice + "#signing-1"}
	return r, store, p, s, services
}
func proposal() g.IntentProposal {
	return g.IntentProposal{Tool: "read", Arguments: []byte(`{"path":"public.txt"}`), LifetimeSeconds: 30}
}
func TestCapturedIntentIssuance(t *testing.T) {
	r, _, _, signer, services := issuerFixture(t)
	ctx := context.Background()
	issuer, e := r.NewIntentIssuer(ctx, services)
	if e != nil {
		t.Fatal(e)
	}
	defer issuer.Retire()
	approved, e := issuer.Authorize(ctx, proposal())
	if e != nil {
		t.Fatal(e)
	}
	if signer.calls != 0 {
		t.Fatal("approval signed")
	}
	client, e := issuer.Issue(ctx, signer.path, approved)
	if e != nil {
		t.Fatal(e)
	}
	defer client.Close()
	raw, e := client.JournaledIntent()
	if e != nil {
		t.Fatal(e)
	}
	var envelope struct {
		Intent json.RawMessage `json:"intent"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		t.Fatal("intent envelope")
	}
	var intent struct {
		Original string `json:"original_digest"`
		ID       string `json:"request_id"`
	}
	if json.Unmarshal(envelope.Intent, &intent) != nil {
		t.Fatal("intent")
	}
	if intent.ID != r.ID() || intent.Original != commitment([][]byte{[]byte(" 요청\n"), []byte("e\u0301")}) || signer.calls != 1 {
		t.Fatal("signed input binding differs")
	}
	if _, e = issuer.Issue(ctx, signer.path, approved); e == nil || signer.calls != 1 {
		t.Fatal("approval reused")
	}
}
func TestIssuerRejectsChangedInputBeforeSigning(t *testing.T) {
	for _, mode := range []string{"changed", "unavailable", "bindings", "digest", "authorize", "approve", "change-bindings", "change-authorize", "change-approve", "measurement", "client-policy", "signer-panic"} {
		t.Run(mode, func(t *testing.T) {
			r, store, p, signer, services := issuerFixture(t)
			ctx := context.Background()
			issuer, e := r.NewIntentIssuer(ctx, services)
			if e != nil {
				t.Fatal(e)
			}
			defer issuer.Retire()
			approved, e := issuer.Authorize(ctx, proposal())
			if e != nil {
				t.Fatal(e)
			}
			switch mode {
			case "changed":
				store.corrupt = true
			case "unavailable":
				store.failLoad = true
			case "bindings", "digest", "authorize", "approve":
				p.fail = mode
			case "change-bindings", "change-authorize", "change-approve":
				p.change = strings.TrimPrefix(mode, "change-")
			case "measurement":
				issuer.Retire()
				services.Measurement = fixtureMeasurement{store: store, change: true}
				issuer, e = r.NewIntentIssuer(ctx, services)
				if e != nil {
					t.Fatal(e)
				}
				approved, e = issuer.Authorize(ctx, proposal())
				if e != nil {
					return
				}
			case "client-policy":
				issuer.Retire()
				services.Client.Policy = &fixturePolicy{request: r, store: store, fail: "digest"}
				issuer, e = r.NewIntentIssuer(ctx, services)
				if e != nil {
					t.Fatal(e)
				}
				approved, e = issuer.Authorize(ctx, proposal())
				if e != nil {
					return
				}
			case "signer-panic":
				store.panicLoad = true
			}
			if _, e = issuer.Issue(ctx, signer.path, approved); e == nil || signer.calls != 0 {
				t.Fatal("invalid input reached signing")
			}
		})
	}
}
func TestIssuerRequiresServicesAndAvailableInput(t *testing.T) {
	r, store, _, _, services := issuerFixture(t)
	ctx := context.Background()
	for _, missing := range []string{"policy", "client-policy", "signer", "measurement", "authority", "key"} {
		t.Run(missing, func(t *testing.T) {
			s := services
			switch missing {
			case "policy":
				s.Policy = nil
			case "client-policy":
				s.Client.Policy = nil
			case "signer":
				s.Signer = nil
			case "measurement":
				s.Measurement = nil
			case "authority":
				s.Client.IntentAuthority = nil
			case "key":
				s.KeyID = ""
			}
			if _, e := r.NewIntentIssuer(ctx, s); e == nil {
				t.Fatal("missing service")
			}
		})
	}
	store.corrupt = true
	if _, e := r.NewIntentIssuer(ctx, services); e == nil {
		t.Fatal("changed input")
	}
	var zero capture.Request
	if _, e := zero.NewIntentIssuer(ctx, services); e == nil {
		t.Fatal("zero request")
	}
}

func TestTypedNilServicesRefused(t *testing.T) {
	r, _, _, _, services := issuerFixture(t)
	var authority *fixtureAuthority
	var clock *fixtureClock
	var sender *noSender
	var measurement *fixtureMeasurement
	var policy *fixturePolicy
	var signer *fixtureSigner
	for _, missing := range []string{"intent-authority", "result-authority", "clock", "sender", "measurement", "policy", "client-policy", "signer"} {
		t.Run(missing, func(t *testing.T) {
			s := services
			switch missing {
			case "intent-authority":
				s.Client.IntentAuthority = authority
			case "result-authority":
				s.Client.ResultAuthority = authority
			case "clock":
				s.Client.Clock = clock
			case "sender":
				s.Client.Sender = sender
			case "measurement":
				s.Measurement = measurement
			case "policy":
				s.Policy = policy
			case "client-policy":
				s.Client.Policy = policy
			case "signer":
				s.Signer = signer
			}
			if _, e := r.NewIntentIssuer(context.Background(), s); e == nil {
				t.Fatal("typed nil service accepted")
			}
		})
	}
}
