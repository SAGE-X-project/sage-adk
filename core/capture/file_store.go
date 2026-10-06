//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

const maxRecordBytes = 2 << 20

// FileStore retains originals in exclusive files in a private directory. It does
// not isolate other code running under the same OS account or prevent rollback.
// The host must protect the directory, ancestors and capabilities, and define
// retention for the plaintext input. Existing directory mode must be exactly 0700.
type FileStore struct{ root *os.Root }
type record struct {
	Version   string   `json:"version"`
	RequestID string   `json:"request_id"`
	Inputs    [][]byte `json:"inputs"`
}

// OpenFileStore opens or creates the final private directory; all ancestors must
// already exist and be protected. The supplied directory cannot be a symlink.
func OpenFileStore(dir string) (*FileStore, error) {
	created := os.Mkdir(dir, 0700)
	if created != nil && !os.IsExist(created) {
		return nil, ErrCapture
	}
	if created == nil {
		parent, e := os.Open(filepath.Dir(dir))
		if e != nil {
			return nil, ErrCapture
		}
		e = parent.Sync()
		closeErr := parent.Close()
		if e != nil || closeErr != nil {
			return nil, ErrCapture
		}
	}
	info, e := os.Lstat(dir)
	if e != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return nil, ErrCapture
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		return nil, ErrCapture
	}
	actual, e := root.Stat(".")
	if e != nil || !os.SameFile(info, actual) || actual.Mode().Perm() != 0700 {
		root.Close()
		return nil, ErrCapture
	}
	return &FileStore{root: root}, nil
}

// Close releases the directory handle. Operations after Close refuse processing.
func (s *FileStore) Close() error {
	if s == nil || s.root == nil {
		return ErrCapture
	}
	return s.root.Close()
}
func recordName(id string) (string, error) {
	u, e := uuid.Parse(id)
	if e != nil || u.String() != id || u.Version() != 4 || u.Variant() != uuid.RFC4122 {
		return "", ErrCapture
	}
	return id + ".json", nil
}
func (s *FileStore) private() bool {
	if s == nil || s.root == nil {
		return false
	}
	i, e := s.root.Stat(".")
	return e == nil && i.IsDir() && i.Mode().Perm() == 0700
}
func validFile(f *os.File) bool {
	i, e := f.Stat()
	if e != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || i.Size() > maxRecordBytes {
		return false
	}
	st, ok := i.Sys().(*syscall.Stat_t)
	return ok && st.Nlink == 1
}

// Create persists a new record and synchronizes file and directory. Failure may
// leave a permanent incomplete record; it must be reconciled by trusted operators
// and never overwritten by a retry.
func (s *FileStore) Create(ctx context.Context, id string, inputs [][]byte) error {
	if !active(ctx) || !s.private() {
		return ErrCapture
	}
	name, e := recordName(id)
	if e != nil {
		return e
	}
	if _, e = g.OriginalCommitment(inputs); e != nil {
		return ErrCapture
	}
	raw, e := json.Marshal(record{"0.10.0", id, clone(inputs)})
	if e != nil || len(raw) > maxRecordBytes {
		return ErrCapture
	}
	f, e := s.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if e != nil {
		return ErrCapture
	}
	defer f.Close()
	if !validFile(f) || !active(ctx) {
		return ErrCapture
	}
	n, e := f.Write(raw)
	if e != nil || n != len(raw) || !active(ctx) {
		return ErrCapture
	}
	if e = f.Sync(); e != nil {
		return ErrCapture
	}
	if e = f.Close(); e != nil {
		return ErrCapture
	}
	d, e := s.root.Open(".")
	if e != nil {
		return ErrCapture
	}
	defer d.Close()
	if e = d.Sync(); e != nil || !active(ctx) {
		return ErrCapture
	}
	return nil
}

// Load performs a bounded no-follow regular-file read and validates the exact
// record schema and original UTF-8 limits. It returns independently owned bytes.
func (s *FileStore) Load(ctx context.Context, id string) ([][]byte, error) {
	if !active(ctx) || !s.private() {
		return nil, ErrCapture
	}
	name, e := recordName(id)
	if e != nil {
		return nil, e
	}
	f, e := s.root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, ErrCapture
	}
	defer f.Close()
	if !validFile(f) {
		return nil, ErrCapture
	}
	raw, e := io.ReadAll(io.LimitReader(f, maxRecordBytes+1))
	if e != nil || len(raw) > maxRecordBytes || !active(ctx) {
		return nil, ErrCapture
	}
	var r record
	if json.Unmarshal(raw, &r) != nil || r.Version != "0.10.0" || r.RequestID != id || r.Inputs == nil {
		return nil, ErrCapture
	}
	canonical, e := json.Marshal(r)
	if e != nil || !bytes.Equal(raw, canonical) {
		return nil, ErrCapture
	}
	if _, e = g.OriginalCommitment(r.Inputs); e != nil {
		return nil, ErrCapture
	}
	return clone(r.Inputs), nil
}
