// SPDX-License-Identifier: LGPL-3.0-or-later
package guardservices

import (
	"context"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
	"sync"
	"time"
)

// SystemClock shares UTC and one process-local monotonic origin across Registry,
// HPKE, native host and Client services. Its zero value is invalid. Regression
// permanently refuses subsequent samples; never reset it to bypass an operation.
// Do not copy it. This observes the local OS clock, not external time attestation.
type SystemClock struct {
	mu                sync.Mutex
	read              func() (int64, int64)
	lastUTC, lastMono int64
	sampled, failed   bool
}

// NewSystemClock creates a process-owned local clock. Share this same instance
// with every service in the host; do not create a separate clock per connection.
func NewSystemClock() *SystemClock {
	origin := time.Now()
	return &SystemClock{read: func() (int64, int64) { now := time.Now(); return now.UnixMilli(), now.Sub(origin).Milliseconds() }}
}

// Sample returns UTC milliseconds and elapsed monotonic milliseconds together.
// Cancelled calls fail without resetting or poisoning an otherwise healthy clock.
func (c *SystemClock) Sample(ctx context.Context) (utc, mono int64, err error) {
	if c == nil || !active(ctx) {
		return 0, 0, ErrDenied
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	defer func() {
		if recover() != nil {
			c.failed = true
			utc, mono, err = 0, 0, ErrDenied
		}
	}()
	if c.failed || c.read == nil {
		return 0, 0, ErrDenied
	}
	utc, mono = c.read()
	if utc < 0 || utc > 9007199254740000 || mono < 0 || (c.sampled && (utc < c.lastUTC || mono < c.lastMono)) {
		c.failed = true
		return 0, 0, ErrDenied
	}
	if !active(ctx) {
		return 0, 0, ErrDenied
	}
	c.lastUTC, c.lastMono, c.sampled = utc, mono, true
	return utc, mono, nil
}

// Now supplies Registry/HPKE/native-host seconds and the same monotonic origin.
func (c *SystemClock) Now() (r.Stamp, error) {
	u, m, e := c.Sample(context.Background())
	if e != nil {
		return r.Stamp{}, e
	}
	return r.Stamp{Unix: u / 1000, MonoMS: m}, nil
}
