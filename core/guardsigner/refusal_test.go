//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// answerOnce serves one connection from this account with an OK frame
// carrying body, standing in for a signer that returns a malformed answer.
func answerOnce(t *testing.T, body []byte) string {
	t.Helper()
	path := filepath.Join(shortDir(t), "f")
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := l.AcceptUnix()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		if _, _, err = readFrame(conn); err == nil {
			_ = writeFrame(conn, statusOK, body)
		}
	}()
	t.Cleanup(func() { _ = l.Close(); <-done })
	return path
}

// The client refuses answers of the wrong size even with an OK status, so a
// confused or substituted signer cannot hand back a truncated key, signature
// or shared value.
func TestClientRefusesWrongSizeAnswers(t *testing.T) {
	ctx := context.Background()
	short := []byte{1, 2, 3}
	if _, err := client(t, answerOnce(t, short)).PublicKey(ctx); err == nil {
		t.Fatal("short public key accepted")
	}
	if _, err := client(t, answerOnce(t, short)).Sign(ctx, append([]byte(DomainIntent), '{', '}')); err == nil {
		t.Fatal("short signature accepted")
	}
	if _, err := client(t, answerOnce(t, short)).KEM().PublicKey(ctx); err == nil {
		t.Fatal("short KEM public key accepted")
	}
	if _, err := client(t, answerOnce(t, short)).KEM().ECDH(ctx, make([]byte, 32)); err == nil {
		t.Fatal("short shared value accepted")
	}
	c := client(t, filepath.Join(shortDir(t), "unused"))
	if _, err := c.Sign(ctx, make([]byte, maxMessage+1)); err == nil {
		t.Fatal("oversized message sent")
	}
	var none *KEMClient
	if _, err := none.PublicKey(ctx); err == nil {
		t.Fatal("nil KEM client returned a key")
	}
}

// When every connection slot is busy the signer closes further connections
// instead of queueing them, and serves again once a slot frees.
func TestServerShedsConnectionsBeyondCapacity(t *testing.T) {
	c := config(testKey(t), "intent")
	c.MaxConnections = 1
	_, path := start(t, c)
	busy, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	// Wait until the idle connection holds the only slot: the signer then
	// closes a second connection without answering.
	shed := false
	for i := 0; i < 50 && !shed; i++ {
		if status, ok := rawExchange(t, path, []byte("SGS")); !ok || status != statusDenied {
			shed = true
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if !shed {
		t.Fatal("connection beyond capacity was served")
	}
	_ = busy.Close()
	cl := client(t, path)
	for i := 0; ; i++ {
		if _, err := cl.PublicKey(context.Background()); err == nil {
			break
		} else if i == 50 {
			t.Fatal("signer did not recover a freed slot", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestServerRefusesInvalidUse(t *testing.T) {
	var none *Server
	if none.PublicKey() != nil {
		t.Fatal("nil server returned a key")
	}
	none.Close()
	s, err := NewServer(config(testKey(t), "intent"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Serve(context.Background(), nil) == nil {
		t.Fatal("nil listener served")
	}
	if none.Serve(context.Background(), &net.UnixListener{}) == nil {
		t.Fatal("nil server served")
	}
}

func TestKeyFileRefusals(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing", "key")
	if _, err := GenerateKeyFile(missing); err == nil {
		t.Fatal("key written into a missing directory")
	}
	if _, err := GenerateKEMFile(missing); err == nil {
		t.Fatal("kem key written into a missing directory")
	}
	if _, err := LoadKEMFile(missing); err == nil {
		t.Fatal("missing kem key loaded")
	}
	short := filepath.Join(t.TempDir(), "short")
	if err := os.WriteFile(short, make([]byte, 31), 0400); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKEMFile(short); err == nil {
		t.Fatal("short kem key loaded")
	}
}
