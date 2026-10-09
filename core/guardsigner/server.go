// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"errors"
	"net"
	"sync"
	"time"
)

// Config is protected signer administration, never host, model or peer input.
// Key is copied; the caller should discard its own copy. Roles select allowed
// signing domains; "kem" enables X25519 key agreement with KEM, the 32-byte
// X25519 private key, which is required for and only accepted with that role.
// AllowedUIDs lists host accounts that may request operations.
type Config struct {
	Key            ed25519.PrivateKey
	KEM            []byte
	Roles          []string
	AllowedUIDs    []uint32
	Timeout        time.Duration
	MaxConnections int
}

// Server signs allowed domains for allowlisted local peers. Its zero value is
// invalid. One request is served per connection.
type Server struct {
	mu      sync.Mutex
	key     ed25519.PrivateKey
	public  ed25519.PublicKey
	domains [][]byte
	kem     *ecdh.PrivateKey
	allowed map[uint32]bool
	timeout time.Duration
	slots   chan struct{}
	closed  bool
}

// NewServer validates configuration and copies the key.
func NewServer(c Config) (*Server, error) {
	if len(c.Key) != ed25519.PrivateKeySize || len(c.Roles) == 0 || len(c.AllowedUIDs) == 0 || !validTimeout(c.Timeout) || c.MaxConnections < 1 || c.MaxConnections > maxConnections {
		return nil, ErrDenied
	}
	key := append(ed25519.PrivateKey(nil), c.Key...)
	// Reject a key whose embedded public half does not match its seed.
	if !bytes.Equal(ed25519.NewKeyFromSeed(key.Seed()), key) {
		return nil, ErrDenied
	}
	s := &Server{key: key, public: key.Public().(ed25519.PublicKey), allowed: map[uint32]bool{}, timeout: c.Timeout, slots: make(chan struct{}, c.MaxConnections)}
	seen := map[string]bool{}
	for _, role := range c.Roles {
		if role == "kem" && !seen[role] {
			seen[role] = true
			continue
		}
		domains, ok := roleDomains[role]
		if !ok || seen[role] {
			return nil, ErrDenied
		}
		seen[role] = true
		for _, d := range domains {
			s.domains = append(s.domains, []byte(d))
		}
	}
	if seen["kem"] != (len(c.KEM) != 0) {
		return nil, ErrDenied
	}
	if seen["kem"] {
		kem, err := ecdh.X25519().NewPrivateKey(c.KEM)
		if err != nil {
			return nil, ErrDenied
		}
		s.kem = kem
	}
	for _, uid := range c.AllowedUIDs {
		s.allowed[uid] = true
	}
	return s, nil
}

// PublicKey returns the public half of the held key.
func (s *Server) PublicKey() ed25519.PublicKey {
	if s == nil {
		return nil
	}
	return append(ed25519.PublicKey(nil), s.public...)
}

// Serve accepts connections until ctx ends or the listener fails. It closes
// the listener on return and waits for in-flight requests.
func (s *Server) Serve(ctx context.Context, l *net.UnixListener) error {
	if s == nil || l == nil || ctx == nil {
		return ErrDenied
	}
	stop := context.AfterFunc(ctx, func() { _ = l.Close() })
	defer stop()
	defer func() { _ = l.Close() }()
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		conn, err := l.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return ErrDenied
		}
		select {
		case s.slots <- struct{}{}:
		default:
			_ = conn.Close()
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-s.slots }()
			s.handle(conn)
		}()
	}
}

func (s *Server) handle(conn *net.UnixConn) {
	defer func() { _ = conn.Close() }()
	defer func() { _ = recover() }()
	if conn.SetDeadline(time.Now().Add(s.timeout)) != nil {
		return
	}
	uid, err := peerUID(conn)
	if err != nil || !s.allowed[uid] {
		_ = writeFrame(conn, statusDenied, nil)
		return
	}
	op, payload, err := readFrame(conn)
	if err != nil {
		_ = writeFrame(conn, statusDenied, nil)
		return
	}
	out, err := s.answer(op, payload)
	if err != nil {
		_ = writeFrame(conn, statusDenied, nil)
		return
	}
	_ = writeFrame(conn, statusOK, out)
}

func (s *Server) answer(op byte, payload []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrDenied
	}
	switch op {
	case opPublicKey:
		if len(payload) != 0 {
			return nil, ErrDenied
		}
		return append([]byte(nil), s.public...), nil
	case opSign:
		if !s.permitted(payload) {
			return nil, ErrDenied
		}
		return ed25519.Sign(s.key, payload), nil
	case opKEMPublic:
		if s.kem == nil || len(payload) != 0 {
			return nil, ErrDenied
		}
		return s.kem.PublicKey().Bytes(), nil
	case opECDH:
		if s.kem == nil || len(payload) != 32 {
			return nil, ErrDenied
		}
		peer, err := ecdh.X25519().NewPublicKey(payload)
		if err != nil {
			return nil, ErrDenied
		}
		// crypto/ecdh refuses an all-zero (low-order) result.
		return s.kem.ECDH(peer)
	}
	return nil, ErrDenied
}

// permitted requires a known domain and a nonempty body after it.
func (s *Server) permitted(message []byte) bool {
	for _, d := range s.domains {
		if len(message) > len(d) && bytes.HasPrefix(message, d) {
			return true
		}
	}
	return false
}

// Close retires the key; later requests are denied. It does not stop Serve.
func (s *Server) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.key {
		s.key[i] = 0
	}
	s.key = nil
	s.kem = nil
	s.closed = true
}
