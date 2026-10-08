//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardapproval

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func digest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func descriptors(epoch string) ([]byte, []byte) {
	rules := []byte(`{"tool":"calculator"}`)
	policy, _ := json.Marshal(map[string]any{"version": "0.10.0", "issuer": "did:sage:web:agent.example:alice", "epoch": epoch, "engine": "sage-adk/exact-operation/1", "artifacts": map[string]any{"version": "0.10.0", "files": []any{map[string]any{"path": "rules.json", "sha256": digest(rules)}}}})
	manifest, _ := json.Marshal(map[string]any{"version": "0.10.0", "files": []any{map[string]any{"path": "calculator.json", "sha256": digest([]byte("calculator"))}}})
	return policy, manifest
}

const epoch1 = "00000000-0000-4000-8000-000000000001"
const epoch2 = "00000000-0000-4000-8000-000000000002"

func operatorKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, k, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func pub(k ed25519.PrivateKey) ed25519.PublicKey { return k.Public().(ed25519.PublicKey) }

func sign(t *testing.T, k ed25519.PrivateKey, epoch string, seq uint64) []byte {
	t.Helper()
	p, m := descriptors(epoch)
	raw, err := Sign(k, p, m, seq)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestApprovalChecksExactCommitments(t *testing.T) {
	k := operatorKey(t)
	p, m := descriptors(epoch1)
	raw := sign(t, k, epoch1, 1)
	a, err := Check(raw, p, m, []ed25519.PublicKey{pub(k)})
	if err != nil || a.Sequence != 1 || a.Approver != hex.EncodeToString(pub(k)) {
		t.Fatal(a, err)
	}
	// Key rotation: either pinned approver is accepted.
	other := operatorKey(t)
	if _, err = Check(raw, p, m, []ed25519.PublicKey{pub(other), pub(k)}); err != nil {
		t.Fatal(err)
	}
}

func mutate(t *testing.T, raw []byte, field string, value any) []byte {
	t.Helper()
	var d map[string]any
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if value == nil {
		delete(d, field)
	} else {
		d[field] = value
	}
	out, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestApprovalRefusals(t *testing.T) {
	k := operatorKey(t)
	p, m := descriptors(epoch1)
	p2, m2 := descriptors(epoch2)
	raw := sign(t, k, epoch1, 1)
	pinned := []ed25519.PublicKey{pub(k)}
	otherRaw := sign(t, operatorKey(t), epoch1, 1)
	_, otherManifest := descriptors(epoch1)
	otherManifest = bytes.Replace(otherManifest, []byte("calculator.json"), []byte("calculator2.json"), 1)
	cases := map[string]struct {
		raw, policy, manifest []byte
		pinned                []ed25519.PublicKey
	}{
		"unpinned-approver":  {otherRaw, p, m, pinned},
		"no-pinned-keys":     {raw, p, m, nil},
		"other-policy":       {raw, p2, m, pinned},
		"other-manifest":     {raw, p, otherManifest, pinned},
		"other-epoch-pair":   {raw, p2, m2, pinned},
		"raised-sequence":    {mutate(t, raw, "sequence", 2), p, m, pinned},
		"wrong-version":      {mutate(t, raw, "version", "sage-adk-approval/2"), p, m, pinned},
		"missing-signature":  {mutate(t, raw, "signature", nil), p, m, pinned},
		"short-signature":    {mutate(t, raw, "signature", "AAAA"), p, m, pinned},
		"extra-field":        {mutate(t, raw, "note", "x"), p, m, pinned},
		"uppercase-approver": {mutate(t, raw, "approver", strings.ToUpper(hex.EncodeToString(pub(k)))), p, m, pinned},
		"zero-sequence":      {mutate(t, raw, "sequence", 0), p, m, pinned},
		"non-canonical":      {append([]byte(" "), raw...), p, m, pinned},
		"oversized":          {bytes.Repeat([]byte("a"), maxApprovalBytes+1), p, m, pinned},
		"empty":              {nil, p, m, pinned},
		"invalid-policy":     {raw, []byte(`{"version":"0.10.0"}`), m, pinned},
	}
	for name, c := range cases {
		if a, err := Check(c.raw, c.policy, c.manifest, c.pinned); err == nil {
			t.Fatalf("%s accepted: %+v", name, a)
		}
	}
	for name, seq := range map[string]uint64{"zero": 0, "inexact": maxSequence + 1} {
		if out, err := Sign(k, p, m, seq); err == nil || out != nil {
			t.Fatalf("%s sequence signed", name)
		}
	}
	if out, err := Sign(k[:32], p, m, 1); err == nil || out != nil {
		t.Fatal("short key signed")
	}
}

func newLedger(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "approvals")
}

func TestLedgerSupersedesAndRefusesRollback(t *testing.T) {
	k := operatorKey(t)
	pinned := []ed25519.PublicKey{pub(k)}
	p1, m1 := descriptors(epoch1)
	p2, m2 := descriptors(epoch2)
	dir := newLedger(t)
	l, err := OpenLedger(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := l.Current(); ok {
		t.Fatal("new ledger has an approval")
	}
	first := sign(t, k, epoch1, 1)
	if _, err = l.Accept(first, p1, m1, pinned); err != nil {
		t.Fatal(err)
	}
	if _, err = l.Accept(first, p1, m1, pinned); err != nil {
		t.Fatal("same approval refused", err)
	}
	if _, err = l.Accept(sign(t, k, epoch2, 1), p2, m2, pinned); err == nil {
		t.Fatal("different approval at the same sequence accepted")
	}
	if _, err = l.Accept(sign(t, k, epoch2, 2), p2, m2, pinned); err != nil {
		t.Fatal(err)
	}
	if _, err = l.Accept(first, p1, m1, pinned); err == nil {
		t.Fatal("superseded approval accepted")
	}
	// Restart keeps the highest accepted approval.
	reopened, err := OpenLedger(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := reopened.Current(); !ok || a.Sequence != 2 {
		t.Fatal("ledger not durable", a)
	}
	if _, err = reopened.Accept(first, p1, m1, pinned); err == nil {
		t.Fatal("rollback accepted after restart")
	}
	if _, err = reopened.Accept(sign(t, k, epoch2, 2), p2, m2, pinned); err != nil {
		t.Fatal("current approval refused after restart", err)
	}
	if _, err = OpenLedger(dir, true); err == nil {
		t.Fatal("existing ledger recreated")
	}
}

func TestLedgerOpenRefusals(t *testing.T) {
	if _, err := OpenLedger(newLedger(t), false); err == nil {
		t.Fatal("missing ledger recreated during recovery")
	}
	if _, err := OpenLedger("relative", true); err == nil {
		t.Fatal("relative path accepted")
	}
	open := newLedger(t)
	if err := os.Mkdir(open, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenLedger(open, false); err == nil {
		t.Fatal("shared directory accepted")
	}
	for name, content := range map[string]string{"garbage": "{", "unknown-field": `{"version":"sage-adk-approval/1","sequence":1,"x":1}`, "zero": `{"version":"sage-adk-approval/1","sequence":0}`} {
		dir := newLedger(t)
		l, err := OpenLedger(dir, true)
		if err != nil || l == nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, ledgerName), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = OpenLedger(dir, false); err == nil {
			t.Fatalf("%s ledger accepted", name)
		}
	}
	dir := newLedger(t)
	if _, err := OpenLedger(dir, true); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.WriteFile(target, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, ledgerName)); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenLedger(dir, false); err == nil {
		t.Fatal("symlinked ledger accepted")
	}
	var none *Ledger
	if _, err := none.Accept(nil, nil, nil, nil); err == nil {
		t.Fatal("nil ledger accepted")
	}
}
