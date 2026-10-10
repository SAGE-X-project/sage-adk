//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardapproval

import (
	"bytes"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

// The operator tool refuses to sign descriptors that the core would not
// commit to, so an approval can never name an unverifiable policy or manifest.
func TestSignRefusesInvalidDescriptors(t *testing.T) {
	k := operatorKey(t)
	p, m := descriptors(epoch1)
	for name, c := range map[string]struct{ policy, manifest []byte }{
		"policy-not-json":     {[]byte("{"), m},
		"policy-incomplete":   {[]byte(`{"version":"0.10.0"}`), m},
		"manifest-not-json":   {p, []byte("{")},
		"manifest-incomplete": {p, []byte(`{"version":"0.10.0"}`)},
	} {
		if out, err := Sign(k, c.policy, c.manifest, 1); err == nil || out != nil {
			t.Fatalf("%s signed", name)
		}
	}
	raw := sign(t, k, epoch1, 1)
	if _, err := Check(raw, p, []byte(`{"version":"0.10.0"}`), []ed25519.PublicKey{pub(k)}); err == nil {
		t.Fatal("invalid manifest checked")
	}
}

// A ledger file readable by others, larger than any approval, or replaced by
// a directory is refused on recovery.
func TestLedgerRefusesUnsafeRecordFiles(t *testing.T) {
	k := operatorKey(t)
	p, m := descriptors(epoch1)
	for name, damage := range map[string]func(string) error{
		"group-readable": func(path string) error { return os.Chmod(path, 0640) },
		"oversized": func(path string) error {
			return os.WriteFile(path, bytes.Repeat([]byte(" "), maxApprovalBytes+1), 0600)
		},
		"directory": func(path string) error {
			if err := os.Remove(path); err != nil {
				return err
			}
			return os.Mkdir(path, 0700)
		},
	} {
		dir := newLedger(t)
		l, err := OpenLedger(dir, true)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = l.Accept(sign(t, k, epoch1, 1), p, m, []ed25519.PublicKey{pub(k)}); err != nil {
			t.Fatal(err)
		}
		if err = damage(filepath.Join(dir, ledgerName)); err != nil {
			t.Fatal(err)
		}
		if _, err = OpenLedger(dir, false); err == nil {
			t.Fatalf("%s ledger accepted", name)
		}
	}
}

// A refused or unrecorded approval leaves the ledger unchanged: a check
// failure, an occupied temporary path and an unreplaceable record all refuse
// without advancing the current approval.
func TestLedgerAcceptFailuresKeepCurrent(t *testing.T) {
	k := operatorKey(t)
	pinned := []ed25519.PublicKey{pub(k)}
	p1, m1 := descriptors(epoch1)
	p2, m2 := descriptors(epoch2)
	dir := newLedger(t)
	l, err := OpenLedger(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = l.Accept(sign(t, k, epoch1, 1), p1, m1, pinned); err != nil {
		t.Fatal(err)
	}
	if _, err = l.Accept(sign(t, k, epoch2, 2), p1, m1, pinned); err == nil {
		t.Fatal("approval for other descriptors accepted")
	}
	// An occupied temporary path cannot be cleared or created.
	tmp := filepath.Join(dir, ledgerName+".tmp")
	if err = os.MkdirAll(filepath.Join(tmp, "foreign"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = l.Accept(sign(t, k, epoch2, 2), p2, m2, pinned); err == nil {
		t.Fatal("approval accepted without a durable record")
	}
	if err = os.RemoveAll(tmp); err != nil {
		t.Fatal(err)
	}
	// A record path that cannot be replaced refuses after writing the
	// temporary file, and removes it.
	record := filepath.Join(dir, ledgerName)
	if err = os.Remove(record); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(record, "foreign"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = l.Accept(sign(t, k, epoch2, 2), p2, m2, pinned); err == nil {
		t.Fatal("approval accepted without replacing the record")
	}
	if _, err = os.Lstat(tmp); !os.IsNotExist(err) {
		t.Fatal("temporary record left behind", err)
	}
	if a, ok := l.Current(); !ok || a.Sequence != 1 {
		t.Fatal("refused approvals changed the current approval", a)
	}
	var none *Ledger
	if _, ok := none.Current(); ok {
		t.Fatal("nil ledger has an approval")
	}
}
