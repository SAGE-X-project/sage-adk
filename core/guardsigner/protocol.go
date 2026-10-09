// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	adkerrors "github.com/sage-x-project/sage-adk/pkg/errors"
)

// ErrDenied refuses unavailable, unauthorized or malformed signer operations.
// It does not reveal key bytes, message bytes or the reason for refusal.
var ErrDenied = &adkerrors.Error{Category: adkerrors.CategorySecurity, Code: "GUARD_SIGNER_DENIED", Message: "guard signer operation denied"}

// Signing domains used by SAGE 0.10.0 Ed25519 signatures on the native path.
const (
	DomainIntent       = "sage-execution-intent|0.10.0\x00"
	DomainResult       = "sage-tool-result|0.10.0\x00"
	DomainWireRequest  = "sage-wire-request|0.10.0\n"
	DomainWireResponse = "sage-wire-response|0.10.0\n"
	DomainCompletion   = "sage-hpke-complete|0.10.0\n"
)

// Roles group domains that one key may sign. A key configured only for
// "intent" cannot sign a tool result or a transport envelope. The separate
// "kem" role enables X25519 key agreement with Config.KEM and signs nothing.
var roleDomains = map[string][]string{
	"intent":    {DomainIntent},
	"result":    {DomainResult},
	"transport": {DomainWireRequest, DomainWireResponse, DomainCompletion},
}

const (
	magic          = "SGS1"
	opPublicKey    = byte(1)
	opSign         = byte(2)
	opKEMPublic    = byte(3)
	opECDH         = byte(4)
	statusOK       = byte(0)
	statusDenied   = byte(1)
	headerBytes    = 9
	maxMessage     = 1<<20 + 64
	minTimeout     = time.Millisecond
	maxTimeout     = 30 * time.Second
	maxConnections = 1024
)

// writeFrame writes magic, one kind byte, a big-endian length and the payload.
func writeFrame(w io.Writer, kind byte, payload []byte) error {
	if len(payload) > maxMessage {
		return ErrDenied
	}
	frame := make([]byte, headerBytes+len(payload))
	copy(frame, magic)
	frame[4] = kind
	binary.BigEndian.PutUint32(frame[5:headerBytes], uint32(len(payload)))
	copy(frame[headerBytes:], payload)
	_, err := w.Write(frame)
	return err
}

// readFrame reads one bounded frame. Each connection carries exactly one
// request frame and one response frame.
func readFrame(r io.Reader) (byte, []byte, error) {
	header := make([]byte, headerBytes)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0, nil, ErrDenied
	}
	if !bytes.Equal(header[:4], []byte(magic)) {
		return 0, nil, ErrDenied
	}
	n := binary.BigEndian.Uint32(header[5:])
	if n > maxMessage {
		return 0, nil, ErrDenied
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, ErrDenied
	}
	return header[4], payload, nil
}

func validTimeout(t time.Duration) bool { return t >= minTimeout && t <= maxTimeout }
