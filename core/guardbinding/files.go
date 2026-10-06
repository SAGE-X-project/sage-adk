//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardbinding

import (
	"golang.org/x/sys/unix"
	"os"
)

func openArtifact(root *os.Root, path string) (*os.File, error) {
	// No-follow refuses the final symlink. Non-blocking open plus a subsequent
	// fstat refuses a non-regular replacement without waiting on a FIFO/device.
	// The root confines reads; administration must protect and serialize all
	// intermediate-directory mutations. This is not an OS isolation service.
	return root.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
}
