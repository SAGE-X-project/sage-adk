// SPDX-License-Identifier: LGPL-3.0-or-later
package a2abinding

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

const protectedHeaderLimit = 16384
const protectedBodyLimit = 32768

// HTTPTransportConfig selects one exact peer session and a bounded TLS 1.3,
// HTTP/1.1 carriage. Dialer is trusted routing configuration, not a peer hint.
// Clock/Source/custody are provisioned in the session and Client verifier;
// their Unix time meaning must agree. Providers must be bounded/non-reentrant.
type HTTPTransportConfig struct {
	Session *hpke.AuthenticatedCompletion010
	URL     string
	TLS     *tls.Config
	Dialer  *net.Dialer
	TTL     int64
	Timeout time.Duration
}

// HTTPTransportOwner imports the core's exclusive application owner. Every
// value copy shares private state. It is message protection, not an Execution
// Guard, live Registry attestation or authority to perform application effects.
type HTTPTransportOwner struct{ state *protectedHTTPState }

type protectedHTTPState struct {
	op                 sync.Mutex
	ioMu               sync.Mutex
	closing            atomic.Bool
	owner              *hpke.HTTPApplicationOwner010
	conn               net.Conn
	cancel             context.CancelFunc
	listener           net.Listener
	target             *url.URL
	exact, local, peer string
	tlsConfig          *tls.Config
	dial               func(context.Context, string, string) (net.Conn, error)
	ttl                int64
	timeout            time.Duration
}

func protectedTarget(raw string) (*url.URL, error) {
	u, e := url.Parse(raw)
	if e != nil || len(raw) > 8192 || u.Scheme != "https" || u.Host == "" || u.Path == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" || u.String() != raw || u.Host != strings.ToLower(u.Host) || strings.HasSuffix(u.Host, ":443") || strings.ContainsAny(raw, "\"\\ \r\n\t") {
		return nil, ErrRejected
	}
	for _, b := range []byte(raw) {
		if b < 33 || b > 126 {
			return nil, ErrRejected
		}
	}
	return u, nil
}

func protectedTLS(c *tls.Config) (*tls.Config, error) {
	if c == nil || c.InsecureSkipVerify || c.VerifyConnection != nil || c.VerifyPeerCertificate != nil || c.GetConfigForClient != nil || c.KeyLogWriter != nil || c.Renegotiation != tls.RenegotiateNever || c.Rand != nil || c.Time != nil || (c.MinVersion != 0 && c.MinVersion != tls.VersionTLS13) || (c.MaxVersion != 0 && c.MaxVersion != tls.VersionTLS13) {
		return nil, ErrRejected
	}
	for _, p := range c.NextProtos {
		if p != "http/1.1" {
			return nil, ErrRejected
		}
	}
	result := c.Clone()
	result.MinVersion, result.MaxVersion = tls.VersionTLS13, tls.VersionTLS13
	result.NextProtos = []string{"http/1.1"}
	result.SessionTicketsDisabled = true
	result.ClientSessionCache = nil
	if c.RootCAs != nil {
		result.RootCAs = c.RootCAs.Clone()
	}
	if c.ClientCAs != nil {
		result.ClientCAs = c.ClientCAs.Clone()
	}
	result.Certificates = append([]tls.Certificate(nil), c.Certificates...)
	for i := range result.Certificates {
		cert := &result.Certificates[i]
		cert.Certificate = append([][]byte(nil), cert.Certificate...)
		for j := range cert.Certificate {
			cert.Certificate[j] = append([]byte(nil), cert.Certificate[j]...)
		}
		cert.OCSPStaple = append([]byte(nil), cert.OCSPStaple...)
		cert.SignedCertificateTimestamps = append([][]byte(nil), cert.SignedCertificateTimestamps...)
		for j := range cert.SignedCertificateTimestamps {
			cert.SignedCertificateTimestamps[j] = append([]byte(nil), cert.SignedCertificateTimestamps[j]...)
		}
		// PrivateKey and optional certificate callbacks remain trusted custody.
	}
	return result, nil
}

func newProtectedHTTP(c HTTPTransportConfig) (*protectedHTTPState, error) {
	target, e := protectedTarget(c.URL)
	if e != nil || c.Session == nil || c.TTL < 1 || c.TTL > 300 || c.Timeout <= 0 || c.Timeout > time.Minute {
		return nil, ErrRejected
	}
	tc, e := protectedTLS(c.TLS)
	if e != nil {
		return nil, ErrRejected
	}
	owned, e := c.Session.TakeHTTPApplication(c.URL)
	if e != nil {
		return nil, ErrRejected
	}
	local, peer, e := owned.Participants()
	if e != nil {
		owned.Close()
		return nil, ErrRejected
	}
	d := net.Dialer{}
	if c.Dialer != nil {
		d = *c.Dialer
	}
	return &protectedHTTPState{owner: owned, target: target, exact: c.URL, local: local, peer: peer, tlsConfig: tc, dial: d.DialContext, ttl: c.TTL, timeout: c.Timeout}, nil
}

// NewHTTPTransportOwner consumes an unused completed session. Its private owner
// must not be used through a second adapter. It never uses an unsigned fallback.
func NewHTTPTransportOwner(c HTTPTransportConfig) (*HTTPTransportOwner, error) {
	if c.TLS != nil && c.TLS.ServerName != "" {
		u, e := protectedTarget(c.URL)
		if e != nil || c.TLS.ServerName != u.Hostname() {
			return nil, ErrRejected
		}
	}
	state, e := newProtectedHTTP(c)
	if e != nil {
		return nil, e
	}
	state.tlsConfig.ServerName = state.target.Hostname()
	return &HTTPTransportOwner{state: state}, nil
}

// interrupt signals closure before waiting for any operation/provider/IO lock.
// A previous operation's cancel hook captures its own connection only.
func (s *protectedHTTPState) interrupt() {
	s.closing.Store(true)
	s.ioMu.Lock()
	conn, cancel, listener := s.conn, s.cancel, s.listener
	s.ioMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if conn != nil {
		_ = conn.Close()
	}
	if listener != nil {
		_ = listener.Close()
	}
}

// Close interrupts active IO and then serializes core key cleanup. A trusted
// synchronous provider must honor its bounded contract; it cannot be preempted.
func (o *HTTPTransportOwner) Close() {
	if o == nil || o.state == nil {
		return
	}
	s := o.state
	s.interrupt()
	s.op.Lock()
	defer s.op.Unlock()
	s.owner.Close()
}

// Register cancellation before DNS/TCP setup, not only after a connection exists.
func (s *protectedHTTPState) publishCancel(cancel context.CancelFunc) error {
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	if s.closing.Load() {
		return ErrRejected
	}
	s.cancel = cancel
	return nil
}

func (s *protectedHTTPState) publishConn(conn net.Conn, cancel context.CancelFunc) error {
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	if s.closing.Load() {
		_ = conn.Close()
		return ErrRejected
	}
	s.conn, s.cancel = conn, cancel
	return nil
}
func (s *protectedHTTPState) clearConn() {
	s.ioMu.Lock()
	s.conn = nil
	s.cancel = nil
	s.ioMu.Unlock()
}

// wireWindow is private data extracted only from a locally sealed or core-
// accepted envelope. It is never accepted as peer-provided authority.
type wireWindow struct {
	Created int64 `json:"created"`
	Expires int64 `json:"expires"`
}

func protectedWindow(body []byte) (wireWindow, error) {
	var w wireWindow
	if json.Unmarshal(body, &w) != nil || w.Created < 0 || w.Expires <= w.Created || w.Expires-w.Created > 300 {
		return w, ErrRejected
	}
	return w, nil
}
func (w wireWindow) live(t registry010.Stamp) bool {
	return w.Created <= t.Unix+30 && t.Unix < w.Expires
}

// round includes every blocking final authority/card check in one five-second
// interval. Its last clock sample is diagnostic and is not returned as a grant.
func (s *protectedHTTPState) round(ctx context.Context, verify func() (*VerifiedCard, error), windows ...wireWindow) error {
	if ctx == nil || ctx.Err() != nil || s.closing.Load() || verify == nil {
		return ErrRejected
	}
	start, e := s.owner.LocalStamp()
	if e != nil {
		return ErrRejected
	}
	if _, e = s.owner.Observe(ctx); e != nil {
		return ErrRejected
	}
	card, e := verify()
	if e != nil || card == nil {
		return ErrRejected
	}
	end, e := s.owner.LocalStamp()
	if e != nil || ctx.Err() != nil || s.closing.Load() || end.MonoMS < start.MonoMS || end.Unix < start.Unix || end.MonoMS-start.MonoMS > 5000 || !card.bounds.live(end.Unix) {
		return ErrRejected
	}
	for _, w := range windows {
		if !w.live(end) {
			return ErrRejected
		}
	}
	return nil
}

// Exchange implements the protected transport interface. It uses one verified
// TLS connection and one pre-serialized application write, with no retries.
// A possible partial write/effect remains uncertain; failure retires this owner.
func (o *HTTPTransportOwner) Exchange(parent context.Context, call ProtectedCall, finalCheck func(context.Context) error) (result AuthenticatedResponse, err error) {
	if o == nil || o.state == nil || parent == nil || parent.Err() != nil || finalCheck == nil || call.revalidate == nil {
		return result, ErrRejected
	}
	s := o.state
	s.op.Lock()
	defer s.op.Unlock()
	if s.closing.Load() || call.target.URL != s.exact || call.recipient != s.peer {
		return result, ErrRejected
	}
	ctx, cancel := context.WithTimeout(parent, s.timeout)
	defer cancel()
	good := false
	defer func() {
		if !good {
			s.interrupt()
			s.owner.Close()
		}
		s.clearConn()
	}()
	if s.publishCancel(cancel) != nil {
		return result, ErrRejected
	}
	address := s.target.Host
	if s.target.Port() == "" {
		address = net.JoinHostPort(s.target.Hostname(), "443")
	}
	raw, e := s.dial(ctx, "tcp", address)
	if e != nil {
		return result, ErrRejected
	}
	defer func() { _ = raw.Close() }()
	if s.publishConn(raw, cancel) != nil {
		return result, ErrRejected
	}
	stop := context.AfterFunc(ctx, func() { _ = raw.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if raw.SetDeadline(deadline) != nil {
		return result, ErrRejected
	}
	conn := tls.Client(raw, s.tlsConfig)
	if conn.HandshakeContext(ctx) != nil || conn.ConnectionState().NegotiatedProtocol != "http/1.1" {
		return result, ErrRejected
	}
	message, e := s.owner.SealRequest(ctx, call.Payload(), call.Metadata(), s.ttl)
	if e != nil {
		return result, ErrRejected
	}
	requestWindow, e := protectedWindow(message.Body)
	if e != nil {
		return result, ErrRejected
	}
	frame, e := protectedRequestFrame(ctx, message)
	if e != nil {
		return result, ErrRejected
	}
	verify := func() (*VerifiedCard, error) {
		if finalCheck(ctx) != nil {
			return nil, ErrRejected
		}
		return call.revalidate(ctx)
	}
	if s.emitFrame(ctx, conn, frame, verify, requestWindow) != nil {
		return result, ErrRejected
	}
	response, e := protectedReadResponse(conn)
	if e != nil {
		return result, ErrRejected
	}
	view, e := s.owner.OpenResponse(ctx, response)
	if e != nil || view == nil || view.Sender() != s.peer || view.Recipient() != s.local || !view.Success() || view.Error() != "" || response.Status != http.StatusOK {
		return result, ErrRejected
	}
	responseWindow, e := protectedWindow(response.Body)
	if e != nil {
		return result, ErrRejected
	}
	if s.round(ctx, func() (*VerifiedCard, error) { return call.revalidate(ctx) }, requestWindow, responseWindow) != nil || ctx.Err() != nil || s.closing.Load() {
		return result, ErrRejected
	}
	result = AuthenticatedResponse{Sender: view.Sender(), Payload: view.Payload()}
	good = true
	return result, nil
}

// Only the privately owned verified TLS connection reaches this boundary in
// production. Tests may supply an inert writer to observe admission and errors.
func (s *protectedHTTPState) emitFrame(ctx context.Context, writer io.Writer, frame []byte, verify func() (*VerifiedCard, error), windows ...wireWindow) error {
	if writer == nil || len(frame) == 0 || len(frame) > protectedHeaderLimit+protectedBodyLimit || s.round(ctx, verify, windows...) != nil || ctx.Err() != nil || s.closing.Load() {
		return ErrRejected
	}
	// One direct write after the last sample, no user/provider callbacks or retry.
	if n, e := writer.Write(frame); e != nil || n != len(frame) {
		return ErrRejected
	}
	return nil
}

func protectedRequestFrame(ctx context.Context, m hpke.HTTPMessage010) ([]byte, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, m.Target, bytes.NewReader(m.Body))
	if e != nil || m.Method != http.MethodPost || req.URL.Host != m.Authority || m.Status != 0 {
		return nil, ErrRejected
	}
	req.Host = m.Authority
	req.Close = true
	for _, h := range m.Headers {
		req.Header.Add(h[0], h[1])
	}
	var frame bytes.Buffer
	if req.Write(&frame) != nil || frame.Len() > protectedHeaderLimit+protectedBodyLimit {
		return nil, ErrRejected
	}
	return frame.Bytes(), nil
}

// Read framing before net/http can combine duplicate lengths or normalize fields.
// This is an explicit narrow fixed-length HTTP/1.1 subset, not a general proxy.
func protectedReadFrame(input io.Reader) ([]byte, [][2]string, error) {
	r := bufio.NewReaderSize(io.LimitReader(input, protectedHeaderLimit+protectedBodyLimit+1), 4096)
	var frame bytes.Buffer
	headers := [][2]string{}
	seen := map[string]bool{}
	first := true
	length := -1
	for {
		line, e := r.ReadString('\n')
		if e != nil || !strings.HasSuffix(line, "\r\n") || frame.Len()+len(line) > protectedHeaderLimit {
			return nil, nil, ErrRejected
		}
		frame.WriteString(line)
		if first {
			first = false
			continue
		}
		if line == "\r\n" {
			break
		}
		name, value, ok := strings.Cut(strings.TrimSuffix(line, "\r\n"), ":")
		lower := strings.ToLower(name)
		if !ok || name == "" || len(headers) >= 256 || seen[lower] {
			return nil, nil, ErrRejected
		}
		for _, b := range []byte(name) {
			if !((b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b))) {
				return nil, nil, ErrRejected
			}
		}
		value = strings.Trim(value, " \t")
		for _, b := range []byte(value) {
			if b < 32 || b > 126 {
				return nil, nil, ErrRejected
			}
		}
		switch lower {
		case "transfer-encoding", "trailer", "content-encoding", "upgrade", "expect", "proxy-connection", "forwarded":
			return nil, nil, ErrRejected
		case "connection":
			if value != "close" {
				return nil, nil, ErrRejected
			}
		case "content-length":
			n, e := strconv.Atoi(value)
			if e != nil || n < 1 || n > protectedBodyLimit || strconv.Itoa(n) != value {
				return nil, nil, ErrRejected
			}
			length = n
		}
		if strings.HasPrefix(lower, "x-forwarded-") {
			return nil, nil, ErrRejected
		}
		seen[lower] = true
		headers = append(headers, [2]string{name, value})
	}
	if length < 1 {
		return nil, nil, ErrRejected
	}
	body := make([]byte, length)
	if _, e := io.ReadFull(r, body); e != nil {
		return nil, nil, ErrRejected
	}
	frame.Write(body)
	return frame.Bytes(), headers, nil
}

func protectedReadResponse(input io.Reader) (hpke.HTTPMessage010, error) {
	frame, headers, e := protectedReadFrame(input)
	if e != nil {
		return hpke.HTTPMessage010{}, ErrRejected
	}
	response, e := http.ReadResponse(bufio.NewReader(bytes.NewReader(frame)), nil)
	if e != nil {
		return hpke.HTTPMessage010{}, ErrRejected
	}
	defer func() { _ = response.Body.Close() }()
	if response.Proto != "HTTP/1.1" || response.StatusCode < 200 || response.StatusCode > 599 || response.StatusCode == http.StatusNoContent || response.ContentLength < 1 || response.ContentLength > protectedBodyLimit || len(response.TransferEncoding) > 0 || len(response.Trailer) > 0 {
		return hpke.HTTPMessage010{}, ErrRejected
	}
	body, e := io.ReadAll(io.LimitReader(response.Body, protectedBodyLimit+1))
	if e != nil || int64(len(body)) != response.ContentLength {
		return hpke.HTTPMessage010{}, ErrRejected
	}
	return hpke.HTTPMessage010{Status: response.StatusCode, Headers: headers, Body: body}, nil
}
