package a2abinding

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type ownerFunc func(context.Context, ProtectedCall, func(context.Context) error) (AuthenticatedResponse, error)

func (f ownerFunc) Exchange(c context.Context, p ProtectedCall, g func(context.Context) error) (AuthenticatedResponse, error) {
	return f(c, p, g)
}

type handlerFunc func(context.Context, []byte) ([]byte, error)

func (f handlerFunc) Handle(c context.Context, b []byte) ([]byte, error) { return f(c, b) }
func rpc(method string, params any) []byte {
	return encode(map[string]any{"jsonrpc": "2.0", "id": "one", "method": method, "params": params})
}
func reply() []byte { return []byte(`{"jsonrpc":"2.0","id":"one","result":{"ok":true}}`) }
func metadata() map[string]string {
	return map[string]string{"a2a_version": "1.0", "a2a_extensions": ExtensionURI}
}
func TestClientPreservesBytesAndNeverRetries(t *testing.T) {
	for _, mode := range []string{"success", "child context", "owner error", "no final", "twice final", "record changed", "wrong sender", "wrong response id", "malformed response", "canceled check"} {
		t.Run(mode, func(t *testing.T) {
			m, s, k := fixture(t)
			card := signed(t, m, k)
			count := 0
			raw := []byte(" {\"jsonrpc\":\"2.0\",\"id\":\"one\",\"method\":\"GetTask\",\"params\":{\"id\":\"task\"}} ")
			owner := ownerFunc(func(ctx context.Context, p ProtectedCall, check func(context.Context) error) (AuthenticatedResponse, error) {
				count++
				if string(p.Payload()) != string(raw) || p.Recipient() != peer || p.Target().URL != endpoint || p.Metadata()["a2a_version"] != "1.0" {
					t.Fatal("exact binding lost")
				}
				copy := p.Metadata()
				copy["a2a_version"] = "0.3"
				b := p.Payload()
				b[0] = '!'
				if p.Metadata()["a2a_version"] != "1.0" || p.Payload()[0] == '!' {
					t.Fatal("mutable call")
				}
				if mode == "owner error" {
					return AuthenticatedResponse{}, errors.New("lost")
				}
				if mode == "record changed" {
					s.record.Version = "4"
				}
				if mode == "canceled check" {
					c, cancel := context.WithCancel(ctx)
					cancel()
					check(c)
					return AuthenticatedResponse{peer, reply()}, nil
				}
				if mode != "no final" {
					checkctx := ctx
					if mode == "child context" {
						var cancel context.CancelFunc
						checkctx, cancel = context.WithCancel(ctx)
						defer cancel()
					}
					if e := check(checkctx); e != nil {
						return AuthenticatedResponse{}, e
					}
					if mode == "twice final" {
						check(ctx)
					}
				}
				r := AuthenticatedResponse{peer, reply()}
				if mode == "wrong sender" {
					r.Sender = "other"
				}
				if mode == "wrong response id" {
					r.Payload = []byte(`{"jsonrpc":"2.0","id":"other","result":{}}`)
				}
				if mode == "malformed response" {
					r.Payload = []byte(`{`)
				}
				return r, nil
			})
			c, e := NewClient(verifier(t, s), owner)
			if e != nil {
				t.Fatal(e)
			}
			out, e := c.Call(context.Background(), card, raw)
			good := mode == "success" || mode == "child context"
			if good && (e != nil || string(out) != string(reply())) {
				t.Fatal(e)
			}
			if !good && e != ErrRejected {
				t.Fatal("accepted failure", mode, e)
			}
			if count != 1 {
				t.Fatal("retry", count)
			}
		})
	}
}
func TestServerVersionTenantAndUnsupportedMethods(t *testing.T) {
	for _, mode := range []string{"success", "tenant", "tenant other", "tenant omitted", "tenant empty", "unexpected tenant", "wrong recipient", "wrong url", "missing version", "legacy version", "missing extension", "space extension", "duplicate extension", "streaming", "subscribe", "push", "extended", "invalid params", "wide integer", "handler error", "bad response", "card error", "card changes"} {
		t.Run(mode, func(t *testing.T) {
			m, s, k := fixture(t)
			if mode == "tenant" || mode == "tenant other" || mode == "tenant omitted" || mode == "tenant empty" {
				m["supportedInterfaces"].([]any)[0].(map[string]any)["tenant"] = "billing"
			}
			card := signed(t, m, k)
			effects := 0
			h := handlerFunc(func(context.Context, []byte) ([]byte, error) {
				effects++
				if mode == "handler error" {
					return nil, errors.New("effect failed")
				}
				if mode == "bad response" {
					return []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`), nil
				}
				return reply(), nil
			})
			cfg := ServerConfig{DID: peer, URL: endpoint, Verifier: verifier(t, s), Card: func(context.Context) ([]byte, error) {
				if mode == "card error" {
					return nil, errors.New("gone")
				}
				return card, nil
			}, Handler: h, Tenant: func(context.Context, string) (string, error) {
				if mode == "tenant other" {
					return "did:sage:web:agents.example.com:other", nil
				}
				if mode == "card changes" {
					s.record.Version = "4"
				}
				return peer, nil
			}}
			srv, e := NewServer(cfg)
			if e != nil {
				t.Fatal(e)
			}
			p := map[string]any{"id": "task"}
			in := AuthenticatedRequest{endpoint, peer, metadata(), nil}
			method := "GetTask"
			switch mode {
			case "tenant", "tenant other":
				p["tenant"] = "billing"
			case "tenant empty":
				p["tenant"] = ""
			case "unexpected tenant":
				p["tenant"] = "other"
			case "card changes":
				p["tenant"] = "billing"
				m["supportedInterfaces"].([]any)[0].(map[string]any)["tenant"] = "billing"
				card = signed(t, m, k)
			case "wrong recipient":
				in.Recipient = "other"
			case "wrong url":
				in.URL = endpoint + "/"
			case "missing version":
				delete(in.Metadata, "a2a_version")
			case "legacy version":
				in.Metadata["a2a_version"] = "0.3"
			case "missing extension":
				delete(in.Metadata, "a2a_extensions")
			case "space extension":
				in.Metadata["a2a_extensions"] = " " + ExtensionURI
			case "duplicate extension":
				in.Metadata["a2a_extensions"] = ExtensionURI + "," + ExtensionURI
			case "streaming":
				method = "SendStreamingMessage"
			case "subscribe":
				method = "SubscribeToTask"
			case "push":
				method = "CreateTaskPushNotificationConfig"
			case "extended":
				method = "GetExtendedAgentCard"
			case "invalid params":
				delete(p, "id")
			case "wide integer":
				p["historyLength"] = json.Number("9007199254740992")
			}
			in.Payload = rpc(method, p)
			out, e := srv.Handle(context.Background(), in)
			switch mode {
			case "success", "tenant":
				if e != nil || effects != 1 {
					t.Fatal(e, effects)
				}
			case "streaming", "subscribe", "push", "extended", "invalid params", "wide integer":
				if e != nil || effects != 0 {
					t.Fatal(e, effects)
				}
				var result map[string]any
				json.Unmarshal(out, &result)
				if result["error"] == nil {
					t.Fatal("missing A2A error")
				}
			case "handler error", "bad response":
				if e != nil || effects != 1 {
					t.Fatal(e, effects)
				}
				var v map[string]any
				json.Unmarshal(out, &v)
				if v["error"].(map[string]any)["code"] != float64(-32603) {
					t.Fatal("application failure mislabeled as authentication failure")
				}
			default:
				if e != ErrRejected || effects != 0 {
					t.Fatal(mode, e, effects)
				}
			}
		})
	}
}
func TestMalformedRPCMetadataAndLocalConfiguration(t *testing.T) {
	for _, raw := range []string{`{}`, `{"jsonrpc":"2.0","id":null,"method":"GetTask"}`, `{"jsonrpc":"2.0","id":true,"method":"GetTask"}`, `{"jsonrpc":"2.0","id":1.5,"method":"GetTask"}`, `{"jsonrpc":"2.0","id":1,"method":"unknown"}`, `{"jsonrpc":"2.0","id":1,"method":"GetTask","params":[]}`, `{"jsonrpc":"2.0","id":1,"method":"GetTask","params":{"tenant":1}}`, `{"jsonrpc":"2.0","id":1,"result":{},"error":{}}`, `{"jsonrpc":"2.0","id":1,"error":{"code":1.5,"message":"bad"}}`, `{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":true}}`, `{"jsonrpc":"2.0","id":1,"error":{"code":-1,"message":"bad","extra":1}}`} {
		if _, e := parseRPC([]byte(raw), false); e == nil {
			t.Fatal(raw)
		}
		if _, e := parseRPC([]byte(raw), true); e == nil {
			t.Fatal(raw)
		}
	}
	for _, m := range []map[string]string{{}, {"a2a_version": "1.0", "a2a_extensions": ExtensionURI + ",bad"}, {"a2a_version": "1.0", "a2a_extensions": ExtensionURI + ",urn:other", "Bad": "x"}, {"a2a_version": "1.0", "a2a_extensions": ExtensionURI, "text": string(make([]byte, 1025))}} {
		if validMetadata(m) {
			t.Fatal("invalid metadata")
		}
	}
	if _, e := NewClient(nil, nil); e != ErrRejected {
		t.Fatal(e)
	}
	if _, e := NewServer(ServerConfig{}); e != ErrRejected {
		t.Fatal(e)
	}
	_, s, _ := fixture(t)
	v := verifier(t, s)
	if _, e := NewServer(ServerConfig{DID: "bad", URL: endpoint, Verifier: v, Card: func(context.Context) ([]byte, error) { return nil, nil }, Handler: handlerFunc(func(context.Context, []byte) ([]byte, error) { return nil, nil })}); e != ErrRejected {
		t.Fatal(e)
	}
}
func TestTenantPresenceExtraExtensionsAndPushAliases(t *testing.T) {
	for _, mode := range []string{"client extra required", "server extra missing", "server extra active", "client tenant missing", "client tenant empty", "client tenant extra", "router missing", "router error", "push camel", "push proto"} {
		t.Run(mode, func(t *testing.T) {
			m, s, k := fixture(t)
			if mode == "client extra required" || mode == "server extra missing" || mode == "server extra active" {
				m["capabilities"].(map[string]any)["extensions"] = append(m["capabilities"].(map[string]any)["extensions"].([]any), map[string]any{"uri": "urn:required", "required": true})
			}
			if mode == "client tenant missing" || mode == "client tenant empty" || mode == "router missing" || mode == "router error" {
				m["supportedInterfaces"].([]any)[0].(map[string]any)["tenant"] = "billing"
			}
			card := signed(t, m, k)
			params := map[string]any{"id": "task"}
			if mode == "client tenant empty" {
				params["tenant"] = ""
			}
			if mode == "client tenant extra" {
				params["tenant"] = "other"
			}
			if len(mode) >= 6 && mode[:6] == "client" {
				calls := 0
				c, _ := NewClient(verifier(t, s), ownerFunc(func(context.Context, ProtectedCall, func(context.Context) error) (AuthenticatedResponse, error) {
					calls++
					return AuthenticatedResponse{}, nil
				}))
				if _, e := c.Call(context.Background(), card, rpc("GetTask", params)); e != ErrRejected || calls != 0 {
					t.Fatal("client did not stop", e, calls)
				}
				return
			}
			effects := 0
			cfg := ServerConfig{DID: peer, URL: endpoint, Verifier: verifier(t, s), Card: func(context.Context) ([]byte, error) { return card, nil }, Handler: handlerFunc(func(context.Context, []byte) ([]byte, error) { effects++; return reply(), nil })}
			if mode == "router error" {
				cfg.Tenant = func(context.Context, string) (string, error) { return "", errors.New("route gone") }
			}
			meta := metadata()
			method := "GetTask"
			if mode == "server extra active" {
				meta["a2a_extensions"] += ",urn:required"
			}
			if mode == "router missing" || mode == "router error" {
				params["tenant"] = "billing"
			}
			if mode == "push camel" || mode == "push proto" {
				name := "taskPushNotificationConfig"
				if mode == "push proto" {
					name = "task_push_notification_config"
				}
				method = "SendMessage"
				params = map[string]any{"message": map[string]any{"messageId": "m", "role": "ROLE_USER", "parts": []any{map[string]any{"text": "hello"}}}, "configuration": map[string]any{name: map[string]any{"url": "https://example.com/push"}}}
			}
			srv, _ := NewServer(cfg)
			out, e := srv.Handle(context.Background(), AuthenticatedRequest{endpoint, peer, meta, rpc(method, params)})
			if mode == "server extra active" {
				if e != nil || effects != 1 {
					t.Fatal(e, effects)
				}
				return
			}
			if effects != 0 {
				t.Fatal("unsupported request caused effect")
			}
			if mode == "router missing" || mode == "router error" {
				if e != ErrRejected {
					t.Fatal(e)
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			var v map[string]any
			json.Unmarshal(out, &v)
			code := v["error"].(map[string]any)["code"]
			want := float64(-32003)
			if mode == "server extra missing" {
				want = -32008
			}
			if code != want {
				t.Fatal("wrong A2A error", code, want)
			}
		})
	}
	var c *Client
	if _, e := c.Call(context.Background(), nil, nil); e != ErrRejected {
		t.Fatal(e)
	}
	var s *Server
	if _, e := s.Handle(context.Background(), AuthenticatedRequest{}); e != ErrRejected {
		t.Fatal(e)
	}
	_, src, _ := fixture(t)
	if _, e := NewClient(verifier(t, src), nil); e != ErrRejected {
		t.Fatal(e)
	}
}
