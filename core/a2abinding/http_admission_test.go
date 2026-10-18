package a2abinding

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

type protectedMemoryConn struct {
	writes, closes atomic.Int32
	short          bool
	err            error
}

func (c *protectedMemoryConn) Read([]byte) (int, error) { return 0, io.EOF }
func (c *protectedMemoryConn) Write(b []byte) (int, error) {
	c.writes.Add(1)
	if c.err != nil {
		return 0, c.err
	}
	if c.short {
		return len(b) - 1, nil
	}
	return len(b), nil
}
func (c *protectedMemoryConn) Close() error                     { c.closes.Add(1); return nil }
func (c *protectedMemoryConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (c *protectedMemoryConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (c *protectedMemoryConn) SetDeadline(time.Time) error      { return nil }
func (c *protectedMemoryConn) SetReadDeadline(time.Time) error  { return nil }
func (c *protectedMemoryConn) SetWriteDeadline(time.Time) error { return nil }

func TestProtectedHTTPFinalAdmission010(t *testing.T) {
	for _, mode := range []string{"success", "nil-context", "canceled", "closed", "unavailable-source", "verify-error", "nil-card", "budget", "rollback", "card-expired", "card-key-expired", "wire-expired", "wire-future", "nil-writer", "empty-frame", "large-frame", "short-write", "write-error"} {
		t.Run(mode, func(t *testing.T) {
			pair := protectedSessions(t, endpoint, true)
			owned, e := pair.left.TakeHTTPApplication(endpoint)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(owned.Close)
			state := &protectedHTTPState{owner: owned}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			input := ctx
			writer := &protectedMemoryConn{}
			var output io.Writer = writer
			frame := []byte("inert prepared frame")
			window := wireWindow{Created: 100, Expires: 130}
			calls := 0
			verify := func() (*VerifiedCard, error) {
				calls++
				switch mode {
				case "verify-error":
					return nil, errors.New("fixture card unavailable")
				case "nil-card":
					return nil, nil
				case "budget":
					pair.fixture.advance(5001, 1)
				case "rollback":
					pair.fixture.advance(-1, 0)
				}
				bounds := cardBounds{issued: 100, expires: 105}
				if mode == "card-expired" {
					bounds.expires = 100
				}
				if mode == "card-key-expired" {
					bounds.keyHasExpiry = true
					bounds.keyExpires = 100
				}
				return &VerifiedCard{bounds: bounds}, nil
			}
			switch mode {
			case "nil-context":
				input = nil
			case "canceled":
				cancel()
			case "closed":
				state.closing.Store(true)
			case "unavailable-source":
				pair.fixture.setMode("read-error")
			case "wire-expired":
				window.Expires = 100
			case "wire-future":
				window.Created = 131
				window.Expires = 150
			case "nil-writer":
				output = nil
			case "empty-frame":
				frame = nil
			case "large-frame":
				frame = make([]byte, protectedHeaderLimit+protectedBodyLimit+1)
			case "short-write":
				writer.short = true
			case "write-error":
				writer.err = errors.New("fixture write unavailable")
			}
			e = state.emitFrame(input, output, frame, verify, window)
			if mode == "success" {
				if e != nil || writer.writes.Load() != 1 || calls != 1 {
					t.Fatal("missing exact local admission", e, writer.writes.Load(), calls)
				}
			} else {
				if e != ErrRejected {
					t.Fatal("invalid final admission accepted", e)
				}
				expected := int32(0)
				if mode == "short-write" || mode == "write-error" {
					expected = 1
				}
				if writer.writes.Load() != expected {
					t.Fatal("unexpected write/retry", writer.writes.Load(), expected)
				}
			}
		})
	}
}

func TestProtectedHTTPClockAndCardBounds010(t *testing.T) {
	m, source, key := fixture(t)
	expires := int64(102)
	source.record.Keys[0].Expires = &expires
	card, e := verifier(t, source).Verify(context.Background(), signed(t, m, key))
	if e != nil || card == nil || !card.bounds.live(101) || card.bounds.live(102) {
		t.Fatal("selected signing key expiry missing", e)
	}
	expires = 1000
	if card.bounds.live(102) {
		t.Fatal("source mutation changed privately retained bound")
	}
	for _, raw := range []string{"not-json", `{"created":-1,"expires":1}`, `{"created":2,"expires":2}`, `{"created":1,"expires":302}`} {
		if _, e := protectedWindow([]byte(raw)); e != ErrRejected {
			t.Fatal("invalid local envelope bounds accepted", raw)
		}
	}
	w, e := protectedWindow([]byte(`{"created":100,"expires":130}`))
	if e != nil || !w.live(registry010.Stamp{Unix: 100}) || w.live(registry010.Stamp{Unix: 130}) || (wireWindow{Created: 131, Expires: 150}).live(registry010.Stamp{Unix: 100}) {
		t.Fatal("wire skew/deadline changed", e)
	}
}

func TestProtectedHTTPShutdownDuringDial010(t *testing.T) {
	for _, lateConn := range []bool{false, true} {
		t.Run(map[bool]string{false: "canceled-dial", true: "late-connection"}[lateConn], func(t *testing.T) {
			pair := protectedSessions(t, endpoint, true)
			owner, e := NewHTTPTransportOwner(HTTPTransportConfig{Session: pair.left, URL: endpoint, TLS: &tls.Config{}, TTL: 30, Timeout: 5 * time.Second})
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(owner.Close)
			copied := *owner
			if copied.state != owner.state {
				t.Fatal("wrapper copies split state")
			}
			started := make(chan struct{}, 1)
			conn := &protectedMemoryConn{}
			dials := atomic.Int32{}
			owner.state.dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
				dials.Add(1)
				started <- struct{}{}
				<-ctx.Done()
				if lateConn {
					return conn, nil
				}
				return nil, ctx.Err()
			}
			call := ProtectedCall{target: Interface{URL: endpoint}, recipient: peer, revalidate: func(context.Context) (*VerifiedCard, error) { return nil, ErrRejected }}
			result := make(chan error, 2)
			exchange := func(o *HTTPTransportOwner) {
				_, e := o.Exchange(context.Background(), call, func(context.Context) error { return nil })
				result <- e
			}
			go exchange(owner)
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("unit dial not entered")
			}
			go exchange(&copied)
			closed := make(chan struct{})
			go func() { copied.Close(); close(closed) }()
			select {
			case <-closed:
			case <-time.After(2 * time.Second):
				t.Fatal("Close waited for five-second dial instead of canceling setup")
			}
			for range 2 {
				select {
				case e := <-result:
					if e != ErrRejected {
						t.Fatal(e)
					}
				case <-time.After(time.Second):
					t.Fatal("copied exchange did not stop")
				}
			}
			if dials.Load() != 1 || conn.writes.Load() != 0 || (lateConn && conn.closes.Load() == 0) || !owner.state.closing.Load() {
				t.Fatal("alias/dial cancellation permitted late write or retry", dials.Load(), conn.writes.Load(), conn.closes.Load())
			}
		})
	}
	var missing *HTTPTransportOwner
	missing.Close()
	(&HTTPTransportOwner{}).Close()
	var noIngress *HTTPIngress
	noIngress.Close()
	(&HTTPIngress{}).Close()
}
