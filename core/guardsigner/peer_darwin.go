//go:build darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner

import (
	"net"

	"golang.org/x/sys/unix"
)

// peerUID returns the kernel-reported effective account of the connected peer.
func peerUID(c *net.UnixConn) (uint32, error) {
	raw, err := c.SyscallConn()
	if err != nil {
		return 0, ErrDenied
	}
	var cred *unix.Xucred
	var credErr error
	if err = raw.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	}); err != nil || credErr != nil || cred == nil {
		return 0, ErrDenied
	}
	return cred.Uid, nil
}
