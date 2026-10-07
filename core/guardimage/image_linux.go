//go:build linux && (amd64 || arm64)

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardimage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

const seals = unix.F_SEAL_WRITE | unix.F_SEAL_GROW | unix.F_SEAL_SHRINK | unix.F_SEAL_SEAL | unix.F_SEAL_EXEC

type linuxProcess struct {
	image  *os.File
	pidfd  int
	pid    int
	digest string
	stat   unix.Stat_t
	done   chan struct{}
}

func start(ctx context.Context, c Config, raw []byte) (backend, error) {
	if !active(ctx) {
		return nil, ErrDenied
	}
	fd, err := unix.MemfdCreate("sage-approved-host", unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING|unix.MFD_EXEC)
	if err != nil {
		return nil, ErrUnsupported
	}
	file := os.NewFile(uintptr(fd), "sage-approved-host")
	owned := false
	defer func() {
		if !owned {
			_ = file.Close()
		}
	}()
	if _, err = file.Write(raw); err != nil {
		return nil, ErrDenied
	}
	if err = unix.Fchmod(fd, 0500); err != nil {
		return nil, ErrDenied
	}
	if _, err = unix.FcntlInt(uintptr(fd), unix.F_ADD_SEALS, seals); err != nil {
		return nil, ErrDenied
	}
	var stat unix.Stat_t
	if unix.Fstat(fd, &stat) != nil || stat.Size != int64(len(raw)) {
		return nil, ErrDenied
	}
	actual, err := unix.FcntlInt(uintptr(fd), unix.F_GET_SEALS, 0)
	if err != nil || actual&seals != seals {
		return nil, ErrDenied
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, ErrDenied
	}
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(file, int64(len(raw))+1))
	if err != nil || n != int64(len(raw)) || hex.EncodeToString(hash.Sum(nil)) != c.SHA256 {
		return nil, ErrDenied
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, ErrDenied
	}
	if !active(ctx) {
		return nil, ErrDenied
	}
	pidfd := -1
	cmd := exec.Command("/proc/self/fd/3")
	cmd.ExtraFiles = []*os.File{file}
	cmd.Env = []string{}
	if c.Stdin != nil {
		cmd.Stdin = c.Stdin
	}
	if c.Stdout != nil {
		cmd.Stdout = c.Stdout
	}
	if c.Stderr != nil {
		cmd.Stderr = c.Stderr
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{PidFD: &pidfd}
	if err = cmd.Start(); err != nil {
		if pidfd >= 0 {
			_ = unix.Close(pidfd)
		}
		return nil, ErrDenied
	}
	p := &linuxProcess{image: file, pidfd: pidfd, pid: cmd.Process.Pid, digest: c.SHA256, stat: stat, done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(p.done) }()
	if pidfd < 0 || !active(ctx) {
		_ = cmd.Process.Kill()
		<-p.done
		if pidfd >= 0 {
			_ = unix.Close(pidfd)
		}
		return nil, ErrDenied
	}
	owned = true
	return p, nil
}

func (p *linuxProcess) live() bool {
	if p.pidfd < 0 {
		return false
	}
	fds := []unix.PollFd{{Fd: int32(p.pidfd), Events: unix.POLLIN}}
	n, err := unix.Poll(fds, 0)
	return err == nil && n == 0 && fds[0].Revents == 0
}

func (p *linuxProcess) observe(ctx context.Context) (Observation, error) {
	if !active(ctx) || !p.live() || p.image == nil {
		return Observation{}, ErrDenied
	}
	fd, err := unix.Open("/proc/"+strconv.Itoa(p.pid)+"/exe", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return Observation{}, ErrDenied
	}
	file := os.NewFile(uintptr(fd), "running-host-image")
	defer func() { _ = file.Close() }()
	var stat unix.Stat_t
	actual, err := unix.FcntlInt(uintptr(fd), unix.F_GET_SEALS, 0)
	if err != nil || actual&seals != seals || unix.Fstat(fd, &stat) != nil || stat.Dev != p.stat.Dev || stat.Ino != p.stat.Ino || stat.Size != p.stat.Size {
		return Observation{}, ErrDenied
	}
	h := sha256.New()
	buffer := make([]byte, 64<<10)
	var total int64
	for {
		if !active(ctx) {
			return Observation{}, ErrDenied
		}
		n, e := file.Read(buffer)
		total += int64(n)
		if total > p.stat.Size {
			return Observation{}, ErrDenied
		}
		if n > 0 {
			_, _ = h.Write(buffer[:n])
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return Observation{}, ErrDenied
		}
	}
	if total != p.stat.Size || hex.EncodeToString(h.Sum(nil)) != p.digest {
		return Observation{}, ErrDenied
	}
	maps, err := os.Open("/proc/" + strconv.Itoa(p.pid) + "/maps")
	if err != nil {
		return Observation{}, ErrDenied
	}
	data, err := io.ReadAll(io.LimitReader(maps, (1<<20)+1))
	_ = maps.Close()
	if err != nil || len(data) > 1<<20 || !validMaps(data, uint64(unix.Major(p.stat.Dev)), uint64(unix.Minor(p.stat.Dev)), p.stat.Ino) || !active(ctx) || !p.live() {
		return Observation{}, ErrDenied
	}
	return Observation{SHA256: p.digest, Architecture: runtime.GOARCH, PID: p.pid}, nil
}

func (p *linuxProcess) close(ctx context.Context) error {
	if !active(ctx) {
		return ErrDenied
	}
	if p.live() {
		if unix.PidfdSendSignal(p.pidfd, unix.SIGKILL, nil, 0) != nil {
			return ErrDenied
		}
	}
	select {
	case <-p.done:
	case <-ctx.Done():
		return ErrDenied
	}
	if p.image != nil {
		_ = p.image.Close()
		p.image = nil
	}
	if p.pidfd >= 0 {
		_ = unix.Close(p.pidfd)
		p.pidfd = -1
	}
	return nil
}
