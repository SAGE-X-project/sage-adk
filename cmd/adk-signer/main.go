//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later

// Command adk-signer holds one Ed25519 signing key in its own process and
// serves signatures to allowlisted local host accounts over a Unix socket.
//
//	adk-signer keygen -key PATH
//	adk-signer kemgen -key PATH
//	adk-signer serve -key PATH [-kem-key PATH] -socket PATH -allow-uid UID[,UID] -roles ROLE[,ROLE]
//
// Run it under an account separate from the protected host and from any model
// or plugin process. Deployment owns account, directory and file protection.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sage-x-project/sage-adk/core/guardsigner"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "adk-signer:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: adk-signer keygen|serve [flags]")
	}
	switch args[0] {
	case "keygen":
		return keygen(args[1:], out)
	case "kemgen":
		return kemgen(args[1:], out)
	case "serve":
		return serve(args[1:], out)
	}
	return errors.New("unknown command " + strconv.Quote(args[0]))
}

func keygen(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	key := fs.String("key", "", "new seed file path (must not exist)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" || fs.NArg() != 0 {
		return errors.New("keygen requires -key and no arguments")
	}
	public, err := guardsigner.GenerateKeyFile(*key)
	if err != nil {
		return errors.New("key file not created")
	}
	_, err = fmt.Fprintln(out, hex.EncodeToString(public))
	return err
}

func kemgen(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("kemgen", flag.ContinueOnError)
	key := fs.String("key", "", "new X25519 key file path (must not exist)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" || fs.NArg() != 0 {
		return errors.New("kemgen requires -key and no arguments")
	}
	public, err := guardsigner.GenerateKEMFile(*key)
	if err != nil {
		return errors.New("key file not created")
	}
	_, err = fmt.Fprintln(out, hex.EncodeToString(public))
	return err
}

func serve(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	key := fs.String("key", "", "seed file path")
	kemKey := fs.String("kem-key", "", "X25519 key file path for the kem role")
	socket := fs.String("socket", "", "absolute Unix socket path (must not exist)")
	uids := fs.String("allow-uid", "", "comma-separated host account IDs")
	roles := fs.String("roles", "", "comma-separated roles: intent, result, transport, kem")
	timeout := fs.Duration("timeout", 2*time.Second, "per-request deadline")
	conns := fs.Int("max-connections", 16, "concurrent request bound")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *key == "" || *socket == "" || *uids == "" || *roles == "" || fs.NArg() != 0 {
		return errors.New("serve requires -key, -socket, -allow-uid and -roles")
	}
	allowed, err := parseUIDs(*uids)
	if err != nil {
		return err
	}
	private, err := guardsigner.LoadKeyFile(*key)
	if err != nil {
		return errors.New("key file unavailable or not private")
	}
	var kem []byte
	if *kemKey != "" {
		if kem, err = guardsigner.LoadKEMFile(*kemKey); err != nil {
			return errors.New("kem key file unavailable or not private")
		}
	}
	server, err := guardsigner.NewServer(guardsigner.Config{Key: private, KEM: kem, Roles: strings.Split(*roles, ","), AllowedUIDs: allowed, Timeout: *timeout, MaxConnections: *conns})
	for i := range private {
		private[i] = 0
	}
	for i := range kem {
		kem[i] = 0
	}
	if err != nil {
		return errors.New("invalid signer configuration")
	}
	defer server.Close()
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: *socket, Net: "unix"})
	if err != nil {
		return errors.New("socket unavailable")
	}
	// Account access is enforced by the peer check; the mode lets the socket
	// directory's group decide who may connect at all.
	if err = os.Chmod(*socket, 0660); err != nil {
		_ = listener.Close()
		return errors.New("socket mode not set")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if _, err = fmt.Fprintln(out, "adk-signer ready", hex.EncodeToString(server.PublicKey())); err != nil {
		_ = listener.Close()
		return err
	}
	if err = server.Serve(ctx, listener); err != nil {
		return errors.New("signer stopped")
	}
	return nil
}

func parseUIDs(raw string) ([]uint32, error) {
	var out []uint32
	for _, s := range strings.Split(raw, ",") {
		v, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return nil, errors.New("invalid -allow-uid")
		}
		out = append(out, uint32(v))
	}
	return out, nil
}
