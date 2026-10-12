package a2abinding

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"github.com/sage-x-project/did-go/advanced/sage"
	didgo "github.com/sage-x-project/did-go/did"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"testing"
)

const peer = "did:sage:web:agents.example.com:billing-bot"
const endpoint = "https://agents.example.com/a2a"

type source struct {
	record sage.Record
	err    error
	calls  int
}

func (s *source) Observe(context.Context, didgo.DID) (sage.Record, error) {
	s.calls++
	return s.record, s.err
}

type clock struct{ ticks []registry010.Stamp }

func (c *clock) Now() (registry010.Stamp, error) {
	if len(c.ticks) == 0 {
		return registry010.Stamp{MonoMS: 100, Unix: 101}, nil
	}
	t := c.ticks[0]
	c.ticks = c.ticks[1:]
	return t, nil
}
func fixture(t *testing.T) (map[string]any, *source, ed25519.PrivateKey) {
	t.Helper()
	key := ed25519.NewKeyFromSeed(make([]byte, 32))
	d, e := didgo.Parse(peer)
	if e != nil {
		t.Fatal(e)
	}
	r := sage.Record{DID: d, Controller: "operator", State: sage.StateActive, Version: "3", Keys: []sage.Key{{Name: "sign", Alg: sage.AlgEd25519, State: sage.KeyAccepted, Material: key.Public().(ed25519.PublicKey)}}, Services: []sage.Service{{Name: "a2a", Type: "A2AInterface", URI: endpoint}}, Observation: sage.Observation{Issued: new(int64(100)), Source: "fixture-only", Time: 101}}
	return map[string]any{"name": "billing", "description": "", "version": "1", "supportedInterfaces": []any{map[string]any{"url": endpoint, "protocolBinding": BindingURI, "protocolVersion": "1.0"}}, "capabilities": map[string]any{"streaming": false, "extensions": []any{map[string]any{"uri": ExtensionURI, "required": true, "params": map[string]any{"sageVersion": "0.10.0", "did": peer, "recordVersion": "3", "issued": 100, "expires": 105}}}}, "defaultInputModes": []any{"application/json"}, "defaultOutputModes": []any{"application/json"}, "skills": []any{map[string]any{"id": "billing", "name": "billing", "description": "", "tags": []any{"billing"}}}}, &source{record: r}, key
}
func signed(t *testing.T, m map[string]any, key ed25519.PrivateKey) []byte {
	t.Helper()
	delete(m, "signatures")
	payload, e := jcs.Marshal(m)
	if e != nil {
		t.Fatal(e)
	}
	h, _ := json.Marshal(map[string]string{"alg": "Ed25519", "kid": peer + "#sign", "typ": CardType})
	enc := base64.RawURLEncoding.EncodeToString
	protected := enc(h)
	sig := ed25519.Sign(key, []byte(protected+"."+enc(payload)))
	m["signatures"] = []any{map[string]any{"protected": protected, "signature": enc(sig)}}
	b, _ := json.Marshal(m)
	return b
}
func TestVerifiedCardBindsExactPeerAndRecord(t *testing.T) {
	m, s, k := fixture(t)
	v, e := NewVerifier(VerifierConfig{Source: s, Clock: &clock{}, ExpectedPeer: peer})
	if e != nil {
		t.Fatal(e)
	}
	c, e := v.Verify(context.Background(), signed(t, m, k))
	if e != nil {
		t.Fatal(e)
	}
	if c.DID() != peer || c.RecordVersion() != "3" || len(c.Interfaces()) != 1 || c.Interfaces()[0].URL != endpoint || s.calls != 1 {
		t.Fatal("card binding lost")
	}
}
func TestShapeAndIdentityFailuresDeny(t *testing.T) {
	for _, name := range []string{"version", "binding", "extension", "expiry", "record", "peer", "service", "revoked", "created"} {
		t.Run(name, func(t *testing.T) {
			m, s, k := fixture(t)
			want := peer
			switch name {
			case "version":
				m["supportedInterfaces"].([]any)[0].(map[string]any)["protocolVersion"] = "0.3"
			case "binding":
				m["supportedInterfaces"].([]any)[0].(map[string]any)["protocolBinding"] = "JSONRPC"
			case "extension":
				m["capabilities"].(map[string]any)["extensions"].([]any)[0].(map[string]any)["required"] = false
			case "expiry":
				m["capabilities"].(map[string]any)["extensions"].([]any)[0].(map[string]any)["params"].(map[string]any)["expires"] = 101
			case "record":
				s.record.Version = "4"
			case "peer":
				want = "did:sage:web:agents.example.com:other"
			case "service":
				s.record.Services[0].URI = endpoint + "/"
			case "revoked":
				s.record.Keys[0].State = sage.KeyRevoked
			case "created":
				s.record.State = sage.StateCreated
			}
			v, e := NewVerifier(VerifierConfig{Source: s, Clock: &clock{}, ExpectedPeer: want})
			if e != nil {
				t.Fatal(e)
			}
			if _, e = v.Verify(context.Background(), signed(t, m, k)); e != ErrRejected {
				t.Fatalf("want generic rejection, got %v", e)
			}
		})
	}
}
