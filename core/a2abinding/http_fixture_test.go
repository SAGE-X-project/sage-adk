package a2abinding

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sage-x-project/did-go/advanced/sage"
	didgo "github.com/sage-x-project/did-go/did"
	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

const protectedAlice = "did:sage:web:agents.example.com:client"

// Entirely owned prevalidated fixture authority, never an operating Registry.
// Runtime controls below carry only inert messages or unavailable/expired state.
type protectedFixture struct {
	mu                  sync.Mutex
	mono, unix          int64
	mode, target        string
	reads, observations int
	hook                func(string, string)
}

func (f *protectedFixture) Now() (registry010.Stamp, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.mode == "clock-error" {
		return registry010.Stamp{}, errors.New("fixture clock unavailable")
	}
	return registry010.Stamp{MonoMS: f.mono, Unix: f.unix}, nil
}
func (f *protectedFixture) advance(ms, seconds int64) {
	f.mu.Lock()
	f.mono += ms
	f.unix += seconds
	f.mu.Unlock()
}
func (f *protectedFixture) setMode(mode string) { f.mu.Lock(); f.mode = mode; f.mu.Unlock() }
func (f *protectedFixture) setHook(hook func(string, string)) {
	f.mu.Lock()
	f.hook = hook
	f.mu.Unlock()
}
func (f *protectedFixture) record(ctx context.Context, did, stage string) (sage.Record, error) {
	if ctx == nil || ctx.Err() != nil {
		return sage.Record{}, ErrRejected
	}
	f.mu.Lock()
	hook := f.hook
	f.mu.Unlock()
	if hook != nil {
		hook(stage, did)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if stage == "read" {
		f.reads++
	} else {
		f.observations++
	}
	if ctx.Err() != nil || f.mode == "source-error" || (stage == "read" && f.mode == "read-error") || (stage == "observe" && f.mode == "observe-error") {
		return sage.Record{}, ErrRejected
	}
	if did != protectedAlice && did != peer {
		return sage.Record{}, ErrRejected
	}
	d, e := didgo.Parse(did)
	if e != nil {
		return sage.Record{}, e
	}
	seed := byte(1)
	if did == peer {
		seed = 0
	}
	key := sage.Key{Name: "sign", Alg: sage.AlgEd25519, State: sage.KeyAccepted, Material: append([]byte(nil), ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32)).Public().(ed25519.PublicKey)...)}
	version := "3"
	if (f.mode == "revoke-client" && did == protectedAlice) || (f.mode == "revoke-server" && did == peer) {
		key.State = sage.KeyRevoked
		version = "4"
	}
	keys := []sage.Key{}
	if did == peer {
		kem, e := ecdh.X25519().NewPrivateKey(bytes.Repeat([]byte{3}, 32))
		if e != nil {
			return sage.Record{}, e
		}
		keys = append(keys, sage.Key{Name: "kem", Alg: sage.AlgX25519, State: sage.KeyAccepted, Material: kem.PublicKey().Bytes()})
	}
	keys = append(keys, key)
	issued := int64(100)
	return sage.Record{DID: d, Controller: "fixture-operator", State: sage.StateActive, Version: version, Keys: keys, Services: []sage.Service{{Name: "a2a", Type: "A2AInterface", URI: f.target}}, Observation: sage.Observation{Source: "owned-a2a-fixture", Issued: &issued, Time: f.unix}}, nil
}
func (f *protectedFixture) Observe(ctx context.Context, did didgo.DID) (sage.Record, error) {
	return f.record(ctx, did.String(), "observe")
}
func (f *protectedFixture) Read(ctx context.Context, did string) (registry010.Snapshot, error) {
	r, e := f.record(ctx, did, "read")
	if e != nil {
		return registry010.Snapshot{}, e
	}
	keys := []registry010.Key{}
	for _, k := range r.Keys {
		keys = append(keys, registry010.Key{Name: k.Name, Alg: k.Alg, State: string(k.State), Material: hex.EncodeToString(k.Material)})
	}
	// Observation time is not part of the canonical immutable record digest.
	body, e := json.Marshal(map[string]any{"did": did, "controller": r.Controller, "state": r.State, "version": r.Version, "keys": keys, "services": r.Services})
	if e != nil {
		return registry010.Snapshot{}, e
	}
	digest := sha256.Sum256(body)
	f.mu.Lock()
	mono := f.mono
	f.mu.Unlock()
	return registry010.Snapshot{Source: "owned-a2a-fixture", Registry: "web:agents.example.com", Network: "local", DID: did, Version: r.Version, State: string(r.State), Digest: hex.EncodeToString(digest[:]), Ready: true, Validated: true, Finalized: true, AcquiredMS: mono, Keys: keys}, nil
}

type protectedReplay struct {
	mu       sync.Mutex
	seen     map[string]bool
	accepted int
}

func (r *protectedReplay) reserve(v hpke.Replay010, contextKey bool, validate func() error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	prefix := v.Sender + "|" + v.Recipient + "|"
	keys := []string{prefix + "id|" + v.ID, prefix + "nonce|" + v.Nonce}
	if contextKey && v.Context != "" {
		keys = append(keys, v.Sender+"|context|"+v.Context)
	}
	for _, k := range keys {
		if r.seen[k] {
			return ErrRejected
		}
	}
	for _, k := range keys {
		r.seen[k] = true
	}
	if validate != nil {
		if e := validate(); e != nil {
			return e
		}
		r.accepted++
	}
	return nil
}
func (r *protectedReplay) Reserve(v hpke.Replay010) error { return r.reserve(v, true, nil) }
func (r *protectedReplay) ReserveRecord(v hpke.Replay010, validate func() error) error {
	return r.reserve(v, false, validate)
}

type protectedPair struct {
	left, right             *hpke.AuthenticatedCompletion010
	fixture                 *protectedFixture
	leftReplay, rightReplay *protectedReplay
}

func protectedSessions(t *testing.T, target string, httpHandshake bool) protectedPair {
	t.Helper()
	f := &protectedFixture{unix: 100, target: target}
	stores := []*protectedReplay{}
	makeEndpoint := func(did string, seed byte) *hpke.CompletionEndpoint010 {
		journal, e := registry010.OpenJournal(filepath.Join(t.TempDir(), "registry"), true)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = journal.Close() })
		gate, e := registry010.NewGate(registry010.Config{Source: "owned-a2a-fixture", Registry: "web:agents.example.com", Network: "local"}, f, f, journal)
		if e != nil {
			t.Fatal(e)
		}
		store := &protectedReplay{seen: map[string]bool{}}
		stores = append(stores, store)
		var kem []byte
		if did == peer {
			kem = bytes.Repeat([]byte{3}, 32)
		}
		endpoint, e := hpke.NewCompletionEndpoint010(did, did+"#sign", bytes.Repeat([]byte{seed}, 32), kem, gate, f, store)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(endpoint.Close)
		return endpoint
	}
	a, b := makeEndpoint(protectedAlice, 1), makeEndpoint(peer, 0)
	ctx := context.Background()
	var left, right *hpke.AuthenticatedCompletion010
	if httpHandshake {
		if a.BindHTTP(target) != nil || b.BindHTTP(target) != nil {
			t.Fatal("fixture HTTP setup")
		}
		pending, hello, e := a.StartHTTP(ctx, peer, peer+"#sign", 300)
		if e != nil {
			t.Fatal(e)
		}
		var reply hpke.HTTPMessage010
		right, reply, e = b.RespondHTTP(ctx, hello, 300)
		if e != nil {
			t.Fatal(e)
		}
		left, e = pending.CompleteHTTP(ctx, reply)
		if e != nil {
			t.Fatal(e)
		}
	} else {
		pending, hello, e := a.Start(ctx, peer, peer+"#sign", 300)
		if e != nil {
			t.Fatal(e)
		}
		var reply []byte
		right, reply, e = b.Respond(ctx, hello, 300)
		if e != nil {
			t.Fatal(e)
		}
		left, e = pending.Complete(ctx, reply)
		if e != nil {
			t.Fatal(e)
		}
	}
	t.Cleanup(left.Close)
	t.Cleanup(right.Close)
	return protectedPair{left, right, f, stores[0], stores[1]}
}
func (p protectedPair) verifier(t *testing.T) *Verifier {
	t.Helper()
	v, e := NewVerifier(VerifierConfig{Source: p.fixture, Clock: p.fixture, ExpectedPeer: peer})
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func (p protectedPair) card(t *testing.T) []byte {
	t.Helper()
	m, _, key := fixture(t)
	m["supportedInterfaces"].([]any)[0].(map[string]any)["url"] = p.fixture.target
	return signed(t, m, key)
}
func protectedCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, 32))
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "owned SAGE fixture"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, e := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if e != nil {
		t.Fatal(e)
	}
	cert, e := x509.ParseCertificate(der)
	if e != nil {
		t.Fatal(e)
	}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: cert}, roots
}
