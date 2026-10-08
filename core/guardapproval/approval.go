// SPDX-License-Identifier: LGPL-3.0-or-later

// Package guardapproval checks that an exact-operation policy and component
// manifest were approved by a pinned operator key before a host loads them.
// An approval signs the policy and manifest commitments together with a
// sequence number. A durable ledger refuses a lower sequence, and refuses a
// different approval at the accepted sequence, so a newer approval supersedes
// every older one.
//
// The signature domain is local ADK host administration. It is not a SAGE
// protocol message and grants no execution by itself: the host must still
// verify the original request, the intent, the Registry key and the loaded
// component. The operator key must stay outside the host, model and plugins.
package guardapproval

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"

	adkerrors "github.com/sage-x-project/sage-adk/pkg/errors"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// ErrDenied refuses a missing, unsigned, unpinned, mismatched or superseded
// approval without revealing which check failed.
var ErrDenied = &adkerrors.Error{Category: adkerrors.CategorySecurity, Code: "OPERATION_APPROVAL_DENIED", Message: "operation approval missing or not accepted"}

// Version identifies this approval format.
const Version = "sage-adk-approval/1"

// Domain prefixes the canonical approval body before signing.
const Domain = "sage-adk-operation-approval|1\x00"

const maxApprovalBytes = 4096

// maxSequence keeps sequence numbers exact as JSON numbers (RFC 8785).
const maxSequence = 1<<53 - 1

func validSequence(n uint64) bool { return n >= 1 && n <= maxSequence }

// Approval is the verified content of one approval document.
type Approval struct {
	Approver string `json:"approver"`
	Policy   string `json:"policy"`
	Manifest string `json:"manifest"`
	Sequence uint64 `json:"sequence"`
	Version  string `json:"version"`
}

type document struct {
	Approval
	Signature string `json:"signature"`
}

// commitments returns the core policy and manifest commitments, which already
// cover rules.json and every artifact digest named by the descriptors.
func commitments(policy, manifest []byte) (string, string, error) {
	canonical, err := g.Canonicalize(policy)
	if err != nil {
		return "", "", ErrDenied
	}
	p, err := g.PolicyCommitment(canonical)
	if err != nil {
		return "", "", ErrDenied
	}
	m, err := g.ManifestCommitment(manifest)
	if err != nil {
		return "", "", ErrDenied
	}
	return p, m, nil
}

func signedBytes(a Approval) ([]byte, error) {
	raw, err := json.Marshal(a)
	if err != nil {
		return nil, ErrDenied
	}
	canonical, err := g.Canonicalize(raw)
	if err != nil {
		return nil, ErrDenied
	}
	return append([]byte(Domain), canonical...), nil
}

// Sign creates an approval document for the exact policy and manifest. It is
// meant for the operator's own tool, never for the host.
func Sign(key ed25519.PrivateKey, policy, manifest []byte, sequence uint64) ([]byte, error) {
	if len(key) != ed25519.PrivateKeySize || !validSequence(sequence) {
		return nil, ErrDenied
	}
	p, m, err := commitments(policy, manifest)
	if err != nil {
		return nil, err
	}
	a := Approval{Approver: hex.EncodeToString(key.Public().(ed25519.PublicKey)), Policy: p, Manifest: m, Sequence: sequence, Version: Version}
	msg, err := signedBytes(a)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(document{a, base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, msg))})
	if err != nil {
		return nil, ErrDenied
	}
	return g.Canonicalize(raw)
}

// Check verifies an approval document against pinned approver keys and the
// exact policy and manifest. It does not consult or advance a ledger.
func Check(raw, policy, manifest []byte, approvers []ed25519.PublicKey) (Approval, error) {
	if len(raw) == 0 || len(raw) > maxApprovalBytes || len(approvers) == 0 {
		return Approval{}, ErrDenied
	}
	canonical, err := g.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Approval{}, ErrDenied
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields) != 6 {
		return Approval{}, ErrDenied
	}
	var d document
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&d) != nil || d.Version != Version || !validSequence(d.Sequence) {
		return Approval{}, ErrDenied
	}
	public, err := hex.DecodeString(d.Approver)
	if err != nil || len(public) != ed25519.PublicKeySize || hex.EncodeToString(public) != d.Approver {
		return Approval{}, ErrDenied
	}
	pinned := false
	for _, k := range approvers {
		pinned = pinned || bytes.Equal(k, public)
	}
	signature, err := base64.RawURLEncoding.DecodeString(d.Signature)
	if !pinned || err != nil || len(signature) != ed25519.SignatureSize {
		return Approval{}, ErrDenied
	}
	msg, err := signedBytes(d.Approval)
	if err != nil || !ed25519.Verify(public, msg, signature) {
		return Approval{}, ErrDenied
	}
	p, m, err := commitments(policy, manifest)
	if err != nil || p != d.Policy || m != d.Manifest {
		return Approval{}, ErrDenied
	}
	return d.Approval, nil
}
