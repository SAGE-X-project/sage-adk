//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/guardsigner"
)

func buildSigner(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "adk-signer")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

// The signer runs as a separate OS process. The test signs one benign intent
// message through the socket, then stops the process with SIGTERM.
func TestSignerProcessServesAndStops(t *testing.T) {
	bin := buildSigner(t)
	dir, err := os.MkdirTemp("/tmp", "gsbin")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	key, socket := filepath.Join(dir, "seed"), filepath.Join(dir, "s")
	out, err := exec.Command(bin, "keygen", "-key", key).Output()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := hex.DecodeString(strings.TrimSpace(string(out)))
	if err != nil || len(pub) != ed25519.PublicKeySize {
		t.Fatal("keygen output", err)
	}
	uid := strconv.Itoa(os.Getuid())
	cmd := exec.Command(bin, "serve", "-key", key, "-socket", socket, "-allow-uid", uid, "-roles", "intent")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill() }()
	ready := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(stdout).ReadString('\n')
		ready <- line
	}()
	select {
	case line := <-ready:
		if line != "adk-signer ready "+hex.EncodeToString(pub)+"\n" {
			t.Fatalf("ready line %q stderr %s", line, stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("signer not ready")
	}
	info, err := os.Stat(socket)
	if err != nil || info.Mode().Perm() != 0660 {
		t.Fatal("socket mode", err)
	}
	c, err := guardsigner.NewClient(socket, uint32(os.Getuid()), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	msg := []byte(guardsigner.DomainIntent + `{"a":2,"b":3}`)
	sig, err := c.Sign(ctx, msg)
	if err != nil || !ed25519.Verify(pub, msg, sig) {
		t.Fatal("process signature", err)
	}
	if _, err = c.Sign(ctx, []byte(guardsigner.DomainResult+`{}`)); err == nil {
		t.Fatal("process signed outside its role")
	}
	if err = cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	if err = cmd.Wait(); err != nil {
		t.Fatalf("signer exit: %v stderr %s", err, stderr.String())
	}
	if _, err = c.PublicKey(ctx); err == nil {
		t.Fatal("stopped signer answered")
	}
}

func TestSignerCommandRefusals(t *testing.T) {
	dir := t.TempDir()
	open := filepath.Join(dir, "open")
	if err := os.WriteFile(open, make([]byte, 32), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	for name, args := range map[string][]string{
		"no-command":     {},
		"unknown":        {"sign"},
		"keygen-no-key":  {"keygen"},
		"keygen-exists":  {"keygen", "-key", open},
		"serve-missing":  {"serve", "-key", open},
		"serve-bad-uid":  {"serve", "-key", open, "-socket", filepath.Join(dir, "s"), "-allow-uid", "x", "-roles", "intent"},
		"serve-open-key": {"serve", "-key", open, "-socket", filepath.Join(dir, "s"), "-allow-uid", "1", "-roles", "intent"},
	} {
		if err := run(args, &out); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

// TestSignerAcrossAccounts is a client for an already running signer under a
// separate OS account. It runs only when a harness supplies the environment,
// for example a Linux container with distinct signer and host users.
func TestSignerAcrossAccounts(t *testing.T) {
	socket, uid, expect := os.Getenv("ADK_SIGNER_SOCKET"), os.Getenv("ADK_SIGNER_UID"), os.Getenv("ADK_SIGNER_EXPECT")
	if socket == "" {
		t.Skip("no external signer configured")
	}
	signer, err := strconv.ParseUint(uid, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	c, err := guardsigner.NewClient(socket, uint32(signer), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte(guardsigner.DomainIntent + `{"a":2,"b":3}`)
	pub, pubErr := c.PublicKey(context.Background())
	sig, signErr := c.Sign(context.Background(), msg)
	switch expect {
	case "signed":
		if pubErr != nil || signErr != nil || !ed25519.Verify(pub, msg, sig) {
			t.Fatal("allowed account was not served", pubErr, signErr)
		}
		t.Logf("uid %d signed with %s", os.Getuid(), hex.EncodeToString(pub))
	case "denied":
		if pubErr == nil || signErr == nil {
			t.Fatal("account outside the allowlist was served")
		}
		t.Logf("uid %d denied", os.Getuid())
	default:
		t.Fatal("ADK_SIGNER_EXPECT must be signed or denied")
	}
}
