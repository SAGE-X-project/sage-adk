package a2abinding

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// Inert production-path runtime only: no changed signatures, ciphertext,
// identities, replay probes or external destinations are sent over this TLS.
func TestProtectedHTTPRuntime010(t *testing.T) {
	for _, httpHandshake := range []bool{false, true} {
		for _, mode := range []string{"success", "repeat", "unsupported", "source-after-setup", "expired-after-setup", "cancel-after-setup", "close-after-setup", "source-after-handler", "expired-after-handler", "close-during-handler", "cancel-during-handler"} {
			t.Run(mode+map[bool]string{false: "/bare-completion", true: "/http-completion"}[httpHandshake], func(t *testing.T) {
				listener, e := net.Listen("tcp", "127.0.0.1:0")
				if e != nil {
					t.Fatal(e)
				}
				t.Cleanup(func() { _ = listener.Close() })
				target := "https://" + listener.Addr().String() + "/a%2Fa?x=%2f"
				pair := protectedSessions(t, target, httpHandshake)
				cert, roots := protectedCertificate(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cfg := HTTPTransportConfig{Session: pair.left, URL: target, TLS: &tls.Config{RootCAs: roots}, TTL: 30, Timeout: 3 * time.Second}
				owner, e := NewHTTPTransportOwner(cfg)
				if e != nil {
					t.Fatal(e)
				}
				t.Cleanup(owner.Close)
				effects := atomic.Int32{}
				want := []byte(" {\"jsonrpc\":\"2.0\",\"id\":\"one\",\"method\":\"GetTask\",\"params\":{\"id\":\"task\"}} ")
				if mode == "unsupported" {
					want = rpc("SendStreamingMessage", map[string]any{"message": map[string]any{}})
				}
				card := pair.card(t)
				server, e := NewServer(ServerConfig{DID: peer, URL: target, Verifier: pair.verifier(t), Card: func(context.Context) ([]byte, error) { return card, nil }, Handler: handlerFunc(func(_ context.Context, payload []byte) ([]byte, error) {
					effects.Add(1)
					if !bytes.Equal(payload, want) {
						t.Error("real Handler lost exact request bytes")
					}
					if mode == "close-during-handler" {
						owner.Close()
					}
					if mode == "cancel-during-handler" {
						cancel()
					}
					if mode == "source-after-handler" {
						pair.fixture.setMode("read-error")
					}
					if mode == "expired-after-handler" {
						pair.fixture.advance(6000, 6)
					}
					return reply(), nil
				})})
				if e != nil {
					t.Fatal(e)
				}
				setup := atomic.Int32{}
				serverTLS := &tls.Config{GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
					setup.Add(1)
					switch mode {
					case "source-after-setup":
						pair.fixture.setMode("read-error")
					case "expired-after-setup":
						pair.fixture.advance(6000, 6)
					case "cancel-after-setup":
						cancel()
					case "close-after-setup":
						owner.Close()
					}
					return &cert, nil
				}}
				ingress, e := NewHTTPIngress(HTTPIngressConfig{Transport: HTTPTransportConfig{Session: pair.right, URL: target, TLS: serverTLS, TTL: 30, Timeout: 3 * time.Second}, Server: server})
				if e != nil {
					t.Fatal(e)
				}
				serveCtx, stopServe := context.WithCancel(context.Background())
				done := make(chan error, 1)
				go func() { done <- ingress.ServeTLS(serveCtx, listener) }()
				t.Cleanup(func() {
					stopServe()
					ingress.Close()
					select {
					case <-done:
					case <-time.After(5 * time.Second):
						t.Error("owned TLS listener did not stop")
					}
				})
				client, e := NewClient(pair.verifier(t), owner)
				if e != nil {
					t.Fatal(e)
				}
				out, e := client.Call(ctx, card, want)
				good := mode == "success" || mode == "repeat" || mode == "unsupported"
				if good {
					if e != nil {
						t.Fatal("real protected TLS exchange", e)
					}
					if (mode == "success" || mode == "repeat") && (!bytes.Equal(out, reply()) || effects.Load() != 1) {
						t.Fatal("exact result/effect lost", string(out), effects.Load())
					}
					if mode == "unsupported" {
						var response struct {
							Error struct {
								Code int `json:"code"`
							} `json:"error"`
						}
						if json.Unmarshal(out, &response) != nil || response.Error.Code != -32004 || effects.Load() != 0 {
							t.Fatal("unsupported method was not an authenticated JSONRPC error", string(out), effects.Load())
						}
					}
					if mode == "repeat" {
						copied := *owner
						second, err := NewClient(pair.verifier(t), &copied)
						if err != nil {
							t.Fatal(err)
						}
						again, err := second.Call(ctx, card, want)
						if err != nil || !bytes.Equal(again, reply()) || effects.Load() != 2 {
							t.Fatal("next explicit call was closed by old cancellation", err, effects.Load())
						}
					}
					pair.leftReplay.mu.Lock()
					left := pair.leftReplay.accepted
					pair.leftReplay.mu.Unlock()
					pair.rightReplay.mu.Lock()
					right := pair.rightReplay.accepted
					pair.rightReplay.mu.Unlock()
					expected := 1
					if mode == "repeat" {
						expected = 2
					}
					if left != expected || right != expected {
						t.Fatal("missing real request/response acceptance", left, right)
					}
				} else {
					if e != ErrRejected || out != nil {
						t.Fatal("unavailable/expired exchange returned usable result", e, string(out))
					}
					expected := int32(0)
					if mode == "source-after-handler" || mode == "expired-after-handler" || mode == "close-during-handler" || mode == "cancel-during-handler" {
						expected = 1
					}
					if effects.Load() != expected {
						t.Fatal("unexpected effect or automatic retry", effects.Load(), expected)
					}
					if !owner.state.closing.Load() {
						t.Fatal("failed exchange owner remains usable")
					}
				}
				expectedSetups := int32(1)
				if mode == "repeat" {
					expectedSetups = 2
				}
				if setup.Load() != expectedSetups {
					t.Fatal("missing actual owned TLS setup or retry", setup.Load())
				}
				t.Logf("owned-production-http handshake=%t mode=%s handler=%d usable-result=%t", httpHandshake, mode, effects.Load(), good)
			})
		}
	}
}
