//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner

import (
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"os"
	"syscall"
)

// GenerateKeyFile creates a new 32-byte Ed25519 seed file readable only by the
// current account. It never overwrites an existing file and returns only the
// public key, which the operator registers separately.
func GenerateKeyFile(path string) (ed25519.PublicKey, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, ErrDenied
	}
	defer zero(private)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0400)
	if err != nil {
		return nil, ErrDenied
	}
	_, writeErr := f.Write(private.Seed())
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return nil, ErrDenied
	}
	return public, nil
}

// LoadKeyFile reads a seed file created by GenerateKeyFile. The file must be a
// regular file owned by the current account with no group or other access.
func LoadKeyFile(path string) (ed25519.PrivateKey, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, ErrDenied
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() != ed25519.SeedSize {
		return nil, ErrDenied
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != uint32(os.Getuid()) {
		return nil, ErrDenied
	}
	seed := make([]byte, ed25519.SeedSize)
	defer zero(seed)
	if _, err = io.ReadFull(f, seed); err != nil {
		return nil, ErrDenied
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
