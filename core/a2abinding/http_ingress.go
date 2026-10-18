// SPDX-License-Identifier: LGPL-3.0-or-later
package a2abinding

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
)

// HTTPIngressConfig transfers a completed responder session and the actual A2A
// policy Server. Transport selects exact target, TLS certificates and IO bounds;
// its Dialer must be absent. Effects still require the host Execution Guard.
type HTTPIngressConfig struct {
	Transport HTTPTransportConfig
	Server    *Server
}

// HTTPIngress owns a TLS-only, sequential, per-session listener. It offers no
// bare JSONRPC handler or alternate binding. Wrapper copies share all state.
// It is not a multi-agent discovery or multi-session listener implementation.
type HTTPIngress struct{ state *protectedIngressState }
type protectedIngressState struct {
	transport *protectedHTTPState
	server    *Server
	serving   atomic.Bool
}

// NewHTTPIngress consumes an unused core session for one configured local agent.
// ServeTLS takes exclusive ownership of its listener; no separate raw handler is
// exported. Trusted dependencies must not reenter this ingress from callbacks.
func NewHTTPIngress(c HTTPIngressConfig) (*HTTPIngress, error) {
	if c.Server == nil || c.Transport.Dialer != nil || c.Server.cfg.URL != c.Transport.URL || c.Transport.TLS == nil || (len(c.Transport.TLS.Certificates) == 0 && c.Transport.TLS.GetCertificate == nil) {
		return nil, ErrRejected
	}
	state, e := newProtectedHTTP(c.Transport)
	if e != nil {
		return nil, e
	}
	if state.local != c.Server.cfg.DID {
		state.owner.Close()
		return nil, ErrRejected
	}
	return &HTTPIngress{state: &protectedIngressState{transport: state, server: c.Server}}, nil
}

// Close interrupts accept/connection IO before serialized core cleanup. Closure
// is shared by every ingress value copy, including one whose ServeTLS is active.
func (in *HTTPIngress) Close() {
	if in == nil || in.state == nil {
		return
	}
	s := in.state.transport
	s.interrupt()
	s.op.Lock()
	defer s.op.Unlock()
	s.owner.Close()
}

// ServeTLS owns TLS handshakes, bounded raw HTTP framing and the only network
// application path to the real policy Server. It accepts one HTTP/1.1 request
// per connection with no keepalive, pipeline dispatch or protocol fallback.
// The fixed-length header block is inspected before net/http can normalize
// duplicate lengths; its parser is used only over already bounded owned bytes.
func (in *HTTPIngress) ServeTLS(ctx context.Context, listener net.Listener) error {
	if in == nil || in.state == nil || ctx == nil || ctx.Err() != nil || listener == nil {
		return ErrRejected
	}
	st := in.state
	s := st.transport
	if !st.serving.CompareAndSwap(false, true) {
		return ErrRejected
	}
	s.ioMu.Lock()
	if s.closing.Load() {
		s.ioMu.Unlock()
		_ = listener.Close()
		return ErrRejected
	}
	s.listener = listener
	s.ioMu.Unlock()
	stop := context.AfterFunc(ctx, s.interrupt)
	defer stop()
	defer func() { s.interrupt(); s.op.Lock(); s.owner.Close(); s.op.Unlock() }()
	for {
		conn, e := listener.Accept()
		if e != nil {
			return ErrRejected
		}
		st.connection(ctx, conn)
		if s.closing.Load() || ctx.Err() != nil {
			return ErrRejected
		}
	}
}

func (st *protectedIngressState) connection(parent context.Context, raw net.Conn) {
	s := st.transport
	s.op.Lock()
	defer s.op.Unlock()
	defer func() { _ = raw.Close() }()
	if s.closing.Load() || parent.Err() != nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, s.timeout)
	defer cancel()
	if s.publishConn(raw, cancel) != nil {
		return
	}
	defer s.clearConn()
	stop := context.AfterFunc(ctx, func() { _ = raw.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if raw.SetDeadline(deadline) != nil {
		return
	}
	conn := tls.Server(raw, s.tlsConfig)
	if conn.HandshakeContext(ctx) != nil || conn.ConnectionState().NegotiatedProtocol != "http/1.1" {
		return
	}
	message, e := protectedReadRequest(conn, s)
	if e != nil {
		protectedAuthenticationFailure(conn, http.StatusBadRequest)
		return
	}
	request, e := s.owner.OpenRequest(ctx, message)
	if e != nil || request == nil || request.Sender() != s.peer || request.Recipient() != s.local {
		protectedAuthenticationFailure(conn, http.StatusUnauthorized)
		return
	}
	emitted := false
	// Once acceptance succeeds, a failed/uncertain response cannot be retried or
	// converted into a fresh session/effect. Denial/replay remains consumed.
	defer func() {
		if !emitted {
			s.interrupt()
			s.owner.Close()
		}
	}()
	requestWindow, e := protectedWindow(message.Body)
	if e != nil {
		return
	}
	payload, e := st.server.handle(ctx, AuthenticatedRequest{URL: message.Target, Recipient: request.Recipient(), Metadata: request.Metadata(), Payload: request.Payload()}, func(checkCtx context.Context, verify func() (*VerifiedCard, error)) error {
		return s.round(checkCtx, verify, requestWindow)
	})
	success, code, status := true, "", http.StatusOK
	if e != nil {
		success, code, status = false, "policy_denied", http.StatusUnauthorized
		payload = nil
	}
	// A2AB02 is a request rule. Response metadata is intentionally independent.
	response, e := s.owner.SealResponse(ctx, request.ID(), payload, nil, success, code, s.ttl, status)
	if e != nil {
		return
	}
	responseWindow, e := protectedWindow(response.Body)
	if e != nil {
		return
	}
	frame, e := protectedResponseFrame(response)
	if e != nil {
		return
	}
	verify := func() (*VerifiedCard, error) {
		card, e := st.server.cfg.Card(ctx)
		if e != nil {
			return nil, ErrRejected
		}
		current, e := st.server.cfg.Verifier.Verify(ctx, append([]byte(nil), card...))
		if e != nil || current == nil || current.did != s.local {
			return nil, ErrRejected
		}
		found := false
		for _, i := range current.interfaces {
			found = found || i.URL == s.exact
		}
		if !found {
			return nil, ErrRejected
		}
		return current, nil
	}
	if s.emitFrame(ctx, conn, frame, verify, requestWindow, responseWindow) != nil {
		return
	}
	// A completely emitted signed policy denial also completes this request.
	// It need not retire the server session; the request remains terminal.
	emitted = true
}

func protectedReadRequest(input io.Reader, s *protectedHTTPState) (hpke.HTTPMessage010, error) {
	frame, headers, e := protectedReadFrame(input)
	if e != nil {
		return hpke.HTTPMessage010{}, ErrRejected
	}
	request, e := http.ReadRequest(bufio.NewReader(bytes.NewReader(frame)))
	if e != nil {
		return hpke.HTTPMessage010{}, ErrRejected
	}
	defer func() { _ = request.Body.Close() }()
	if request.Proto != "HTTP/1.1" || request.Method != http.MethodPost || request.Host != s.target.Host || request.URL.IsAbs() || request.URL.Opaque != "" || request.RequestURI != s.target.RequestURI() || "https://"+request.Host+request.RequestURI != s.exact || request.ContentLength < 1 || request.ContentLength > protectedBodyLimit || len(request.TransferEncoding) > 0 || len(request.Trailer) > 0 {
		return hpke.HTTPMessage010{}, ErrRejected
	}
	body, e := io.ReadAll(io.LimitReader(request.Body, protectedBodyLimit+1))
	if e != nil || int64(len(body)) != request.ContentLength {
		return hpke.HTTPMessage010{}, ErrRejected
	}
	return hpke.HTTPMessage010{Method: request.Method, Target: s.exact, Authority: request.Host, Headers: headers, Body: body}, nil
}

func protectedResponseFrame(m hpke.HTTPMessage010) ([]byte, error) {
	if m.Status < 200 || m.Status > 599 || m.Status == http.StatusNoContent || len(m.Body) < 1 || len(m.Body) > protectedBodyLimit {
		return nil, ErrRejected
	}
	response := &http.Response{StatusCode: m.Status, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(m.Body)), ContentLength: int64(len(m.Body)), Close: true}
	for _, h := range m.Headers {
		response.Header.Add(h[0], h[1])
	}
	var frame bytes.Buffer
	if response.Write(&frame) != nil || frame.Len() > protectedHeaderLimit+protectedBodyLimit {
		return nil, ErrRejected
	}
	return frame.Bytes(), nil
}

// Unsigned failure is never a JSONRPC result or permission to retry/downgrade.
func protectedAuthenticationFailure(conn net.Conn, status int) {
	if status != http.StatusBadRequest {
		status = http.StatusUnauthorized
	}
	response := &http.Response{StatusCode: status, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1, Header: http.Header{"Content-Type": []string{"text/plain"}}, Body: io.NopCloser(bytes.NewReader([]byte("authentication failed\n"))), ContentLength: 22, Close: true}
	var frame bytes.Buffer
	if response.Write(&frame) == nil {
		_, _ = conn.Write(frame.Bytes())
	}
}
