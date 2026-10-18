package consumer_test

import (
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	advanced "github.com/sage-x-project/did-go/advanced/sage"
	"github.com/sage-x-project/did-go/consumer"
	"github.com/sage-x-project/did-go/did"
	"github.com/sage-x-project/did-go/producer"
	"github.com/sage-x-project/did-go/resolution"
	"github.com/sage-x-project/sage-adk/core/a2abinding"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"github.com/sage-x-project/sage/pkg/agent/didsource010"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

const adoptionRegistry = "eip155:11155111:0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const adoptionID = "did:sage:" + adoptionRegistry + ":billing-bot"
const adoptionURL = "https://agents.example.com/a2a"

type adoptionClock struct{}

func (adoptionClock) Now() (registry010.Stamp, error) {
	return registry010.Stamp{MonoMS: 100, Unix: 101}, nil
}

type adoptionReader struct {
	snapshot registry010.RecordSnapshot
	calls    atomic.Int32
}

func (r *adoptionReader) ReadRecord(context.Context, string) (registry010.RecordSnapshot, error) {
	r.calls.Add(1)
	return r.snapshot, nil
}

// The record and local clock are test data, not a selected deployment authority.
func adoptionSource(t *testing.T, state string) (*didsource010.EIP155Source, *adoptionReader, ed25519.PrivateKey) {
	t.Helper()
	enc := base64.RawURLEncoding.EncodeToString
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	kem, err := ecdh.X25519().NewPrivateKey(append([]byte{7}, make([]byte, 31)...))
	if err != nil {
		t.Fatal(err)
	}
	keys := []any{}
	for _, k := range []struct {
		name, alg string
		public    []byte
	}{
		{"a-ed", "ed25519", private.Public().(ed25519.PublicKey)},
		{"b-kem", "x25519", kem.PublicKey().Bytes()},
	} {
		challenge, err := registry010.PoPChallenge010(adoptionRegistry, "billing-bot", k.name, k.alg, k.public)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, map[string]any{"name": k.name, "alg": k.alg, "key": enc(k.public), "state": "accepted", "proof": map[string]any{"signer": adoptionID + "#a-ed", "value": enc(ed25519.Sign(private, challenge))}})
	}
	record, err := json.Marshal(map[string]any{"id": adoptionID, "controller": "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "state": state, "version": "3", "keys": keys, "services": []any{map[string]any{"name": "c-a2a", "type": "A2AInterface", "uri": adoptionURL}}})
	if err != nil {
		t.Fatal(err)
	}
	hash := strings.Repeat("a", 64)
	reader := &adoptionReader{snapshot: registry010.RecordSnapshot{Source: "fixture-only", Registry: adoptionRegistry, Network: "fixture", Record: record, BlockHash: hash, RecordBlockHash: hash, KeysBlockHash: hash, ServicesBlockHash: hash, AcquiredMS: 100, Ready: true, Validated: true, Finalized: true}}
	journal, err := registry010.OpenJournal(filepath.Join(t.TempDir(), "history"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := journal.Close(); err != nil {
			t.Error(err)
		}
	})
	source, err := didsource010.NewEIP155Source(registry010.Config{Source: "fixture-only", Registry: adoptionRegistry, Network: "fixture", Blockchain: true}, reader, adoptionClock{}, journal)
	if err != nil {
		t.Fatal(err)
	}
	return source, reader, private
}

func adoptionDID(t *testing.T) did.DID {
	t.Helper()
	d, err := did.Parse(adoptionID)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestAdoptedDIDMetadata(t *testing.T) {
	for _, state := range []string{"active", "created", "deactivated"} {
		t.Run(state, func(t *testing.T) {
			source, reader, _ := adoptionSource(t, state)
			response := advanced.Respond(t.Context(), source, advanced.Request{DID: adoptionDID(t), Accept: advanced.MediaTypeEnvelope}, producer.New())
			var result map[string]json.RawMessage
			if response.Problem != nil || json.Unmarshal(response.Body, &result) != nil || reader.calls.Load() != 1 {
				t.Fatalf("response %+v; reads=%d", response, reader.calls.Load())
			}
			var md map[string]json.RawMessage
			if json.Unmarshal(result["didDocumentMetadata"], &md) != nil {
				t.Fatal("missing metadata")
			}
			want := "false"
			if state == "deactivated" {
				want = "true"
			}
			if string(md["deactivated"]) != want || string(md["versionId"]) != `"3"` || md["sage"] == nil {
				t.Fatalf("metadata %s", result["didDocumentMetadata"])
			}
		})
	}
}

func TestAdoptedDIDTLS(t *testing.T) {
	for _, verb := range []string{http.MethodGet, http.MethodPost} {
		for _, name := range []string{"envelope", "unsupported"} {
			t.Run(verb+"/"+name, func(t *testing.T) {
				source, reader, _ := adoptionSource(t, "active")
				r, err := advanced.NewResolver(source, consumer.New(), producer.New())
				if err != nil {
					t.Fatal(err)
				}
				server := httptest.NewTLSServer(advanced.NewHandler(r, "/identifiers/"))
				defer server.Close()
				req, err := http.NewRequestWithContext(t.Context(), verb, server.URL+"/identifiers/"+adoptionID, strings.NewReader("{}"))
				if err != nil {
					t.Fatal(err)
				}
				accept := "application/did-resolution"
				if name == "unsupported" {
					accept = "text/html"
				}
				req.Header.Set("Accept", accept)
				req.Header.Set("Content-Type", "application/json")
				response, err := server.Client().Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := response.Body.Close(); err != nil {
						t.Error(err)
					}
				}()
				var wire map[string]json.RawMessage
				if err := json.NewDecoder(response.Body).Decode(&wire); err != nil {
					t.Fatal(err)
				}
				if name == "unsupported" {
					if response.StatusCode != http.StatusNotImplemented || response.Header.Get("Content-Type") != "application/problem+json" || string(wire["sageCode"]) != `"representation.unsupported"` || string(wire["type"]) != `"`+resolution.CodeFeatureNotSupported.TypeURL()+`"` || reader.calls.Load() != 0 || wire["didDocument"] != nil {
						t.Fatalf("problem %v; reads=%d", wire, reader.calls.Load())
					}
				} else {
					var md map[string]json.RawMessage
					if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != accept || reader.calls.Load() != 1 || json.Unmarshal(wire["didDocumentMetadata"], &md) != nil || string(md["deactivated"]) != "false" {
						t.Fatalf("envelope %v; reads=%d", wire, reader.calls.Load())
					}
				}
			})
		}
	}
}

func TestAdoptedDIDCardBinding(t *testing.T) {
	for _, state := range []string{"active", "created", "deactivated"} {
		t.Run(state, func(t *testing.T) {
			source, reader, private := adoptionSource(t, state)
			card := map[string]any{"name": "billing", "description": "", "version": "1", "supportedInterfaces": []any{map[string]any{"url": adoptionURL, "protocolBinding": a2abinding.BindingURI, "protocolVersion": "1.0"}}, "capabilities": map[string]any{"streaming": false, "extensions": []any{map[string]any{"uri": a2abinding.ExtensionURI, "required": true, "params": map[string]any{"sageVersion": "0.10.0", "did": adoptionID, "recordVersion": "3", "issued": 100, "expires": 105}}}}, "defaultInputModes": []any{"application/json"}, "defaultOutputModes": []any{"application/json"}, "skills": []any{map[string]any{"id": "billing", "name": "billing", "description": "", "tags": []any{"billing"}}}}
			payload, err := jcs.Marshal(card)
			if err != nil {
				t.Fatal(err)
			}
			header, err := json.Marshal(map[string]string{"alg": "Ed25519", "kid": adoptionID + "#a-ed", "typ": a2abinding.CardType})
			if err != nil {
				t.Fatal(err)
			}
			enc := base64.RawURLEncoding.EncodeToString
			protected := enc(header)
			card["signatures"] = []any{map[string]any{"protected": protected, "signature": enc(ed25519.Sign(private, []byte(protected+"."+enc(payload))))}}
			raw, err := json.Marshal(card)
			if err != nil {
				t.Fatal(err)
			}
			verifier, err := a2abinding.NewVerifier(a2abinding.VerifierConfig{Source: source, Clock: adoptionClock{}, ExpectedPeer: adoptionID})
			if err != nil {
				t.Fatal(err)
			}
			verified, err := verifier.Verify(t.Context(), raw)
			if reader.calls.Load() != 1 {
				t.Fatalf("fresh observations=%d", reader.calls.Load())
			}
			if state != "active" {
				if err != a2abinding.ErrRejected || verified != nil {
					t.Fatalf("inactive discovery became eligible: %v", err)
				}
			} else if err != nil || verified == nil || verified.DID() != adoptionID || verified.RecordVersion() != "3" || len(verified.Interfaces()) != 1 || verified.Interfaces()[0].URL != adoptionURL {
				t.Fatalf("card binding %v %v", verified, err)
			}
		})
	}
}
