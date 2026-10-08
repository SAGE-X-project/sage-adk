//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardapproval

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

const ledgerName = "approval-ledger.json"

// Ledger durably retains the highest accepted approval. It lives in a private
// directory (mode 0700, owned by the host account) that the host must protect
// from model/plugin code. It detects rollback only while this file is intact;
// restoring an older copy of the directory is outside what it can see.
type Ledger struct {
	mu   sync.Mutex
	dir  string
	last *Approval
}

// OpenLedger opens or creates the ledger directory. create must be true only
// for a new deployment; recovery never recreates a missing ledger.
func OpenLedger(dir string, create bool) (*Ledger, error) {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return nil, ErrDenied
	}
	if create {
		if err := os.Mkdir(dir, 0700); err != nil {
			return nil, ErrDenied
		}
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return nil, ErrDenied
	}
	if st, ok := info.Sys().(*syscall.Stat_t); !ok || st.Uid != uint32(os.Getuid()) {
		return nil, ErrDenied
	}
	l := &Ledger{dir: dir}
	last, err := l.read()
	switch {
	case err == nil:
		l.last = &last
	case create && errors.Is(err, fs.ErrNotExist):
	default:
		return nil, ErrDenied
	}
	return l, nil
}

func (l *Ledger) read() (Approval, error) {
	f, err := os.OpenFile(filepath.Join(l.dir, ledgerName), os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return Approval{}, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > maxApprovalBytes {
		return Approval{}, ErrDenied
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxApprovalBytes+1))
	if err != nil {
		return Approval{}, ErrDenied
	}
	var a Approval
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&a) != nil || a.Version != Version || !validSequence(a.Sequence) {
		return Approval{}, ErrDenied
	}
	return a, nil
}

// Accept checks the approval and records it before returning. The same
// approval is accepted again after restart; a lower sequence, or different
// content at the current sequence, is refused.
func (l *Ledger) Accept(raw, policy, manifest []byte, approvers []ed25519.PublicKey) (Approval, error) {
	if l == nil {
		return Approval{}, ErrDenied
	}
	a, err := Check(raw, policy, manifest, approvers)
	if err != nil {
		return Approval{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.last != nil {
		if a.Sequence < l.last.Sequence || (a.Sequence == l.last.Sequence && a != *l.last) {
			return Approval{}, ErrDenied
		}
		if a == *l.last {
			return a, nil
		}
	}
	if err = l.write(a); err != nil {
		return Approval{}, ErrDenied
	}
	l.last = &a
	return a, nil
}

// Current returns the highest accepted approval, if any.
func (l *Ledger) Current() (Approval, bool) {
	if l == nil {
		return Approval{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.last == nil {
		return Approval{}, false
	}
	return *l.last, true
}

// write replaces the ledger with a synced temporary file and syncs the
// directory, so a crash leaves either the old or the new record.
func (l *Ledger) write(a Approval) error {
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	tmp := filepath.Join(l.dir, ledgerName+".tmp")
	_ = os.Remove(tmp)
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(raw)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(tmp)
		return ErrDenied
	}
	if err = os.Rename(tmp, filepath.Join(l.dir, ledgerName)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	d, err := os.Open(l.dir)
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}
