//go:build linux && (amd64 || arm64)

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardimage

import (
	"bufio"
	"context"
	"errors"
	"golang.org/x/sys/unix"
	"os"
	"testing"
	"time"
)

func TestSealedExecutableRuntime(t *testing.T) {
	in, send, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = in.Close() }()
	defer func() { _ = send.Close() }()
	receive, out, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = receive.Close() }()
	defer func() { _ = out.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := config()
	c.Stdin = in
	c.Stdout = out
	c.Stderr = out
	process, err := Start(ctx, c)
	// Linux CI must exercise the real backend; unsupported kernel policy is a
	// refusal/failure, never a skipped or synthetic successful runtime test.
	if err != nil {
		t.Fatal("required Linux runtime unavailable", err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if e := process.Close(cleanup); e != nil {
			t.Error(e)
		}
	}()
	_ = in.Close()
	_ = out.Close()
	read := bufio.NewReader(receive)
	line := func() string {
		t.Helper()
		if e := receive.SetReadDeadline(time.Now().Add(5 * time.Second)); e != nil {
			t.Fatal(e)
		}
		s, e := read.ReadString('\n')
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	if line() != "ready\n" {
		t.Fatal("host not ready")
	}
	first, err := process.Observe(ctx)
	if err != nil || first.SHA256 != c.SHA256 || first.PID <= 0 {
		t.Fatal("sealed runtime identity", err)
	}
	if err = process.CheckSnapshot(ctx, snapshot(t, true, true, testImage), "host"); err != nil {
		t.Fatal("approved runtime snapshot", err)
	}
	if _, err = send.WriteString("calculate\n"); err != nil {
		t.Fatal(err)
	}
	if line() != "{\"success\":true,\"output\":5}\n" {
		t.Fatal("compiled calculator result")
	}
	second, err := process.Observe(ctx)
	if err != nil || first != second {
		t.Fatal("runtime identity changed", err)
	}
	if err = send.Close(); err != nil {
		t.Fatal(err)
	}
	driver := process.driver.(*linuxProcess)
	select {
	case <-driver.done:
	case <-ctx.Done():
		t.Fatal("host did not exit")
	}
	if _, err = process.Observe(ctx); !errors.Is(err, ErrDenied) {
		t.Fatal("observed exited process")
	}
	if err = process.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = process.Observe(ctx); !errors.Is(err, ErrDenied) {
		t.Fatal("observed retired process")
	}
}

func TestSealedExecutableCleanupRetryRuntime(t *testing.T) {
	in, send, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = in.Close(); _ = send.Close() }()
	receive, out, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = receive.Close(); _ = out.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := config()
	c.Stdin = in
	c.Stdout = out
	c.Stderr = out
	process, err := Start(ctx, c)
	if err != nil {
		t.Fatal("required Linux runtime unavailable", err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if e := process.Close(cleanup); e != nil {
			t.Error(e)
		}
	}()
	_ = in.Close()
	_ = out.Close()
	if e := receive.SetReadDeadline(time.Now().Add(5 * time.Second)); e != nil {
		t.Fatal(e)
	}
	if s, e := bufio.NewReader(receive).ReadString('\n'); e != nil || s != "ready\n" {
		t.Fatal("host not ready", e)
	}
	if _, e := process.Observe(ctx); e != nil {
		t.Fatal(e)
	}
	driver := process.driver.(*linuxProcess)
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if process.Close(cancelled) == nil || process.driver == nil || driver.image == nil || driver.pidfd < 0 {
		t.Fatal("cancelled cleanup lost resources")
	}
	if _, e := process.Observe(ctx); !errors.Is(e, ErrDenied) {
		t.Fatal("retired process observed")
	}
	// Only the owned harmless child is terminated; no effect has been requested.
	if e := process.Close(ctx); e != nil {
		t.Fatal(e)
	}
	if driver.image != nil || driver.pidfd != -1 || process.driver != nil {
		t.Fatal("handles retained after successful cleanup")
	}
}

// Unit-only interrupted-syscall schedules: no signals or external processes are
// attacked. A retried poll must still reject exit and retain a fixed retry bound.
func TestPidfdSignalInterruptionUnits(t *testing.T) {
	for _, scenario := range []string{"resume-live", "resume-exited", "persistent-interruption", "other-error", "negative-descriptor"} {
		t.Run(scenario, func(t *testing.T) {
			calls := 0
			fd := 17
			if scenario == "negative-descriptor" {
				fd = -1
			}
			live := pidfdLive(fd, func(fds []unix.PollFd, timeout int) (int, error) {
				calls++
				if len(fds) != 1 || fds[0].Fd != 17 || timeout != 0 {
					t.Fatal("substituted descriptor or blocking poll")
				}
				if scenario == "other-error" {
					return 0, unix.EBADF
				}
				if calls == 1 || scenario == "persistent-interruption" {
					return 0, unix.EINTR
				}
				if scenario == "resume-exited" {
					fds[0].Revents = unix.POLLIN
					return 1, nil
				}
				return 0, nil
			})
			if live != (scenario == "resume-live") {
				t.Fatal("incorrect lifetime after interruption")
			}
			expected := 2
			if scenario == "persistent-interruption" {
				expected = 3
			}
			if scenario == "other-error" {
				expected = 1
			}
			if scenario == "negative-descriptor" {
				expected = 0
			}
			if calls != expected {
				t.Fatal("unbounded or incorrect retry count", calls)
			}
		})
	}
}
