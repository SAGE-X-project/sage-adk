package a2abinding

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSecuritySchemesRemainSignedWithPresenceRules(t *testing.T) {
	m, s, k := fixture(t)
	m["securitySchemes"] = map[string]any{
		"api":   map[string]any{"apiKeySecurityScheme": map[string]any{"description": "API", "location": "header", "name": "X-Key"}},
		"http":  map[string]any{"httpAuthSecurityScheme": map[string]any{"description": "HTTP", "scheme": "Bearer", "bearerFormat": "JWT"}},
		"oauth": map[string]any{"oauth2SecurityScheme": map[string]any{"description": "OAuth", "flows": map[string]any{"authorizationCode": map[string]any{"authorizationUrl": "https://example.com/auth", "tokenUrl": "https://example.com/token", "scopes": map[string]any{"read": "Read"}, "pkceRequired": true}}}},
		"oidc":  map[string]any{"openIdConnectSecurityScheme": map[string]any{"openIdConnectUrl": "https://example.com/.well-known/openid-configuration"}},
		"tls":   map[string]any{"mtlsSecurityScheme": map[string]any{"description": "mTLS"}},
	}
	m["securityRequirements"] = []any{map[string]any{"schemes": map[string]any{"api": map[string]any{"list": []any{"read"}}}}}
	m["provider"] = map[string]any{"url": "https://example.com", "organization": "Provider"}
	m["documentationUrl"] = ""
	m["iconUrl"] = ""
	raw := signed(t, m, k)
	if _, e := verifier(t, s).Verify(context.Background(), raw); e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{"unknown scheme", "two schemes", "bad map", "bad list", "bad scalar", "bad params"} {
		var copy map[string]any
		json.Unmarshal(raw, &copy)
		schemes := copy["securitySchemes"].(map[string]any)
		switch bad {
		case "unknown scheme":
			schemes["api"].(map[string]any)["invented"] = true
		case "two schemes":
			schemes["api"].(map[string]any)["mtlsSecurityScheme"] = map[string]any{}
		case "bad map":
			copy["securitySchemes"] = []any{}
		case "bad list":
			copy["defaultOutputModes"] = true
		case "bad scalar":
			copy["capabilities"].(map[string]any)["streaming"] = "false"
		case "bad params":
			copy["capabilities"].(map[string]any)["extensions"].([]any)[0].(map[string]any)["params"] = false
		}
		rejectCard(t, s, encode(copy))
	}
}
func TestExecutedRequestModelsRejectInvalidFieldsBeforeEffects(t *testing.T) {
	m, s, k := fixture(t)
	card := signed(t, m, k)
	effects := 0
	srv, _ := NewServer(ServerConfig{DID: peer, URL: endpoint, Verifier: verifier(t, s), Card: func(context.Context) ([]byte, error) { return card, nil }, Handler: handlerFunc(func(context.Context, []byte) ([]byte, error) { effects++; return reply(), nil })})
	cases := []struct {
		method string
		params any
		valid  bool
	}{
		{"SendMessage", map[string]any{"message": map[string]any{"messageId": "message", "role": "ROLE_USER", "parts": []any{map[string]any{"data": nil}, map[string]any{"text": "hello"}, map[string]any{"raw": "AQ=="}}}, "configuration": map[string]any{"historyLength": 0, "returnImmediately": true}}, true},
		{"SendMessage", map[string]any{"message": map[string]any{"messageId": "message", "role": 1, "parts": []any{map[string]any{"data": true}}}}, true},
		{"ListTasks", map[string]any{"pageSize": 10, "status": "TASK_STATE_COMPLETED", "statusTimestampAfter": "2026-10-13T00:00:00Z", "includeArtifacts": false}, true},
		{"ListTasks", map[string]any{"status": 3, "historyLength": json.Number("10.0")}, true},
		{"GetTask", map[string]any{"id": "task", "historyLength": "1e1"}, true},
		{"GetTask", map[string]any{"id": "task", "historyLength": nil}, true},
		{"GetTask", map[string]any{"id": "task", "historyLength": "-0"}, true},
		{"GetTask", map[string]any{"id": "task", "historyLength": ""}, false},
		{"GetTask", map[string]any{"id": "task", "historyLength": "100.000000000000001"}, false},
		{"GetTask", map[string]any{"id": "task", "historyLength": "2147483648"}, false},
		{"SendMessage", map[string]any{"message": nil}, false},
		{"GetTask", map[string]any{"id": "task", "historyLength": nil, "history_length": 10}, false},
		{"SendMessage", map[string]any{}, false},
		{"SendMessage", map[string]any{"message": map[string]any{"messageId": "message", "role": "ROLE_USER", "parts": []any{}}}, false},
		{"SendMessage", map[string]any{"message": map[string]any{"messageId": "message", "role": "user", "parts": []any{map[string]any{"text": "hello"}}}}, false},
		{"GetTask", map[string]any{"id": "task", "historyLength": json.Number("2147483648")}, false},
		{"GetTask", map[string]any{"id": "task", "historyLength": json.Number("10.5")}, false},
		{"GetTask", map[string]any{"id": "task", "unknown": true}, false},
		{"ListTasks", map[string]any{"status": 9}, false},
		{"ListTasks", map[string]any{"status": "unknown"}, false},
		{"ListTasks", map[string]any{"statusTimestampAfter": "2026-10-13T00:00:00+00:00"}, false},
		{"ListTasks", map[string]any{"statusTimestampAfter": "2026-10-13T00:00:00.1234567890Z"}, false},
		{"CancelTask", map[string]any{"id": "task"}, true},
	}
	for _, c := range cases {
		before := effects
		out, e := srv.Handle(context.Background(), AuthenticatedRequest{endpoint, peer, metadata(), rpc(c.method, c.params)})
		if e != nil {
			t.Fatal(c.method, e)
		}
		if c.valid && effects != before+1 {
			t.Fatal("valid model not dispatched", c)
		}
		if !c.valid && effects != before {
			t.Fatal("invalid model caused effect", c)
		}
		if !c.valid {
			var v map[string]any
			json.Unmarshal(out, &v)
			if v["error"].(map[string]any)["code"] != float64(-32602) {
				t.Fatal("invalid params code")
			}
		}
	}
}
