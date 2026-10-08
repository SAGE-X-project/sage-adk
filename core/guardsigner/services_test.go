//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/guardservices"
	"github.com/sage-x-project/sage-adk/core/guardsigner"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

const principal = "did:sage:web:agent.example:alice"
const keyid = principal + "#signing-1"

// registrySource is a synthetic Registry used only to exercise the signer
// through the guardservices adapters. It is not an authoritative Source.
type registrySource struct {
	key   ed25519.PublicKey
	clock *guardservices.SystemClock
}

func (s registrySource) Read(_ context.Context, did string) (r.Snapshot, error) {
	stamp, err := s.clock.Now()
	if err != nil {
		return r.Snapshot{}, err
	}
	return r.Snapshot{Source: "synthetic-fixture", Registry: "web:agent.example", Network: "local", DID: did, Version: "1", Digest: strings.Repeat("a", 64), State: "active", Ready: true, Validated: true, Finalized: true, AcquiredMS: stamp.MonoMS, Keys: []r.Key{{Name: "signing-1", Alg: "ed25519", Material: hex.EncodeToString(s.key), State: "accepted"}}}, nil
}

func authority(t *testing.T, key ed25519.PublicKey) *g.RegistryAuthority {
	t.Helper()
	clock := guardservices.NewSystemClock()
	j, err := r.OpenJournal(filepath.Join(t.TempDir(), "registry"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = j.Close() })
	gate, err := r.NewGate(r.Config{Source: "synthetic-fixture", Registry: "web:agent.example", Network: "local"}, registrySource{key, clock}, clock, j)
	if err != nil {
		t.Fatal(err)
	}
	a, err := g.NewRegistryAuthority(gate, principal, keyid)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func serve(t *testing.T, key ed25519.PrivateKey, roles ...string) *guardsigner.Client {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "gsvc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	s, err := guardsigner.NewServer(guardsigner.Config{Key: key, Roles: roles, AllowedUIDs: []uint32{uint32(os.Getuid())}, Timeout: time.Second, MaxConnections: 4})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "s")
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, l) }()
	t.Cleanup(func() { cancel(); <-done; s.Close() })
	c, err := guardsigner.NewClient(path, uint32(os.Getuid()), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func body() []byte {
	return []byte(fmt.Sprintf(`{"alg":"ed25519","issuer":%q,"keyid":%q,"version":"0.10.0"}`, principal, keyid))
}

// The guardservices adapters accept the signer client as their custody backend
// and sign only within the roles that the signer process allows.
func TestSignerBacksRegistryBoundAdapters(t *testing.T) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub := key.Public().(ed25519.PublicKey)
	ctx := context.Background()
	a := authority(t, pub)
	intentOnly := serve(t, key, "intent")
	intent, err := guardservices.NewIntentSigner(ctx, a, principal, keyid, intentOnly)
	if err != nil {
		t.Fatal(err)
	}
	msg := append([]byte(guardsigner.DomainIntent), body()...)
	proof, err := intent.Sign(ctx, keyid, msg)
	if err != nil || !ed25519.Verify(pub, msg, proof) {
		t.Fatal("intent signature", err)
	}
	result, err := guardservices.NewResultSigner(ctx, a, principal, keyid, intentOnly)
	if err != nil {
		t.Fatal(err)
	}
	if proof, err = result.Sign(ctx, keyid, append([]byte(guardsigner.DomainResult), body()...)); err == nil || proof != nil {
		t.Fatal("intent-only signer produced a result signature")
	}
	_, other, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if s, err := guardservices.NewIntentSigner(ctx, a, principal, keyid, serve(t, other, "intent")); err == nil || s != nil {
		t.Fatal("unregistered signer key accepted")
	}
}
