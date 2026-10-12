package a2abinding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// This owner is a test-only TLS fixture, not a chapter 03/08 transport adapter.
// It tests policy wiring. It makes no inner/outer signature or replay claim.
type tlsFixtureOwner struct {
	http  *http.Client
	calls atomic.Int32
}

func (o *tlsFixtureOwner) Exchange(ctx context.Context, c ProtectedCall, check func(context.Context) error) (AuthenticatedResponse, error) {
	o.calls.Add(1)
	if e := check(ctx); e != nil {
		return AuthenticatedResponse{}, e
	}
	frame := map[string]any{"scope": "fixture-policy-only", "recipient": c.Recipient(), "metadata": c.Metadata(), "payload": c.Payload()}
	req, e := http.NewRequestWithContext(ctx, "POST", c.Target().URL, bytes.NewReader(encode(frame)))
	if e != nil {
		return AuthenticatedResponse{}, e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("A2A-Version", "0.3") // ignored: only signed-owner metadata is consumed
	response, e := o.http.Do(req)
	if e != nil {
		return AuthenticatedResponse{}, e
	}
	defer response.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(response.Body, 1024))
	if e != nil || response.StatusCode != 200 {
		return AuthenticatedResponse{}, errors.New("fixture refused")
	}
	var out AuthenticatedResponse
	e = json.Unmarshal(raw, &out)
	return out, e
}
func TestRuntimeTLSFixtureUsesOnlyProtectedPolicyAndExactPayload(t *testing.T) {
	m, clientSource, k := fixture(t)
	_, serverSource, _ := fixture(t)
	var rawCard []byte
	var policy *Server
	effects := atomic.Int32{}
	var expected []byte
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || r.Method != "POST" {
			t.Error("fixture TLS/method lost")
			w.WriteHeader(500)
			return
		}
		var frame struct {
			Scope     string
			Recipient string
			Metadata  map[string]string
			Payload   []byte
		}
		if json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&frame) != nil || frame.Scope != "fixture-policy-only" {
			w.WriteHeader(400)
			return
		}
		out, e := policy.Handle(r.Context(), AuthenticatedRequest{URL: "https://" + r.Host + r.URL.Path, Recipient: frame.Recipient, Metadata: frame.Metadata, Payload: frame.Payload})
		if e != nil {
			w.WriteHeader(403)
			return
		}
		json.NewEncoder(w).Encode(AuthenticatedResponse{Sender: peer, Payload: out})
	}))
	defer ts.Close()
	target := ts.URL + "/a2a"
	m["supportedInterfaces"].([]any)[0].(map[string]any)["url"] = target
	clientSource.record.Services[0].URI = target
	serverSource.record.Services[0].URI = target
	rawCard = signed(t, m, k)
	var e error
	policy, e = NewServer(ServerConfig{DID: peer, URL: target, Verifier: verifier(t, serverSource), Card: func(context.Context) ([]byte, error) { return rawCard, nil }, Handler: handlerFunc(func(_ context.Context, raw []byte) ([]byte, error) {
		effects.Add(1)
		if !bytes.Equal(raw, expected) {
			t.Error("payload re-encoded")
		}
		return reply(), nil
	})})
	if e != nil {
		t.Fatal(e)
	}
	owner := &tlsFixtureOwner{http: ts.Client()}
	c, _ := NewClient(verifier(t, clientSource), owner)
	expected = []byte("{\"jsonrpc\":\"2.0\", \"id\":\"one\",\"method\":\"GetTask\",\"params\":{\"id\":\"task\"}}")
	out, e := c.Call(context.Background(), rawCard, expected)
	if e != nil || !bytes.Equal(out, reply()) || effects.Load() != 1 || owner.calls.Load() != 1 {
		t.Fatal(e, effects.Load(), owner.calls.Load())
	}
	serverSource.record.Version = "4"
	if _, e := c.Call(context.Background(), rawCard, expected); e != ErrRejected || effects.Load() != 1 || owner.calls.Load() != 2 {
		t.Fatal("stale card caused effect/retry", e)
	}
	if clientSource.calls != 4 || serverSource.calls != 3 {
		t.Fatal("missing fresh observations", clientSource.calls, serverSource.calls)
	}
}
