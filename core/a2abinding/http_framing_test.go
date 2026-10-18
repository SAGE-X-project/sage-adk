package a2abinding

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
)

// Structural and altered input controls are memory-only unit data. They are not
// network probes and never contact an external or loopback listener.
func TestProtectedHTTPFixedFraming010(t *testing.T) {
	normal := "POST /a%2Fa?x=%2f HTTP/1.1\r\nHost: agents.example.com\r\nContent-Length: 2\r\nConnection: close\r\n\r\n{}"
	negative := map[string]string{
		"truncated-head":     "POST / HTTP/1.1\r\n",
		"bare-newline":       strings.ReplaceAll(normal, "\r\n", "\n"),
		"duplicate-host":     strings.Replace(normal, "Content-Length:", "host: agents.example.com\r\nContent-Length:", 1),
		"duplicate-length":   strings.Replace(normal, "Content-Length: 2", "Content-Length: 2\r\ncontent-length: 2", 1),
		"conflicting-length": strings.Replace(normal, "Content-Length: 2", "Content-Length: 2\r\ncontent-length: 3", 1),
		"folding":            strings.Replace(normal, "Connection:", " Connection:", 1),
		"invalid-name":       strings.Replace(normal, "Connection:", "Con(n):", 1),
		"missing-colon":      strings.Replace(normal, "Connection:", "Connection", 1),
		"missing-name":       strings.Replace(normal, "Connection:", ":", 1),
		"control-value":      strings.Replace(normal, "Connection: close", "X-Inert: a\x00b", 1),
		"transfer-encoding":  strings.Replace(normal, "Connection: close", "Transfer-Encoding: chunked", 1),
		"trailer":            strings.Replace(normal, "Connection: close", "Trailer: Signature", 1),
		"compression":        strings.Replace(normal, "Connection: close", "Content-Encoding: gzip", 1),
		"upgrade":            strings.Replace(normal, "Connection: close", "Upgrade: h2c", 1),
		"continue":           strings.Replace(normal, "Connection: close", "Expect: 100-continue", 1),
		"proxy":              strings.Replace(normal, "Connection: close", "Proxy-Connection: close", 1),
		"forwarded":          strings.Replace(normal, "Connection: close", "Forwarded: host=elsewhere", 1),
		"x-forwarded":        strings.Replace(normal, "Connection: close", "X-Forwarded-Host: elsewhere", 1),
		"keepalive":          strings.Replace(normal, "Connection: close", "Connection: keep-alive", 1),
		"leading-zero":       strings.Replace(normal, "Content-Length: 2", "Content-Length: 02", 1),
		"negative-length":    strings.Replace(normal, "Content-Length: 2", "Content-Length: -2", 1),
		"oversize-length":    strings.Replace(normal, "Content-Length: 2", "Content-Length: 32769", 1),
		"nonnumeric-length":  strings.Replace(normal, "Content-Length: 2", "Content-Length: none", 1),
		"empty-length":       strings.Replace(normal, "Content-Length: 2", "Content-Length: 0", 1),
		"missing-length":     strings.Replace(normal, "Content-Length: 2\r\n", "", 1),
		"short-body":         strings.TrimSuffix(normal, "}"),
		"large-head":         "POST / HTTP/1.1\r\nX-Inert: " + strings.Repeat("a", protectedHeaderLimit) + "\r\nContent-Length: 2\r\n\r\n{}",
	}
	for name, raw := range negative {
		t.Run(name, func(t *testing.T) {
			if _, _, e := protectedReadFrame(strings.NewReader(raw)); e != ErrRejected {
				t.Fatal("invalid framing accepted", e)
			}
		})
	}
	frame, headers, e := protectedReadFrame(strings.NewReader(normal))
	if e != nil || string(frame) != normal || len(headers) != 3 {
		t.Fatal("exact bounded frame lost", e)
	}
	target, e := protectedTarget(endpoint)
	if e != nil {
		t.Fatal(e)
	}
	state := &protectedHTTPState{target: target, exact: endpoint}
	valid := strings.Replace(normal, "/a%2Fa?x=%2f", "/a2a", 1)
	message, e := protectedReadRequest(strings.NewReader(valid), state)
	if e != nil || message.Method != "POST" || message.Target != endpoint || message.Authority != "agents.example.com" || string(message.Body) != "{}" {
		t.Fatal("actual request facts lost", message, e)
	}
	for name, raw := range map[string]string{
		"method":          strings.Replace(valid, "POST", "GET", 1),
		"version":         strings.Replace(valid, "HTTP/1.1", "HTTP/1.0", 1),
		"host":            strings.Replace(valid, "agents.example.com", "elsewhere.example", 1),
		"route":           strings.Replace(valid, "/a2a", "/elsewhere", 1),
		"absolute-target": strings.Replace(valid, "/a2a", endpoint, 1),
		"startline":       strings.Replace(valid, "POST /a2a HTTP/1.1", "not an HTTP request", 1),
	} {
		t.Run("request/"+name, func(t *testing.T) {
			if _, e := protectedReadRequest(strings.NewReader(raw), state); e != ErrRejected {
				t.Fatal("mismatched request admitted", e)
			}
		})
	}
	response := hpke.HTTPMessage010{Status: 200, Headers: [][2]string{{"Content-Type", "application/json"}}, Body: []byte("{}")}
	encoded, e := protectedResponseFrame(response)
	if e != nil {
		t.Fatal(e)
	}
	received, e := protectedReadResponse(bytes.NewReader(encoded))
	if e != nil || received.Status != 200 || !bytes.Equal(received.Body, response.Body) {
		t.Fatal("response facts lost", e)
	}
	for name, raw := range map[string]string{
		"invalid-status":       strings.Replace(string(encoded), "200 OK", "not-a-status", 1),
		"unsupported-protocol": strings.Replace(string(encoded), "HTTP/1.1", "HTTP/1.0", 1),
		"nonterminal":          strings.Replace(string(encoded), "200 OK", "100 Continue", 1),
		"no-content":           strings.Replace(string(encoded), "200 OK", "204 No Content", 1),
		"outside-status":       strings.Replace(string(encoded), "200 OK", "600 Outside", 1),
	} {
		t.Run("response/"+name, func(t *testing.T) {
			if _, e := protectedReadResponse(strings.NewReader(raw)); e != ErrRejected {
				t.Fatal("invalid response admitted", e)
			}
		})
	}
	for _, status := range []int{0, 199, 204, 600} {
		response.Status = status
		if _, e := protectedResponseFrame(response); e != ErrRejected {
			t.Fatal("invalid sealed response status", status)
		}
	}
	response.Status = 200
	response.Body = nil
	if _, e := protectedResponseFrame(response); e != ErrRejected {
		t.Fatal("empty envelope accepted")
	}
	message = hpke.HTTPMessage010{Method: "POST", Target: endpoint, Authority: "agents.example.com", Body: []byte("{}")}
	encoded, e = protectedRequestFrame(context.Background(), message)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := protectedReadRequest(bytes.NewReader(encoded), state); e != nil {
		t.Fatal("serialized target changed", e)
	}
	message.Authority = "elsewhere.example"
	if _, e := protectedRequestFrame(context.Background(), message); e != ErrRejected {
		t.Fatal("authority changed")
	}
}

func TestProtectedHTTPTransportConfiguration010(t *testing.T) {
	for name, raw := range map[string]string{"plain": "http://agents.example.com/a2a", "userinfo": "https://u@agents.example.com/a2a", "fragment": "https://agents.example.com/a2a#f", "implicit-path": "https://agents.example.com", "uppercase": "https://AGENTS.example.com/a2a", "default-port": "https://agents.example.com:443/a2a", "opaque": "https:agents.example.com/a2a", "space": "https://agents.example.com/a a", "too-large": "https://agents.example.com/" + strings.Repeat("a", 8192)} {
		t.Run(name, func(t *testing.T) {
			if _, e := protectedTarget(raw); e != ErrRejected {
				t.Fatal("unrepresentable target accepted", e)
			}
		})
	}
	cert, roots := protectedCertificate(t)
	bad := map[string]*tls.Config{
		"nil":                 nil,
		"skip":                {InsecureSkipVerify: true},
		"connection-callback": {VerifyConnection: func(tls.ConnectionState) error { return nil }},
		"peer-callback":       {VerifyPeerCertificate: func([][]byte, [][]*x509.Certificate) error { return nil }},
		"config-callback":     {GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) { return nil, nil }},
		"keylog":              {KeyLogWriter: io.Discard},
		"renegotiation":       {Renegotiation: tls.RenegotiateOnceAsClient},
		"random":              {Rand: rand.Reader},
		"time":                {Time: time.Now},
		"minimum":             {MinVersion: tls.VersionTLS12},
		"maximum":             {MaxVersion: tls.VersionTLS12},
		"h2":                  {NextProtos: []string{"h2"}},
	}
	for name, cfg := range bad {
		t.Run(name, func(t *testing.T) {
			if _, e := protectedTLS(cfg); e != ErrRejected {
				t.Fatal("unsupported TLS policy accepted", e)
			}
		})
	}
	cert.OCSPStaple = []byte{1}
	cert.SignedCertificateTimestamps = [][]byte{{2}}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}, RootCAs: roots, ClientCAs: roots, NextProtos: []string{"http/1.1"}, MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13}
	copyConfig, e := protectedTLS(cfg)
	if e != nil || copyConfig.RootCAs == roots || copyConfig.ClientCAs == roots || copyConfig.MinVersion != tls.VersionTLS13 || copyConfig.MaxVersion != tls.VersionTLS13 || copyConfig.ClientSessionCache != nil || !copyConfig.SessionTicketsDisabled {
		t.Fatal("TLS policy or pool ownership lost", e)
	}
	copyConfig.Certificates[0].Certificate[0][0] ^= 1
	copyConfig.Certificates[0].OCSPStaple[0] ^= 1
	copyConfig.Certificates[0].SignedCertificateTimestamps[0][0] ^= 1
	if bytes.Equal(copyConfig.Certificates[0].Certificate[0], cfg.Certificates[0].Certificate[0]) || copyConfig.Certificates[0].OCSPStaple[0] == cfg.Certificates[0].OCSPStaple[0] || copyConfig.Certificates[0].SignedCertificateTimestamps[0][0] == cfg.Certificates[0].SignedCertificateTimestamps[0][0] {
		t.Fatal("public certificate buffers aliased")
	}
}
