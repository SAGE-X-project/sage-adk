// SPDX-License-Identifier: LGPL-3.0-or-later
// Fixed safe runtime fixture: ephemeral keys, loopback, measured native admission
// and only 2+3. Fixture local assurance/Registry are not deployment providers.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	channel "github.com/sage-x-project/sage-adk/core/guardchannel"
	"golang.org/x/sys/unix"
)

type fixtureLocalAssurance struct{ calls atomic.Int64 }

func (f *fixtureLocalAssurance) Check(ctx context.Context, snapshot *b.Snapshot) error {
	// Unit/integration-only provider. The real parent checks the sealed child;
	// this deliberately does not pretend to attest OS/private instruction pages.
	if ctx.Err() != nil || snapshot == nil {
		return b.ErrDenied
	}
	f.calls.Add(1)
	return nil
}

type checkedFixture struct {
	inner *channel.Measurement
	local *fixtureLocalAssurance
}

func (f checkedFixture) Check(ctx context.Context, snapshot *b.Snapshot) error {
	err := f.inner.Check(ctx, snapshot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fixture measurement denied; local checks", f.local.calls.Load())
	}
	return err
}
func main() {
	// Without an inherited channel, fd 4 may belong to Go's poller. Inspect
	// its kind without claiming ownership. Peer authentication still belongs
	// to OpenChild; this fixture preflight establishes no authority.
	domain, domainErr := unix.GetsockoptInt(4, unix.SOL_SOCKET, unix.SO_DOMAIN)
	kind, kindErr := unix.GetsockoptInt(4, unix.SOL_SOCKET, unix.SO_TYPE)
	if domainErr != nil || kindErr != nil || domain != unix.AF_UNIX || kind != unix.SOCK_SEQPACKET {
		fmt.Fprintln(os.Stderr, "measurement unavailable")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	local := &fixtureLocalAssurance{}
	measured, err := channel.OpenChild(ctx, os.NewFile(4, "protected-child"), local, 5*time.Second)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "measurement unavailable")
		os.Exit(1)
	}
	defer func() {
		ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = measured.Close(ctx)
	}()
	raw, err := os.ReadFile("/proc/self/exe")
	if err != nil || len(raw) > 64<<20 {
		fmt.Fprintln(os.Stderr, "image unavailable")
		os.Exit(1)
	}
	fmt.Println("measured-child-ready")
	input := bufio.NewScanner(os.Stdin)
	if !input.Scan() || input.Text() != "run" {
		return
	}
	testing.Main(regexp.MatchString, []testing.InternalTest{{Name: "MeasuredNativeCalculator", F: func(t *testing.T) {
		runNativeFixture(t, checkedFixture{measured, local}, raw)
		if local.calls.Load() < 3 {
			t.Fatal("measurement did not cover loading and admitted execution")
		}
		fmt.Println("verified-guarded-result=5")
	}}}, nil, nil)
}
