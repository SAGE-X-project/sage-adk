//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func shortDir(t *testing.T) string {
	t.Helper()
	// Unix socket paths are limited to about 104 bytes on darwin.
	dir, err := os.MkdirTemp("/tmp", "gsig")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func testKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, k, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func start(t *testing.T, c Config) (*Server, string) {
	t.Helper()
	s, err := NewServer(c)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(shortDir(t), "s")
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, l) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
		s.Close()
	})
	return s, path
}

func self() uint32 { return uint32(os.Getuid()) }

func config(k ed25519.PrivateKey, roles ...string) Config {
	return Config{Key: k, Roles: roles, AllowedUIDs: []uint32{self()}, Timeout: time.Second, MaxConnections: 4}
}

func client(t *testing.T, path string) *Client {
	t.Helper()
	c, err := NewClient(path, self(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSignerRoundTripPerRole(t *testing.T) {
	k := testKey(t)
	_, path := start(t, config(k, "intent", "result", "transport"))
	c := client(t, path)
	ctx := context.Background()
	pub, err := c.PublicKey(ctx)
	if err != nil || !bytes.Equal(pub, k.Public().(ed25519.PublicKey)) {
		t.Fatal("public key", err)
	}
	for _, d := range []string{DomainIntent, DomainResult, DomainWireRequest, DomainWireResponse, DomainCompletion} {
		msg := append([]byte(d), `{"v":"0.10.0"}`...)
		sig, err := c.Sign(ctx, msg)
		if err != nil || !ed25519.Verify(pub, msg, sig) {
			t.Fatalf("domain %q: %v", d, err)
		}
	}
}

func TestSignerRefusesOutOfScopeMessages(t *testing.T) {
	k := testKey(t)
	_, path := start(t, config(k, "intent"))
	c := client(t, path)
	ctx := context.Background()
	for name, msg := range map[string][]byte{
		"result-domain":    append([]byte(DomainResult), '{', '}'),
		"transport-domain": append([]byte(DomainWireRequest), '{', '}'),
		"completion":       append([]byte(DomainCompletion), '{', '}'),
		"http-base":        []byte("\"@method\": POST\n\"@signature-params\": ()"),
		"domain-only":      []byte(DomainIntent),
		"empty":            {},
		"near-domain":      append([]byte("sage-execution-intent|0.9.0\x00"), '{', '}'),
	} {
		if sig, err := c.Sign(ctx, msg); err == nil || sig != nil {
			t.Fatalf("%s signed", name)
		}
	}
	if _, err := c.Sign(ctx, append([]byte(DomainIntent), '{', '}')); err != nil {
		t.Fatal("allowed domain refused after denials", err)
	}
}

func TestSignerRefusesUnlistedAccounts(t *testing.T) {
	k := testKey(t)
	c := config(k, "intent")
	c.AllowedUIDs = []uint32{self() + 1}
	_, path := start(t, c)
	if _, err := client(t, path).PublicKey(context.Background()); err == nil {
		t.Fatal("unlisted host account served")
	}
	_, path = start(t, config(k, "intent"))
	wrong, err := NewClient(path, self()+1, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrong.Sign(context.Background(), append([]byte(DomainIntent), '{', '}')); err == nil {
		t.Fatal("client accepted a socket served by another account")
	}
}

func rawExchange(t *testing.T, path string, frame []byte) (byte, bool) {
	t.Helper()
	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err = conn.Write(frame); err != nil {
		return 0, false
	}
	if u, ok := conn.(*net.UnixConn); ok {
		_ = u.CloseWrite()
	}
	status, _, err := readFrame(conn)
	if err != nil {
		return 0, false
	}
	return status, true
}

func TestSignerRefusesMalformedFrames(t *testing.T) {
	_, path := start(t, config(testKey(t), "intent"))
	frame := func(m string, op byte, n uint32, body []byte) []byte {
		h := make([]byte, headerBytes)
		copy(h, m)
		h[4] = op
		binary.BigEndian.PutUint32(h[5:], n)
		return append(h, body...)
	}
	cases := map[string][]byte{
		"bad-magic":      frame("SGS0", opPublicKey, 0, nil),
		"unknown-op":     frame(magic, 9, 0, nil),
		"oversized":      frame(magic, opSign, maxMessage+1, nil),
		"truncated":      frame(magic, opSign, 10, []byte("abc")),
		"short-header":   []byte("SGS"),
		"public-payload": frame(magic, opPublicKey, 1, []byte{0}),
	}
	for name, f := range cases {
		if status, ok := rawExchange(t, path, f); ok && status == statusOK {
			t.Fatalf("%s accepted", name)
		}
	}
}

func TestServerConfigurationRefusals(t *testing.T) {
	k := testKey(t)
	inconsistent := append(ed25519.PrivateKey(nil), k...)
	inconsistent[63] ^= 1
	for name, c := range map[string]Config{
		"short-key":      {Key: k[:32], Roles: []string{"intent"}, AllowedUIDs: []uint32{1}, Timeout: time.Second, MaxConnections: 1},
		"inconsistent":   {Key: inconsistent, Roles: []string{"intent"}, AllowedUIDs: []uint32{1}, Timeout: time.Second, MaxConnections: 1},
		"no-roles":       {Key: k, AllowedUIDs: []uint32{1}, Timeout: time.Second, MaxConnections: 1},
		"unknown-role":   {Key: k, Roles: []string{"http"}, AllowedUIDs: []uint32{1}, Timeout: time.Second, MaxConnections: 1},
		"duplicate-role": {Key: k, Roles: []string{"intent", "intent"}, AllowedUIDs: []uint32{1}, Timeout: time.Second, MaxConnections: 1},
		"no-accounts":    {Key: k, Roles: []string{"intent"}, Timeout: time.Second, MaxConnections: 1},
		"zero-timeout":   {Key: k, Roles: []string{"intent"}, AllowedUIDs: []uint32{1}, MaxConnections: 1},
		"long-timeout":   {Key: k, Roles: []string{"intent"}, AllowedUIDs: []uint32{1}, Timeout: time.Hour, MaxConnections: 1},
		"no-connections": {Key: k, Roles: []string{"intent"}, AllowedUIDs: []uint32{1}, Timeout: time.Second},
		"too-many":       {Key: k, Roles: []string{"intent"}, AllowedUIDs: []uint32{1}, Timeout: time.Second, MaxConnections: maxConnections + 1},
	} {
		if s, err := NewServer(c); err == nil || s != nil {
			t.Fatalf("%s accepted", name)
		}
	}
	want := append(ed25519.PublicKey(nil), k.Public().(ed25519.PublicKey)...)
	s, err := NewServer(config(k, "intent"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range k {
		k[i] = 0
	}
	if !bytes.Equal(s.PublicKey(), want) {
		t.Fatal("server did not copy its key")
	}
}

func TestClientRefusals(t *testing.T) {
	for name, path := range map[string]string{"relative": "s", "unclean": "/tmp/../tmp/s", "empty": ""} {
		if c, err := NewClient(path, 0, time.Second); err == nil || c != nil {
			t.Fatalf("%s path accepted", name)
		}
	}
	if c, err := NewClient("/tmp/s", 0, 0); err == nil || c != nil {
		t.Fatal("zero timeout accepted")
	}
	_, path := start(t, config(testKey(t), "intent"))
	c := client(t, path)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	//lint:ignore SA1012 nil context refusal is the scenario under test
	for name, ctx := range map[string]context.Context{"nil": nil, "cancelled": cancelled} { //nolint:staticcheck
		if _, err := c.Sign(ctx, append([]byte(DomainIntent), '{', '}')); err == nil {
			t.Fatalf("%s context accepted", name)
		}
	}
	missing := client(t, filepath.Join(shortDir(t), "missing"))
	if _, err := missing.PublicKey(context.Background()); err == nil {
		t.Fatal("missing socket accepted")
	}
	var none *Client
	if _, err := none.PublicKey(context.Background()); err == nil {
		t.Fatal("nil client accepted")
	}
}

func TestCloseRetiresKey(t *testing.T) {
	s, path := start(t, config(testKey(t), "intent"))
	c := client(t, path)
	s.Close()
	if _, err := c.Sign(context.Background(), append([]byte(DomainIntent), '{', '}')); err == nil {
		t.Fatal("closed signer signed")
	}
	if _, err := c.PublicKey(context.Background()); err == nil {
		t.Fatal("closed signer answered")
	}
}

func TestKeyFileCreationAndChecks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed")
	pub, err := GenerateKeyFile(path)
	if err != nil {
		t.Fatal(err)
	}
	k, err := LoadKeyFile(path)
	if err != nil || !bytes.Equal(k.Public().(ed25519.PublicKey), pub) {
		t.Fatal("reload", err)
	}
	if _, err = GenerateKeyFile(path); err == nil {
		t.Fatal("existing key overwritten")
	}
	link := filepath.Join(dir, "link")
	if err = os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadKeyFile(link); err == nil {
		t.Fatal("symlink accepted")
	}
	if err = os.Chmod(path, 0440); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadKeyFile(path); err == nil {
		t.Fatal("group-readable key accepted")
	}
	short := filepath.Join(dir, "short")
	if err = os.WriteFile(short, make([]byte, 31), 0400); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadKeyFile(short); err == nil {
		t.Fatal("short key accepted")
	}
	if _, err = LoadKeyFile(dir); err == nil {
		t.Fatal("directory accepted")
	}
}

func TestFrameRoundTrip(t *testing.T) {
	r, w := io.Pipe()
	go func() { _ = writeFrame(w, opSign, []byte("x")); _ = w.Close() }()
	op, body, err := readFrame(r)
	if err != nil || op != opSign || string(body) != "x" {
		t.Fatal(op, body, err)
	}
	if writeFrame(io.Discard, opSign, make([]byte, maxMessage+1)) == nil {
		t.Fatal("oversized frame written")
	}
}
