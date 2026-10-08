//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardhost_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/guardhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

type running struct {
	receiver *guardhost.Receiver
	listener net.Listener
	stop     context.CancelFunc
	done     chan error
}

func serve(t *testing.T, r *guardhost.Receiver) *running {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	s := &running{receiver: r, listener: l, stop: stop, done: make(chan error, 1)}
	go func() { s.done <- r.Serve(ctx, l, 1) }()
	return s
}

func (s *running) close(t *testing.T) {
	t.Helper()
	s.stop()
	if err := <-s.done; err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("serve: %v", err)
	}
	if err := s.receiver.Close(context.Background()); err != nil {
		t.Error(err)
	}
}

func callOnce(t *testing.T, c *guardhost.Caller, addr string, p g.IntentProposal) ([]byte, error) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.Call(ctx, [][]byte{[]byte("add")}, p, conn)
}

// A receiver restarted on its existing state recovers its nonempty replay
// journal and serves without a new quarantine; missing state never recovers.
func TestReceiverRestartRecoversExistingState(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	rc := w.receiverConfig(t)
	r, err := guardhost.OpenReceiver(ctx, rc)
	if err != nil {
		t.Fatal(err)
	}
	caller, err := guardhost.OpenCaller(ctx, w.callerConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caller.Close() }()
	w.clock.mono.Add(361000)
	s := serve(t, r)
	if out, err := callOnce(t, caller, s.listener.Addr().String(), proposal()); err != nil || string(out) != `{"output":5,"success":true}` {
		t.Fatalf("first call: %s %v", out, err)
	}
	s.close(t)

	rc.Create = false
	r, err = guardhost.OpenReceiver(ctx, rc)
	if err != nil {
		t.Fatal("recovery refused:", err)
	}
	s = serve(t, r)
	defer s.close(t)
	if out, err := callOnce(t, caller, s.listener.Addr().String(), proposal()); err != nil || string(out) != `{"output":5,"success":true}` {
		t.Fatalf("call after restart: %s %v", out, err)
	}

	fresh := w.receiverConfig(t)
	fresh.Create = false
	if r, err := guardhost.OpenReceiver(ctx, fresh); err == nil || r != nil {
		t.Fatal("missing state recovered")
	}
}

// A deactivated receiver identity stops calls even though both hosts stay up.
func TestDeactivatedReceiverIdentityRefusesCall(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	r, err := guardhost.OpenReceiver(ctx, w.receiverConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	s := serve(t, r)
	defer s.close(t)
	caller, err := guardhost.OpenCaller(ctx, w.callerConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = caller.Close() }()
	w.clock.mono.Add(361000)
	w.source.revoked.Store(true)
	if out, err := callOnce(t, caller, s.listener.Addr().String(), proposal()); err == nil || out != nil {
		t.Fatal("call accepted after receiver deactivation")
	}
}

func TestCallerRefusesInvalidCalls(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	caller, err := guardhost.OpenCaller(ctx, w.callerConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	pipe := func() net.Conn { a, b := net.Pipe(); _ = b.Close(); return a }
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if out, err := caller.Call(cancelled, [][]byte{[]byte("add")}, proposal(), pipe()); err == nil || out != nil {
		t.Fatal("cancelled call accepted")
	}
	if out, err := caller.Call(ctx, [][]byte{[]byte("add")}, proposal(), nil); err == nil || out != nil {
		t.Fatal("call without a connection accepted")
	}
	if out, err := caller.Call(ctx, nil, proposal(), pipe()); err == nil || out != nil {
		t.Fatal("call without original inputs accepted")
	}
	if err = caller.Close(); err != nil {
		t.Fatal(err)
	}
	if out, err := caller.Call(ctx, [][]byte{[]byte("add")}, proposal(), pipe()); err == nil || out != nil {
		t.Fatal("closed caller called")
	}
	var none *guardhost.Caller
	if none.Close() != nil {
		t.Fatal("nil caller close")
	}
	if out, err := none.Call(ctx, [][]byte{[]byte("add")}, proposal(), pipe()); err == nil || out != nil {
		t.Fatal("nil caller called")
	}
	var rnone *guardhost.Receiver
	if rnone.Close(ctx) != nil || rnone.Serve(ctx, nil, 1) == nil {
		t.Fatal("nil receiver")
	}
	for name, change := range map[string]func(*guardhost.CallerConfig){
		"relative-dir":   func(c *guardhost.CallerConfig) { c.Dir = "state" },
		"no-recipient":   func(c *guardhost.CallerConfig) { c.Recipient = "" },
		"responder-role": func(c *guardhost.CallerConfig) { c.Connection.Role = g.MCPResponder },
		"unsigned":       func(c *guardhost.CallerConfig) { c.Approval = nil },
		"no-source":      func(c *guardhost.CallerConfig) { c.Source = nil },
	} {
		c := w.callerConfig(t)
		change(&c)
		if cl, err := guardhost.OpenCaller(ctx, c); err == nil || cl != nil {
			t.Fatalf("%s caller opened", name)
		}
	}
}
