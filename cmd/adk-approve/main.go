//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later

// Command adk-approve is the operator's tool for approving one exact
// operation policy and component manifest for a protected ADK host.
//
//	adk-approve keygen -key PATH
//	adk-approve sign -key PATH -policy FILE -manifest FILE -sequence N -out FILE
//	adk-approve verify -approver HEX -policy FILE -manifest FILE -approval FILE
//
// Keep the operator key on the operator's own machine. The host only pins the
// public key and verifies approvals; it never holds this key.
package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"

	"github.com/sage-x-project/sage-adk/core/guardapproval"
	"github.com/sage-x-project/sage-adk/core/guardsigner"
)

const maxDescriptorBytes = 1 << 20

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "adk-approve:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: adk-approve keygen|sign|verify [flags]")
	}
	switch args[0] {
	case "keygen":
		return keygen(args[1:], out)
	case "sign":
		return sign(args[1:], out)
	case "verify":
		return verify(args[1:], out)
	}
	return errors.New("unknown command " + strconv.Quote(args[0]))
}

func keygen(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	key := fs.String("key", "", "new operator seed file (must not exist)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" || fs.NArg() != 0 {
		return errors.New("keygen requires -key")
	}
	public, err := guardsigner.GenerateKeyFile(*key)
	if err != nil {
		return errors.New("key file not created")
	}
	_, err = fmt.Fprintln(out, hex.EncodeToString(public))
	return err
}

func readBounded(path string) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, errors.New("cannot read " + strconv.Quote(path))
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, maxDescriptorBytes+1))
	if err != nil || len(b) > maxDescriptorBytes {
		return nil, errors.New("cannot read " + strconv.Quote(path))
	}
	return b, nil
}

func sign(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	key := fs.String("key", "", "operator seed file")
	policy := fs.String("policy", "", "policy descriptor JSON")
	manifest := fs.String("manifest", "", "component manifest JSON")
	sequence := fs.Uint64("sequence", 0, "approval sequence, higher than every earlier approval")
	output := fs.String("out", "", "new approval file (must not exist)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" || *policy == "" || *manifest == "" || *sequence == 0 || *output == "" || fs.NArg() != 0 {
		return errors.New("sign requires -key, -policy, -manifest, -sequence and -out")
	}
	p, err := readBounded(*policy)
	if err != nil {
		return err
	}
	m, err := readBounded(*manifest)
	if err != nil {
		return err
	}
	private, err := guardsigner.LoadKeyFile(*key)
	if err != nil {
		return errors.New("key file unavailable or not private")
	}
	approval, err := guardapproval.Sign(private, p, m, *sequence)
	for i := range private {
		private[i] = 0
	}
	if err != nil {
		return errors.New("policy or manifest is not a valid descriptor")
	}
	f, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0644)
	if err != nil {
		return errors.New("approval file not created")
	}
	_, writeErr := f.Write(approval)
	syncErr := f.Sync()
	if closeErr := f.Close(); writeErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(*output)
		return errors.New("approval file not written")
	}
	_, err = fmt.Fprintln(out, "approved sequence", *sequence)
	return err
}

func verify(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	approver := fs.String("approver", "", "pinned operator public key (hex)")
	policy := fs.String("policy", "", "policy descriptor JSON")
	manifest := fs.String("manifest", "", "component manifest JSON")
	approval := fs.String("approval", "", "approval file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	public, err := hex.DecodeString(*approver)
	if err != nil || len(public) != ed25519.PublicKeySize || *policy == "" || *manifest == "" || *approval == "" || fs.NArg() != 0 {
		return errors.New("verify requires -approver (64 hex), -policy, -manifest and -approval")
	}
	p, err := readBounded(*policy)
	if err != nil {
		return err
	}
	m, err := readBounded(*manifest)
	if err != nil {
		return err
	}
	raw, err := readBounded(*approval)
	if err != nil {
		return err
	}
	a, err := guardapproval.Check(raw, p, m, []ed25519.PublicKey{public})
	if err != nil {
		return errors.New("approval not accepted")
	}
	_, err = fmt.Fprintf(out, "approval valid sequence=%d policy=%s manifest=%s\n", a.Sequence, a.Policy, a.Manifest)
	return err
}
