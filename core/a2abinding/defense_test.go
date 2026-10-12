package a2abinding

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/sage-x-project/did-go/advanced/sage"
	didgo "github.com/sage-x-project/did-go/did"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

func verifier(t *testing.T, s *source) *Verifier {
	t.Helper()
	v, e := NewVerifier(VerifierConfig{Source: s, Clock: &clock{}, ExpectedPeer: peer})
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func parameters(m map[string]any) map[string]any {
	return m["capabilities"].(map[string]any)["extensions"].([]any)[0].(map[string]any)["params"].(map[string]any)
}
func encode(v any) []byte { b, _ := json.Marshal(v); return b }
func rejectCard(t *testing.T, s *source, raw []byte) {
	t.Helper()
	if _, e := verifier(t, s).Verify(context.Background(), raw); e != ErrRejected {
		t.Fatalf("expected generic failure: %v", e)
	}
}
func TestCardStructureAndBounds(t *testing.T) {
	for _, raw := range [][]byte{[]byte("[]"), []byte("{} {}"), []byte(`{"x":1,"\u0078":2}`), []byte(`{"x":null}`), []byte(`{"x":-0}`), []byte(`{"x":-1e-400}`), []byte(`{"x":1e400}`), []byte(`{"x":"\ud800"}`), []byte(`{"x":"\udc00"}`), []byte(`{"x":"\ud800\u0041"}`), []byte(`{"x":"\uZZZZ"}`), []byte(`{"x":"\q"}`), []byte(`{"x":"`), []byte(`{"x":}`), []byte{255}, append([]byte{0xef, 0xbb, 0xbf}, []byte("{}")...), bytes.Repeat([]byte(" "), 262145), []byte(strings.Repeat("[", 65) + strings.Repeat("]", 65))} {
		_, s, _ := fixture(t)
		rejectCard(t, s, raw)
		if s.calls != 0 {
			t.Fatal("invalid shape reached source")
		}
	}
	for _, name := range []string{"unknown", "nested unknown", "alias duplicate", "wrong type", "required missing", "required empty", "tags empty", "streaming", "push", "extended", "params extra", "params string", "extension duplicate", "no extension", "extension malformed", "wrong sage", "bad did", "bad record", "fraction", "integer range", "negative issued", "long ttl", "zero ttl", "URI", "no signatures"} {
		t.Run(name, func(t *testing.T) {
			m, s, k := fixture(t)
			switch name {
			case "unknown":
				m["invented"] = true
			case "nested unknown":
				m["skills"].([]any)[0].(map[string]any)["invented"] = true
			case "alias duplicate":
				m["supported_interfaces"] = m["supportedInterfaces"]
			case "wrong type":
				m["name"] = 42
			case "required missing":
				delete(m, "description")
			case "required empty":
				m["defaultInputModes"] = []any{}
			case "tags empty":
				m["skills"].([]any)[0].(map[string]any)["tags"] = []any{}
			case "streaming":
				m["capabilities"].(map[string]any)["streaming"] = true
			case "push":
				m["capabilities"].(map[string]any)["pushNotifications"] = true
			case "extended":
				m["capabilities"].(map[string]any)["extendedAgentCard"] = true
			case "params extra":
				parameters(m)["extra"] = true
			case "params string":
				m["capabilities"].(map[string]any)["extensions"].([]any)[0].(map[string]any)["params"] = "bad"
			case "extension duplicate":
				e := m["capabilities"].(map[string]any)["extensions"].([]any)
				m["capabilities"].(map[string]any)["extensions"] = append(e, e[0])
			case "no extension":
				m["capabilities"].(map[string]any)["extensions"] = []any{}
			case "extension malformed":
				delete(parameters(m), "issued")
			case "wrong sage":
				parameters(m)["sageVersion"] = "1.0"
			case "bad did":
				parameters(m)["did"] = "did:sage:web:EXAMPLE.com:a"
			case "bad record":
				parameters(m)["recordVersion"] = "03"
			case "fraction":
				parameters(m)["issued"] = json.Number("100.000000000000001")
			case "integer range":
				parameters(m)["expires"] = json.Number("9007199254740992")
			case "negative issued":
				parameters(m)["issued"] = -1
			case "long ttl":
				parameters(m)["expires"] = 86501
			case "zero ttl":
				parameters(m)["expires"] = 100
			case "URI":
				m["supportedInterfaces"].([]any)[0].(map[string]any)["url"] = "http://agents.example.com/a2a"
			case "no signatures":
				rejectCard(t, s, encode(m))
				return
			}
			rejectCard(t, s, signed(t, m, k))
		})
	}
}
func TestPresenceAliasesAndOwnedCard(t *testing.T) {
	m, s, k := fixture(t)
	raw := signed(t, m, k)
	var wire map[string]any
	json.Unmarshal(raw, &wire)
	wire["securitySchemes"] = map[string]any{} // ordinary default omitted before JCS
	c, e := verifier(t, s).Verify(context.Background(), encode(wire))
	if e != nil {
		t.Fatal(e)
	}
	var normalized map[string]any
	json.Unmarshal(c.JSON(), &normalized)
	if _, ok := normalized["securitySchemes"]; ok {
		t.Fatal("ordinary empty map retained")
	}
	if normalized["description"] != "" || normalized["capabilities"].(map[string]any)["streaming"] != false {
		t.Fatal("required/optional defaults lost")
	}
	b := c.JSON()
	b[0] = '!'
	in := c.Interfaces()
	in[0].URL = "changed"
	if c.JSON()[0] != '{' || c.Interfaces()[0].URL != endpoint {
		t.Fatal("caller mutated verified view")
	}
	delete(wire, "supportedInterfaces")
	wire["supported_interfaces"] = m["supportedInterfaces"]
	if _, e := verifier(t, s).Verify(context.Background(), encode(wire)); e != nil {
		t.Fatal("valid protobuf JSON name rejected", e)
	}
	m, s, k = fixture(t)
	m["name"] = "Emoji \U0001f600"
	m["capabilities"].(map[string]any)["extensions"] = append(m["capabilities"].(map[string]any)["extensions"].([]any), map[string]any{"uri": "urn:optional", "params": map[string]any{"arbitrary": map[string]any{"signatures": "covered", "n": 1.5}}})
	if _, e := verifier(t, s).Verify(context.Background(), signed(t, m, k)); e != nil {
		t.Fatal(e)
	}
}
func TestSignatureShapeKeyRoleAndFreshRecord(t *testing.T) {
	for _, name := range []string{"alg EdDSA", "alg secp", "jku", "unprotected", "duplicate", "kid wrong", "kid missing", "proof bytes", "proof padded", "no candidate", "malformed SAGE header", "ignored other", "wrong source did", "duplicate key", "expired key", "kem key", "source error", "future observation", "old observation", "negative observation"} {
		t.Run(name, func(t *testing.T) {
			m, s, k := fixture(t)
			raw := signed(t, m, k)
			json.Unmarshal(raw, &m)
			entry := m["signatures"].([]any)[0].(map[string]any)
			header := map[string]any{"alg": "Ed25519", "kid": peer + "#sign", "typ": CardType}
			switch name {
			case "alg EdDSA":
				header["alg"] = "EdDSA"
			case "alg secp":
				header["alg"] = "ES256K"
			case "jku":
				header["jku"] = "https://example.invalid/key"
			case "unprotected":
				entry["header"] = map[string]any{}
			case "duplicate":
				m["signatures"] = []any{entry, entry}
			case "kid wrong":
				header["kid"] = "did:sage:web:agents.example.com:other#sign"
			case "kid missing":
				header["kid"] = peer + "#missing"
			case "proof bytes":
				entry["signature"] = "AA"
			case "proof padded":
				entry["signature"] = entry["signature"].(string) + "="
			case "no candidate":
				entry["protected"] = "!!"
			case "malformed SAGE header":
				entry["protected"] = base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"sage-a2a-card+jws","alg":"Ed25519","alg":"Ed25519","kid":"` + peer + `#sign"}`))
			case "ignored other":
				m["signatures"] = append(m["signatures"].([]any), map[string]any{"protected": "!!", "signature": "!!"})
			case "wrong source did":
				s.record.DID, _ = didgo.Parse("did:sage:web:agents.example.com:other")
			case "duplicate key":
				s.record.Keys = append(s.record.Keys, s.record.Keys[0])
			case "expired key":
				s.record.Keys[0].Expires = new(int64(101))
			case "kem key":
				s.record.Keys[0].Alg = sage.AlgX25519
			case "source error":
				s.err = errors.New("private source detail")
			case "future observation":
				s.record.Observation.Time = 102
			case "old observation":
				s.record.Observation.Time = 90
			case "negative observation":
				s.record.Observation.Time = -1
			}
			if strings.HasPrefix(name, "alg") || name == "jku" || strings.HasPrefix(name, "kid") {
				entry["protected"] = base64.RawURLEncoding.EncodeToString(encode(header))
			}
			_, e := verifier(t, s).Verify(context.Background(), encode(m))
			if name == "ignored other" {
				if e != nil {
					t.Fatal(e)
				}
			} else if e != ErrRejected {
				t.Fatal("expected generic rejection", e)
			}
		})
	}
}

type clockFunc func() (registry010.Stamp, error)

func (f clockFunc) Now() (registry010.Stamp, error) { return f() }
func TestFinalClockFailureAndDeadline(t *testing.T) {
	for _, ticks := range [][]registry010.Stamp{{{MonoMS: -1, Unix: 101}}, {{MonoMS: 100, Unix: -1}}, {{MonoMS: 100, Unix: 9007199254740992}}, {{MonoMS: 100, Unix: 101}, {MonoMS: 99, Unix: 101}}, {{MonoMS: 100, Unix: 101}, {MonoMS: 100, Unix: 100}}, {{MonoMS: 100, Unix: 101}, {MonoMS: 5101, Unix: 101}}, {{MonoMS: 100, Unix: 101}, {MonoMS: 100, Unix: 105}}} {
		m, s, k := fixture(t)
		v, _ := NewVerifier(VerifierConfig{Source: s, Clock: &clock{ticks: ticks}})
		if _, e := v.Verify(context.Background(), signed(t, m, k)); e != ErrRejected {
			t.Fatal(e)
		}
	}
	m, s, k := fixture(t)
	v, _ := NewVerifier(VerifierConfig{Source: s, Clock: clockFunc(func() (registry010.Stamp, error) { return registry010.Stamp{}, errors.New("clock lost") })})
	if _, e := v.Verify(context.Background(), signed(t, m, k)); e != ErrRejected {
		t.Fatal(e)
	}
	for _, c := range []VerifierConfig{{}, {Source: s}, {Source: s, Clock: &clock{}, ExpectedPeer: "bad"}} {
		if _, e := NewVerifier(c); e != ErrRejected {
			t.Fatal("bad local config accepted")
		}
	}
	var nilV *Verifier
	if _, e := nilV.Verify(context.Background(), nil); e != ErrRejected {
		t.Fatal(e)
	}
	if _, e := verifier(t, s).Verify(nil, nil); e != ErrRejected {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := verifier(t, s).Verify(ctx, nil); e != ErrRejected {
		t.Fatal(e)
	}
}
func TestES256CardAndStrictSignatures(t *testing.T) {
	m, s, _ := fixture(t)
	priv, e := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	s.record.Keys[0].Alg = sage.AlgP256
	s.record.Keys[0].Material, e = priv.PublicKey.Bytes()
	if e != nil {
		t.Fatal(e)
	}
	payload, _ := jcs.Marshal(m)
	protected := base64.RawURLEncoding.EncodeToString(encode(map[string]string{"alg": "ES256", "kid": peer + "#sign", "typ": CardType}))
	input := []byte(protected + "." + base64.RawURLEncoding.EncodeToString(payload))
	h := sha256.Sum256(input)
	r, z, e := ecdsa.Sign(rand.Reader, priv, h[:])
	if e != nil {
		t.Fatal(e)
	}
	n := elliptic.P256().Params().N
	if z.Cmp(new(big.Int).Rsh(new(big.Int).Set(n), 1)) > 0 {
		z.Sub(n, z)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	z.FillBytes(sig[32:])
	m["signatures"] = []any{map[string]any{"protected": protected, "signature": base64.RawURLEncoding.EncodeToString(sig)}}
	if _, e := verifier(t, s).Verify(context.Background(), encode(m)); e != nil {
		t.Fatal(e)
	}
	high := new(big.Int).Sub(n, z)
	high.FillBytes(sig[32:])
	m["signatures"].([]any)[0].(map[string]any)["signature"] = base64.RawURLEncoding.EncodeToString(sig)
	rejectCard(t, s, encode(m))
	if verifySignature("ES256", s.record.Keys[0], input, []byte{1}) {
		t.Fatal("short accepted")
	}
	s.record.Keys[0].Material = []byte{1}
	if verifySignature("ES256", s.record.Keys[0], input, make([]byte, 64)) {
		t.Fatal("bad point accepted")
	}
	for _, b := range [][]byte{nil, make([]byte, 32), append([]byte{1}, make([]byte, 31)...), bytes.Repeat([]byte{255}, 32)} {
		if prime(b) {
			t.Fatal("invalid Ed25519 point accepted")
		}
	}
}
func TestExactProtocolIntegerInputs(t *testing.T) {
	for raw, want := range map[string]int64{"100.0": 100, "1e2": 100, "9007199254740991": 9007199254740991, "-9007199254740991": -9007199254740991, "1000e-2": 10, "0": 0} {
		got, ok := integerSigned(json.Number(raw))
		if !ok || got != want {
			t.Fatalf("%s %d %v", raw, got, ok)
		}
	}
	for _, raw := range []string{"1.5", "-0", "9007199254740992", "-9007199254740992", "1e-1000", "1e1000", "1e999999999999999999999999", "1/2"} {
		if _, ok := integerSigned(json.Number(raw)); ok {
			t.Fatal("noninteger or range accepted", raw)
		}
	}
	if _, ok := integerSigned("1"); ok {
		t.Fatal("string accepted")
	}
}
func TestInterfaceEligibilityPreservesURIBytes(t *testing.T) {
	for _, target := range []string{"HTTPS://agents.example.com/a2a", "https://agents.example.com:443/a2a", "https://agents.example.com/a2a/"} {
		m, s, k := fixture(t)
		m["supportedInterfaces"].([]any)[0].(map[string]any)["url"] = target
		rejectCard(t, s, signed(t, m, k))
		s.record.Services[0].URI = target
		if _, e := verifier(t, s).Verify(context.Background(), signed(t, m, k)); e != nil {
			t.Fatal("exact eligible URI rejected", target, e)
		}
	}
}
