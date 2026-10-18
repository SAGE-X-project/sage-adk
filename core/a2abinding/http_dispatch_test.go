package a2abinding

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestProtectedHTTPDispatchHook010(t *testing.T) {
	for _, mode := range []string{"once", "missing", "twice", "hook-error", "ignored-verify-error"} {
		t.Run(mode, func(t *testing.T) {
			m, source, key := fixture(t)
			card := signed(t, m, key)
			effects := 0
			server, e := NewServer(ServerConfig{DID: peer, URL: endpoint, Verifier: verifier(t, source), Card: func(context.Context) ([]byte, error) { return card, nil }, Handler: handlerFunc(func(context.Context, []byte) ([]byte, error) { effects++; return reply(), nil })})
			if e != nil {
				t.Fatal(e)
			}
			final := func(_ context.Context, verify func() (*VerifiedCard, error)) error {
				if mode == "missing" {
					return nil
				}
				if mode == "ignored-verify-error" {
					source.record.Version = "4"
				}
				_, err := verify()
				if mode == "twice" {
					_, _ = verify()
					return nil
				}
				if mode == "ignored-verify-error" {
					return nil
				}
				if mode == "hook-error" {
					return ErrRejected
				}
				return err
			}
			out, e := server.handle(context.Background(), AuthenticatedRequest{URL: endpoint, Recipient: peer, Metadata: metadata(), Payload: rpc("GetTask", map[string]any{"id": "task"})}, final)
			if mode == "once" {
				if e != nil || effects != 1 || string(out) != string(reply()) {
					t.Fatal("actual Handler not admitted", e, effects)
				}
			} else if e != ErrRejected || effects != 0 || out != nil {
				t.Fatal("skipped/failed/duplicate final verification reached Handler", e, effects)
			}
		})
	}
}

func TestProtectedHTTPDispatchBudgetIncludesVerify010(t *testing.T) {
	pair := protectedSessions(t, endpoint, true)
	owned, e := pair.right.TakeHTTPApplication(endpoint)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(owned.Close)
	state := &protectedHTTPState{owner: owned}
	card := pair.card(t)
	effects := 0
	server, e := NewServer(ServerConfig{DID: peer, URL: endpoint, Verifier: pair.verifier(t), Card: func(context.Context) ([]byte, error) { return card, nil }, Handler: handlerFunc(func(context.Context, []byte) ([]byte, error) { effects++; return reply(), nil })})
	if e != nil {
		t.Fatal(e)
	}
	_, e = server.handle(context.Background(), AuthenticatedRequest{URL: endpoint, Recipient: peer, Metadata: metadata(), Payload: rpc("GetTask", map[string]any{"id": "task"})}, func(ctx context.Context, verify func() (*VerifiedCard, error)) error {
		return state.round(ctx, func() (*VerifiedCard, error) { pair.fixture.advance(5001, 0); return verify() }, wireWindow{Created: 100, Expires: 130})
	})
	if e != ErrRejected || effects != 0 {
		t.Fatal("final verification time omitted from admission interval", e, effects)
	}
}

func TestProtectedHTTPConstructionAndClosure010(t *testing.T) {
	for _, mode := range []string{"nil-session", "wrong-target", "ttl-zero", "ttl-large", "timeout-zero", "timeout-large", "tls-nil", "server-name", "bad-tls"} {
		t.Run(mode, func(t *testing.T) {
			pair := protectedSessions(t, endpoint, true)
			cfg := HTTPTransportConfig{Session: pair.left, URL: endpoint, TLS: &tls.Config{}, TTL: 30, Timeout: time.Second}
			switch mode {
			case "nil-session":
				cfg.Session = nil
			case "wrong-target":
				cfg.URL = "http://agents.example.com/a2a"
			case "ttl-zero":
				cfg.TTL = 0
			case "ttl-large":
				cfg.TTL = 301
			case "timeout-zero":
				cfg.Timeout = 0
			case "timeout-large":
				cfg.Timeout = time.Minute + 1
			case "tls-nil":
				cfg.TLS = nil
			case "server-name":
				cfg.TLS.ServerName = "elsewhere.example"
			case "bad-tls":
				cfg.TLS.InsecureSkipVerify = true
			}
			if owner, e := NewHTTPTransportOwner(cfg); e != ErrRejected || owner != nil {
				t.Fatal("invalid assembly accepted", e)
			}
			// All configuration checks above precede transfer; a valid constructor
			// can still take this same unused completed session once.
			owner, e := NewHTTPTransportOwner(HTTPTransportConfig{Session: pair.left, URL: endpoint, TLS: &tls.Config{}, TTL: 30, Timeout: time.Second})
			if e != nil {
				t.Fatal("failed configuration consumed ownership", e)
			}
			owner.Close()
			if _, e := NewHTTPTransportOwner(HTTPTransportConfig{Session: pair.left, URL: endpoint, TLS: &tls.Config{}, TTL: 30, Timeout: time.Second}); e != ErrRejected {
				t.Fatal("old session reclaimed after transfer")
			}
		})
	}
	var noOwner *HTTPTransportOwner
	if _, e := noOwner.Exchange(context.Background(), ProtectedCall{}, func(context.Context) error { return nil }); e != ErrRejected {
		t.Fatal("nil transport accepted")
	}
	if _, e := (&HTTPTransportOwner{}).Exchange(context.Background(), ProtectedCall{}, func(context.Context) error { return nil }); e != ErrRejected {
		t.Fatal("zero transport accepted")
	}
	if _, e := NewHTTPIngress(HTTPIngressConfig{}); e != ErrRejected {
		t.Fatal("missing ingress assembly accepted")
	}
	pair := protectedSessions(t, endpoint, true)
	card := pair.card(t)
	server, e := NewServer(ServerConfig{DID: peer, URL: endpoint, Verifier: pair.verifier(t), Card: func(context.Context) ([]byte, error) { return card, nil }, Handler: handlerFunc(func(context.Context, []byte) ([]byte, error) { return reply(), nil })})
	if e != nil {
		t.Fatal(e)
	}
	cert, _ := protectedCertificate(t)
	cfg := HTTPIngressConfig{Transport: HTTPTransportConfig{Session: pair.right, URL: endpoint, TLS: &tls.Config{Certificates: []tls.Certificate{cert}}, TTL: 30, Timeout: time.Second}, Server: server}
	withDialer := cfg
	withDialer.Transport.Dialer = &net.Dialer{}
	if _, e := NewHTTPIngress(withDialer); e != ErrRejected {
		t.Fatal("server accepted client dialing configuration")
	}
	ingress, e := NewHTTPIngress(cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(ingress.Close)
	if ingress.ServeTLS(context.Background(), nil) != ErrRejected {
		t.Fatal("nil listener accepted")
	}
	var missing *HTTPIngress
	if missing.ServeTLS(context.Background(), nil) != ErrRejected {
		t.Fatal("nil ingress accepted")
	}
	ingress.Close()
}

// These refusal controls use only in-memory listeners/connections. They never
// emit malformed traffic or substitute for the real TLS runtime scenarios.
type protectedMemoryListener struct{ accepts, closes int }

func (l *protectedMemoryListener) Accept() (net.Conn, error) { l.accepts++; return nil, ErrRejected }
func (l *protectedMemoryListener) Close() error              { l.closes++; return nil }
func (l *protectedMemoryListener) Addr() net.Addr            { return &net.TCPAddr{} }

type protectedDeadlineConn struct{ protectedMemoryConn }

func (c *protectedDeadlineConn) SetDeadline(time.Time) error { return ErrRejected }

type protectedCapturedConn struct {
	protectedMemoryConn
	body bytes.Buffer
}

func (c *protectedCapturedConn) Write(p []byte) (int, error) { c.writes.Add(1); return c.body.Write(p) }

func TestProtectedHTTPLocalIngressRefusals010(t *testing.T) {
	for _, mode := range []string{"consumed-session", "wrong-local"} {
		t.Run(mode, func(t *testing.T) {
			pair := protectedSessions(t, endpoint, true)
			local := peer
			if mode == "wrong-local" {
				local = protectedAlice
			} else {
				pair.right.Close()
			}
			server, e := NewServer(ServerConfig{DID: local, URL: endpoint, Verifier: pair.verifier(t), Card: func(context.Context) ([]byte, error) { return nil, ErrRejected }, Handler: handlerFunc(func(context.Context, []byte) ([]byte, error) {
				t.Error("refused assembly reached Handler")
				return nil, ErrRejected
			})})
			if e != nil {
				t.Fatal(e)
			}
			cert, _ := protectedCertificate(t)
			ingress, e := NewHTTPIngress(HTTPIngressConfig{Transport: HTTPTransportConfig{Session: pair.right, URL: endpoint, TLS: &tls.Config{Certificates: []tls.Certificate{cert}}, TTL: 30, Timeout: time.Second}, Server: server})
			if e != ErrRejected || ingress != nil {
				t.Fatal("consumed or wrong-local session accepted", e)
			}
			if owner, e := pair.right.TakeHTTPApplication(endpoint); e == nil || owner != nil {
				t.Fatal("refused transferred session was reusable")
			}
		})
	}
	for _, mode := range []string{"already-serving", "closed"} {
		t.Run(mode, func(t *testing.T) {
			state := &protectedIngressState{transport: &protectedHTTPState{}}
			if mode == "already-serving" {
				state.serving.Store(true)
			} else {
				state.transport.closing.Store(true)
			}
			listener := &protectedMemoryListener{}
			if (&HTTPIngress{state: state}).ServeTLS(context.Background(), listener) != ErrRejected || listener.accepts != 0 {
				t.Fatal("invalid serving state accepted a connection")
			}
			wantClosed := 0
			if mode == "closed" {
				wantClosed = 1
			}
			if listener.closes != wantClosed {
				t.Fatal("listener ownership did not follow admission", listener.closes)
			}
		})
	}
	for _, mode := range []string{"closed-connection", "canceled-connection", "deadline-error", "handshake-eof"} {
		t.Run(mode, func(t *testing.T) {
			state := &protectedIngressState{transport: &protectedHTTPState{timeout: time.Second}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			conn := &protectedDeadlineConn{}
			if mode == "closed-connection" {
				state.transport.closing.Store(true)
			}
			if mode == "canceled-connection" {
				cancel()
			}
			var raw net.Conn = conn
			if mode == "handshake-eof" {
				raw = &conn.protectedMemoryConn
				cert, _ := protectedCertificate(t)
				state.transport.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
			}
			state.connection(ctx, raw)
			if conn.closes.Load() != 1 || conn.writes.Load() != 0 || state.transport.conn != nil {
				t.Fatal("refused connection leaked or wrote application bytes")
			}
		})
	}
	t.Run("empty-response", func(t *testing.T) {
		if frame, e := protectedResponseFrame(hpke.HTTPMessage010{Status: http.StatusOK}); e != ErrRejected || frame != nil {
			t.Fatal("empty response accepted")
		}
	})
	for _, status := range []int{http.StatusBadRequest, http.StatusOK} {
		name := "unsigned-bad-request"
		if status == http.StatusOK {
			name = "unsigned-unauthorized"
		}
		t.Run(name, func(t *testing.T) {
			conn := &protectedCapturedConn{}
			protectedAuthenticationFailure(conn, status)
			response, e := http.ReadResponse(bufio.NewReader(&conn.body), nil)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = response.Body.Close() }()
			want := http.StatusUnauthorized
			if status == http.StatusBadRequest {
				want = status
			}
			body, e := io.ReadAll(response.Body)
			if e != nil || response.StatusCode != want || !response.Close || string(body) != "authentication failed\n" || conn.writes.Load() != 1 {
				t.Fatal("generic refusal became a result or retry", e, response.StatusCode, string(body))
			}
		})
	}
}

// Memory TLS gives deterministic policy/terminal-state observations without a
// network listener or a configurable destination. Refusal inputs stay unit-only.
func TestProtectedHTTPMemoryTLS010(t *testing.T) {
	for _, mode := range []string{"policy-denial-reuse", "card-unavailable-after-handler"} {
		t.Run(mode, func(t *testing.T) {
			pair := protectedSessions(t, endpoint, true)
			left, e := pair.left.TakeHTTPApplication(endpoint)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(left.Close)
			card := pair.card(t)
			effects, cardCalls := 0, 0
			server, e := NewServer(ServerConfig{DID: peer, URL: endpoint, Verifier: pair.verifier(t), Card: func(context.Context) ([]byte, error) {
				cardCalls++
				if mode == "card-unavailable-after-handler" && effects == 1 {
					return nil, ErrRejected
				}
				return card, nil
			}, Handler: handlerFunc(func(context.Context, []byte) ([]byte, error) { effects++; return reply(), nil })})
			if e != nil {
				t.Fatal(e)
			}
			cert, roots := protectedCertificate(t)
			ingress, e := NewHTTPIngress(HTTPIngressConfig{Transport: HTTPTransportConfig{Session: pair.right, URL: endpoint, TLS: &tls.Config{Certificates: []tls.Certificate{cert}}, TTL: 30, Timeout: 3 * time.Second}, Server: server})
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(ingress.Close)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			rounds := 1
			if mode == "policy-denial-reuse" {
				rounds = 2
			}
			for step := 0; step < rounds; step++ {
				meta := metadata()
				deny := mode == "policy-denial-reuse" && step == 0
				if deny {
					meta["a2a_version"] = "0.3"
				}
				message, e := left.SealRequest(ctx, rpc("GetTask", map[string]any{"id": "task"}), meta, 30)
				if e != nil {
					t.Fatal(e)
				}
				frame, e := protectedRequestFrame(ctx, message)
				if e != nil {
					t.Fatal(e)
				}
				clientRaw, serverRaw := net.Pipe()
				t.Cleanup(func() { _ = clientRaw.Close(); _ = serverRaw.Close() })
				done := make(chan struct{})
				go func() { defer close(done); ingress.state.connection(ctx, serverRaw) }()
				if e = clientRaw.SetDeadline(time.Now().Add(3 * time.Second)); e != nil {
					t.Fatal(e)
				}
				client := tls.Client(clientRaw, &tls.Config{RootCAs: roots, ServerName: "localhost", MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, NextProtos: []string{"http/1.1"}})
				if e = client.HandshakeContext(ctx); e != nil {
					t.Fatal(e)
				}
				if n, err := client.Write(frame); err != nil || n != len(frame) {
					t.Fatal("unit TLS request write", n, err)
				}
				response, readErr := protectedReadResponse(client)
				select {
				case <-done:
				case <-ctx.Done():
					t.Fatal("memory server did not terminate")
				}
				if mode == "card-unavailable-after-handler" {
					if readErr == nil || effects != 1 || cardCalls != 2 || !ingress.state.transport.closing.Load() {
						t.Fatal("failed final Card check exposed a response or retained the server", readErr, effects, cardCalls)
					}
					pair.rightReplay.mu.Lock()
					accepted := pair.rightReplay.accepted
					pair.rightReplay.mu.Unlock()
					pair.leftReplay.mu.Lock()
					responses := pair.leftReplay.accepted
					pair.leftReplay.mu.Unlock()
					if accepted != 1 || responses != 0 {
						t.Fatal("accepted request history was undone or response invented", accepted, responses)
					}
					continue
				}
				if readErr != nil {
					t.Fatal(readErr)
				}
				view, e := left.OpenResponse(ctx, response)
				if e != nil || view == nil {
					t.Fatal("memory response lost authentication", e)
				}
				if deny {
					if response.Status != http.StatusUnauthorized || view.Success() || view.Error() != "policy_denied" || len(view.Payload()) != 0 || effects != 0 {
						t.Fatal("policy denial reached Handler or became success")
					}
				} else if response.Status != http.StatusOK || !view.Success() || view.Error() != "" || !bytes.Equal(view.Payload(), reply()) || effects != 1 {
					t.Fatal("server did not accept the next explicit valid call", e, effects)
				}
				if ingress.state.transport.closing.Load() {
					t.Fatal("complete signed denial retired the whole server session")
				}
				pair.rightReplay.mu.Lock()
				requests := pair.rightReplay.accepted
				pair.rightReplay.mu.Unlock()
				pair.leftReplay.mu.Lock()
				responses := pair.leftReplay.accepted
				pair.leftReplay.mu.Unlock()
				if requests != step+1 || responses != step+1 {
					t.Fatal("signed terminal request/response acceptance missing", requests, responses)
				}
			}
		})
	}
}
