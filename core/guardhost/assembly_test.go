//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardhost_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/guardhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

// occupy places a foreign directory where new host state would be created, so
// creating that state fails as it would on a reused or tampered directory.
func occupy(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name, "foreign"), 0700); err != nil {
		t.Fatal(err)
	}
}

// Every piece of new receiver state must be created fresh; occupied state,
// an unusable Registry configuration, an unknown issuer or missing artifacts
// fail closed before the host serves.
func TestReceiverAssemblyRefusesUnusableState(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	cases := map[string]func(*guardhost.ReceiverConfig){
		"occupied-intent-journal":    func(c *guardhost.ReceiverConfig) { occupy(t, c.Dir, "registry-intent") },
		"occupied-result-journal":    func(c *guardhost.ReceiverConfig) { occupy(t, c.Dir, "registry-result") },
		"occupied-transport-journal": func(c *guardhost.ReceiverConfig) { occupy(t, c.Dir, "registry-transport") },
		"occupied-replay-journal":    func(c *guardhost.ReceiverConfig) { occupy(t, c.Dir, "replay") },
		"occupied-ledger":            func(c *guardhost.ReceiverConfig) { occupy(t, c.Dir, "ledger") },
		"empty-registry-config":      func(c *guardhost.ReceiverConfig) { c.Registry = r.Config{} },
		"malformed-issuer":           func(c *guardhost.ReceiverConfig) { c.Issuer = "not-a-did" },
		"missing-artifacts":          func(c *guardhost.ReceiverConfig) { c.Operation.Directory = filepath.Join(c.Dir, "missing") },
		"zero-bounds":                func(c *guardhost.ReceiverConfig) { c.Bounds = g.MCPHostBounds{} },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			c := w.receiverConfig(t)
			change(&c)
			if rcv, err := guardhost.OpenReceiver(ctx, c); err == nil || rcv != nil {
				t.Fatal("receiver opened")
			}
		})
	}
	// The unchanged configuration still opens, so each refusal came from its
	// own change.
	rcv, err := guardhost.OpenReceiver(ctx, w.receiverConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	if err = rcv.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestCallerAssemblyRefusesUnusableState(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	cases := map[string]func(*guardhost.CallerConfig){
		// The capture store reopens its own directory, so only a non-directory
		// at its path is refused.
		"capture-store-not-a-directory": func(c *guardhost.CallerConfig) {
			if err := os.WriteFile(filepath.Join(c.Dir, "originals"), nil, 0600); err != nil {
				t.Fatal(err)
			}
		},
		"occupied-intent-journal":    func(c *guardhost.CallerConfig) { occupy(t, c.Dir, "registry-intent") },
		"occupied-result-journal":    func(c *guardhost.CallerConfig) { occupy(t, c.Dir, "registry-result") },
		"occupied-transport-journal": func(c *guardhost.CallerConfig) { occupy(t, c.Dir, "registry-transport") },
		"malformed-recipient": func(c *guardhost.CallerConfig) {
			c.Recipient, c.Connection.Recipient = "not-a-did", "not-a-did"
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			c := w.callerConfig(t)
			change(&c)
			if cl, err := guardhost.OpenCaller(ctx, c); err == nil || cl != nil {
				t.Fatal("caller opened")
			}
		})
	}
}

// Problems that only appear when a call runs (missing artifacts, unusable
// client host bounds, empty originals) end the call without a result and
// close its connection.
func TestCallerRefusesCallsWithUnusablePorts(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	call := func(t *testing.T, c guardhost.CallerConfig, inputs [][]byte) {
		t.Helper()
		caller, err := guardhost.OpenCaller(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = caller.Close() }()
		w.clock.mono.Add(361000)
		local, remote := net.Pipe()
		defer func() { _ = remote.Close() }()
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if out, err := caller.Call(callCtx, inputs, proposal(), local); err == nil || out != nil {
			t.Fatal("call accepted")
		}
		// The call closed its end of the connection.
		_ = remote.SetReadDeadline(time.Now().Add(time.Second))
		if _, err := remote.Read(make([]byte, 1)); err == nil {
			t.Fatal("connection left open")
		}
	}
	t.Run("missing-artifacts", func(t *testing.T) {
		c := w.callerConfig(t)
		c.Operation.Directory = filepath.Join(c.Dir, "missing")
		call(t, c, [][]byte{[]byte("add")})
	})
	t.Run("zero-client-bounds", func(t *testing.T) {
		c := w.callerConfig(t)
		c.Bounds = g.MCPClientHostBounds{}
		call(t, c, [][]byte{[]byte("add")})
	})
	t.Run("empty-originals", func(t *testing.T) {
		call(t, w.callerConfig(t), [][]byte{})
	})
}
