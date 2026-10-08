//go:build !linux && !darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardsigner

import "net"

// peerUID refuses platforms without a supported peer credential check.
func peerUID(*net.UnixConn) (uint32, error) { return 0, ErrDenied }
