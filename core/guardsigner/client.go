// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner

import (
	"context"
	"crypto/ed25519"
	"net"
	"path/filepath"
	"time"
)

// Client requests signatures from one signer socket. It satisfies both
// guardservices.Ed25519Backend and the core hpke.Ed25519Custody010 port.
// Callers must still verify each signature against the registered key; the
// guardservices adapters and core endpoint do so.
type Client struct {
	path      string
	signerUID uint32
	timeout   time.Duration
}

// NewClient binds an absolute socket path and the account that must serve it.
func NewClient(path string, signerUID uint32, timeout time.Duration) (*Client, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || !validTimeout(timeout) {
		return nil, ErrDenied
	}
	return &Client{path: path, signerUID: signerUID, timeout: timeout}, nil
}

// PublicKey asks the signer for its public key.
func (c *Client) PublicKey(ctx context.Context) (ed25519.PublicKey, error) {
	out, err := c.call(ctx, opPublicKey, nil)
	if err != nil || len(out) != ed25519.PublicKeySize {
		return nil, ErrDenied
	}
	return ed25519.PublicKey(out), nil
}

// Sign asks the signer to sign message exactly.
func (c *Client) Sign(ctx context.Context, message []byte) ([]byte, error) {
	out, err := c.call(ctx, opSign, message)
	if err != nil || len(out) != ed25519.SignatureSize {
		return nil, ErrDenied
	}
	return out, nil
}

// KEM returns the X25519 key agreement view of this signer, satisfying the
// core hpke.X25519Custody010 port. The signer must hold the "kem" role.
func (c *Client) KEM() *KEMClient { return &KEMClient{client: c} }

// KEMClient asks the signer for its X25519 public key and for one X25519
// shared value. It never receives the private key.
type KEMClient struct{ client *Client }

// PublicKey returns the signer's X25519 public key.
func (k *KEMClient) PublicKey(ctx context.Context) ([]byte, error) {
	if k == nil {
		return nil, ErrDenied
	}
	out, err := k.client.call(ctx, opKEMPublic, nil)
	if err != nil || len(out) != 32 {
		return nil, ErrDenied
	}
	return out, nil
}

// ECDH returns the X25519 shared value with a 32-byte peer public key.
func (k *KEMClient) ECDH(ctx context.Context, peer []byte) ([]byte, error) {
	if k == nil || len(peer) != 32 {
		return nil, ErrDenied
	}
	out, err := k.client.call(ctx, opECDH, peer)
	if err != nil || len(out) != 32 {
		return nil, ErrDenied
	}
	return out, nil
}

func (c *Client) call(ctx context.Context, op byte, payload []byte) (out []byte, err error) {
	defer func() {
		if recover() != nil {
			out, err = nil, ErrDenied
		}
	}()
	if c == nil || ctx == nil || ctx.Err() != nil || len(payload) > maxMessage {
		return nil, ErrDenied
	}
	dialer := net.Dialer{Timeout: c.timeout}
	raw, err := dialer.DialContext(ctx, "unix", c.path)
	if err != nil {
		return nil, ErrDenied
	}
	conn := raw.(*net.UnixConn)
	defer func() { _ = conn.Close() }()
	deadline := time.Now().Add(c.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if conn.SetDeadline(deadline) != nil {
		return nil, ErrDenied
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Unix(1, 0)) })
	defer stop()
	if uid, e := peerUID(conn); e != nil || uid != c.signerUID {
		return nil, ErrDenied
	}
	if writeFrame(conn, op, payload) != nil {
		return nil, ErrDenied
	}
	status, body, e := readFrame(conn)
	if e != nil || status != statusOK || ctx.Err() != nil {
		return nil, ErrDenied
	}
	return body, nil
}
