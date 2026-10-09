//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner

import (
	"crypto/ecdh"
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
	if err = writePrivate(path, private.Seed()); err != nil {
		return nil, err
	}
	return public, nil
}

// GenerateKEMFile creates a new 32-byte X25519 private key file readable only
// by the current account. It never overwrites and returns the public key.
func GenerateKEMFile(path string) ([]byte, error) {
	k, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, ErrDenied
	}
	private := k.Bytes()
	defer zero(private)
	if err = writePrivate(path, private); err != nil {
		return nil, err
	}
	return k.PublicKey().Bytes(), nil
}

// LoadKEMFile reads a key created by GenerateKEMFile with the same file checks
// as LoadKeyFile. The caller owns and should erase the returned bytes.
func LoadKEMFile(path string) ([]byte, error) {
	raw, err := readPrivate(path)
	if err != nil {
		return nil, err
	}
	if _, err = ecdh.X25519().NewPrivateKey(raw); err != nil {
		zero(raw)
		return nil, ErrDenied
	}
	return raw, nil
}

func writePrivate(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0400)
	if err != nil {
		return ErrDenied
	}
	_, writeErr := f.Write(data)
	syncErr := f.Sync()
	closeErr := f.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return ErrDenied
	}
	return nil
}

// LoadKeyFile reads a seed file created by GenerateKeyFile. The file must be a
// regular file owned by the current account with no group or other access.
func LoadKeyFile(path string) (ed25519.PrivateKey, error) {
	seed, err := readPrivate(path)
	if err != nil {
		return nil, err
	}
	defer zero(seed)
	return ed25519.NewKeyFromSeed(seed), nil
}

// readPrivate returns the 32 bytes of a private regular key file.
func readPrivate(path string) ([]byte, error) {
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
	raw := make([]byte, ed25519.SeedSize)
	if _, err = io.ReadFull(f, raw); err != nil {
		zero(raw)
		return nil, ErrDenied
	}
	return raw, nil
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
