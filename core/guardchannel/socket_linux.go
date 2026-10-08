//go:build linux && (amd64 || arm64)

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardchannel

import (
	"context"
	"errors"
	"os"

	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	image "github.com/sage-x-project/sage-adk/core/guardimage"
	"golang.org/x/sys/unix"
)

type socket struct {
	file     *os.File
	peer     int32
	uid, gid uint32
}

func launch(ctx context.Context, c image.Config) (*image.Process, endpoint, error) {
	fds, e := unix.Socketpair(unix.AF_UNIX, unix.SOCK_SEQPACKET|unix.SOCK_CLOEXEC, 0)
	if e != nil {
		return nil, nil, b.ErrDenied
	}
	parent := os.NewFile(uintptr(fds[0]), "protected-supervisor")
	child := os.NewFile(uintptr(fds[1]), "protected-child")
	defer func() { _ = child.Close() }()
	owned := false
	defer func() {
		if !owned {
			_ = parent.Close()
		}
	}()
	for _, fd := range fds {
		if unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_PASSCRED, 1) != nil {
			return nil, nil, b.ErrDenied
		}
	}
	c.MeasurementChannel = child
	p, e := image.Start(ctx, c)
	if e != nil {
		return nil, nil, e
	}
	owned = true
	return p, &socket{file: parent, uid: uint32(os.Getuid()), gid: uint32(os.Getgid())}, nil
}

func child(file *os.File) (endpoint, error) {
	owned := false
	defer func() {
		if !owned {
			_ = file.Close()
		}
	}()
	if file.Fd() != 4 || os.Getppid() <= 1 {
		return nil, b.ErrDenied
	}
	fd := int(file.Fd())
	domain, e := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_DOMAIN)
	if e != nil || domain != unix.AF_UNIX {
		return nil, b.ErrDenied
	}
	kind, e := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_TYPE)
	if e != nil || kind != unix.SOCK_SEQPACKET {
		return nil, b.ErrDenied
	}
	if unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_PASSCRED, 1) != nil {
		return nil, b.ErrDenied
	}
	flags, e := unix.FcntlInt(file.Fd(), unix.F_GETFD, 0)
	if e != nil {
		return nil, b.ErrDenied
	}
	if _, e = unix.FcntlInt(file.Fd(), unix.F_SETFD, flags|unix.FD_CLOEXEC); e != nil {
		return nil, b.ErrDenied
	}
	flags, e = unix.FcntlInt(file.Fd(), unix.F_GETFD, 0)
	if e != nil || flags&unix.FD_CLOEXEC == 0 {
		return nil, b.ErrDenied
	}
	owned = true
	return &socket{file: file, peer: int32(os.Getppid()), uid: uint32(os.Getuid()), gid: uint32(os.Getgid())}, nil
}
func (s *socket) bindPeer(pid int) error {
	if s == nil || s.peer != 0 || pid <= 0 || int64(pid) > 2147483647 {
		return b.ErrDenied
	}
	s.peer = int32(pid)
	return nil
}
func (s *socket) wait(ctx context.Context, events int16) error {
	if s == nil || s.file == nil || !active(ctx) {
		return b.ErrDenied
	}
	fds := []unix.PollFd{{Fd: int32(s.file.Fd()), Events: events}}
	for active(ctx) {
		n, e := unix.Poll(fds, 10)
		if errors.Is(e, unix.EINTR) {
			continue
		}
		if e != nil {
			return b.ErrDenied
		}
		if n > 0 {
			if fds[0].Revents&(events|unix.POLLHUP) != 0 {
				return nil
			}
			return b.ErrDenied
		}
	}
	return b.ErrDenied
}
func (s *socket) send(ctx context.Context, r record) error {
	raw := encode(r)
	for active(ctx) {
		if s.wait(ctx, unix.POLLOUT) != nil {
			return b.ErrDenied
		}
		n, e := unix.SendmsgN(int(s.file.Fd()), raw, nil, nil, unix.MSG_DONTWAIT|unix.MSG_NOSIGNAL)
		if errors.Is(e, unix.EAGAIN) || errors.Is(e, unix.EINTR) {
			continue
		}
		if e != nil || n != len(raw) {
			return b.ErrDenied
		}
		return nil
	}
	return b.ErrDenied
}

// authenticate validates exactly one kernel-checked sender credential. Close any
// unsolicited received descriptors before refusal; never make them capabilities.
func (s *socket) authenticate(oob []byte) error {
	messages, e := unix.ParseSocketControlMessage(oob)
	if e != nil {
		return b.ErrDenied
	}
	good := len(messages) == 1
	credentials := 0
	for _, m := range messages {
		switch {
		case m.Header.Level == unix.SOL_SOCKET && m.Header.Type == unix.SCM_CREDENTIALS:
			cred, e := unix.ParseUnixCredentials(&m)
			credentials++
			good = good && e == nil && cred != nil && cred.Pid == s.peer && cred.Uid == s.uid && cred.Gid == s.gid
		case m.Header.Level == unix.SOL_SOCKET && m.Header.Type == unix.SCM_RIGHTS:
			rights, e := unix.ParseUnixRights(&m)
			if e == nil {
				for _, fd := range rights {
					_ = unix.Close(fd)
				}
			}
			good = false
		default:
			good = false
		}
	}
	if !good || credentials != 1 {
		return b.ErrDenied
	}
	return nil
}
func (s *socket) receive(ctx context.Context) (record, error) {
	raw := make([]byte, recordBytes)
	// Linux permits at most 253 descriptors in one rights message. Buffer and
	// close unsolicited handles without running or retaining their contents.
	oob := make([]byte, unix.CmsgSpace(unix.SizeofUcred)+unix.CmsgSpace(4*253))
	for active(ctx) {
		if s.wait(ctx, unix.POLLIN) != nil {
			return record{}, b.ErrDenied
		}
		n, on, flags, _, e := unix.Recvmsg(int(s.file.Fd()), raw, oob, unix.MSG_DONTWAIT|unix.MSG_CMSG_CLOEXEC)
		if errors.Is(e, unix.EAGAIN) || errors.Is(e, unix.EINTR) {
			continue
		}
		if e != nil {
			return record{}, b.ErrDenied
		}
		auth := s.authenticate(oob[:on])
		if auth != nil || n != recordBytes || flags&(unix.MSG_TRUNC|unix.MSG_CTRUNC) != 0 {
			return record{}, b.ErrDenied
		}
		return decode(raw)
	}
	return record{}, b.ErrDenied
}
func (s *socket) close() error {
	if s == nil || s.file == nil {
		return nil
	}
	e := s.file.Close()
	s.file = nil
	return e
}
