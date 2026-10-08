//go:build linux && (amd64 || arm64)

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardchannel

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

var runtimeImage struct {
	once sync.Once
	raw  []byte
	err  error
}

func childImage(t *testing.T) []byte {
	t.Helper()
	runtimeImage.once.Do(func() {
		dir, e := os.MkdirTemp("", "sage-channel-fixture-")
		if e != nil {
			runtimeImage.err = e
			return
		}
		defer func() { _ = os.RemoveAll(dir) }()
		path := filepath.Join(dir, "child")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-trimpath", "-o", path, "./testdata/native")
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
		if output, e := cmd.CombinedOutput(); e != nil {
			runtimeImage.err = e
			t.Log(string(output))
			return
		}
		runtimeImage.raw, runtimeImage.err = os.ReadFile(path)
	})
	if runtimeImage.err != nil {
		t.Fatal(runtimeImage.err)
	}
	return runtimeImage.raw
}
func TestMeasuredNativeCalculatorRuntime(t *testing.T) {
	for _, limit := range []uint64{1000, 1} {
		t.Run(map[bool]string{true: "admitted", false: "bounded-exhaustion"}[limit > 1], func(t *testing.T) {
			raw := childImage(t)
			c := fixtureConfig(raw)
			c.Timeout = 5 * time.Second
			c.MaxChecks = limit
			input, send, e := os.Pipe()
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = input.Close(); _ = send.Close() }()
			read, output, e := os.Pipe()
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = read.Close(); _ = output.Close() }()
			c.Image.Stdin = input
			c.Image.Stdout = output
			c.Image.Stderr = output
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			s, e := Start(ctx, c)
			if e != nil {
				t.Fatal("required native Linux bridge unavailable", e)
			}
			defer func() {
				cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
				defer stop()
				if e := s.Close(cleanup); e != nil {
					t.Error(e)
				}
			}()
			_ = input.Close()
			_ = output.Close()
			if e := read.SetReadDeadline(time.Now().Add(25 * time.Second)); e != nil {
				t.Fatal(e)
			}
			reader := bufio.NewReader(read)
			line, e := reader.ReadString('\n')
			if e != nil || line != "measured-child-ready\n" {
				t.Fatal("bootstrap", line, e)
			}
			if _, e := send.WriteString("run\n"); e != nil {
				t.Fatal(e)
			}
			result, e := io.ReadAll(reader)
			if e != nil {
				t.Fatal(e)
			}
			if limit > 1 {
				if !bytes.Contains(result, []byte("verified-guarded-result=5\n")) || !bytes.Contains(result, []byte("PASS\n")) || bytes.Contains(result, []byte("FAIL")) {
					t.Fatalf("native measured exchange (observations=%d, observer stage=%d): %s", s.observations.Load(), s.failure.Load(), result)
				}
				if s.observations.Load() < 3 {
					t.Fatal("missing fresh observations at load/admission/effect")
				}
			} else {
				if bytes.Contains(result, []byte("verified-guarded-result")) || !bytes.Contains(result, []byte("FAIL")) {
					t.Fatalf("exhausted channel admitted native arithmetic: %s", result)
				}
				if s.observations.Load() != 1 {
					t.Fatal("unbounded check budget")
				}
			}
			select {
			case <-s.done:
			case <-ctx.Done():
				t.Fatal("channel worker did not retire on child exit")
			}
			if !s.retired.Load() {
				t.Fatal("exited child session retained")
			}
			if e := s.Close(ctx); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestPrivateChannelCleanupBeforeAnyEffectRuntime(t *testing.T) {
	c := fixtureConfig(childImage(t))
	c.Timeout = 5 * time.Second
	input, send, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = input.Close(); _ = send.Close() }()
	read, out, e := os.Pipe()
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = read.Close(); _ = out.Close() }()
	c.Image.Stdin = input
	c.Image.Stdout = out
	c.Image.Stderr = out
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Second)
	defer stop()
	s, e := Start(ctx, c)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if e := s.Close(cleanup); e != nil {
			t.Error(e)
		}
	}()
	_ = input.Close()
	_ = out.Close()
	if e := read.SetReadDeadline(time.Now().Add(5 * time.Second)); e != nil {
		t.Fatal(e)
	}
	reader := bufio.NewReader(read)
	line, e := reader.ReadString('\n')
	if e != nil || line != "measured-child-ready\n" {
		t.Fatal("bootstrap", line, e)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if s.Close(cancelled) == nil || s.closed || !s.retired.Load() {
		t.Fatal("cancelled cleanup lost ownership")
	}
	// The fixed fixture has not received its only run command: no effect to kill.
	if e := s.Close(ctx); e != nil {
		t.Fatal(e)
	}
	remaining, e := io.ReadAll(reader)
	if e != nil || bytes.Contains(remaining, []byte("verified-guarded-result")) {
		t.Fatal("unexpected native effect", e)
	}
	if s.observations.Load() != 0 || !s.closed {
		t.Fatal("shutdown before admission")
	}
}
func TestMissingPrivateChannelRuntime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixed-child")
	if e := os.WriteFile(path, childImage(t), 0500); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path)
	output, e := cmd.CombinedOutput()
	if e == nil || !bytes.Contains(output, []byte("measurement unavailable")) || bytes.Contains(output, []byte("verified-guarded-result")) {
		t.Fatalf("missing channel fallback: %s %v", output, e)
	}
}
func TestKernelCredentialAndAncillaryRefusalUnits(t *testing.T) {
	s := &socket{peer: 123, uid: 456, gid: 789}
	correct := unix.UnixCredentials(&unix.Ucred{Pid: 123, Uid: 456, Gid: 789})
	if e := s.authenticate(correct); e != nil {
		t.Fatal(e)
	}
	cases := [][]byte{nil, []byte{1}, unix.UnixCredentials(&unix.Ucred{Pid: 124, Uid: 456, Gid: 789}), unix.UnixCredentials(&unix.Ucred{Pid: 123, Uid: 457, Gid: 789}), unix.UnixCredentials(&unix.Ucred{Pid: 123, Uid: 456, Gid: 790}), append(append([]byte(nil), correct...), correct...)}
	for _, oob := range cases {
		if e := s.authenticate(oob); e == nil {
			t.Fatal("accepted wrong/missing/duplicate credential")
		}
	}
	// Unit-only ancillary parser test owns this descriptor and closes it on refusal;
	// no adversarial sendmsg runtime or impersonation program is constructed.
	fd, e := unix.Open("/dev/null", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if e != nil {
		t.Fatal(e)
	}
	if s.authenticate(append(append([]byte(nil), correct...), unix.UnixRights(fd)...)) == nil {
		t.Fatal("accepted unsolicited capability")
	}
	if _, e := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); e == nil {
		_ = unix.Close(fd)
		t.Fatal("unsolicited descriptor leaked")
	}
}
