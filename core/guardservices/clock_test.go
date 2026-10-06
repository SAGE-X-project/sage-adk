// SPDX-License-Identifier: LGPL-3.0-or-later
package guardservices

import (
	"context"
	"testing"
)

func TestClockSharesOneSampleAndRejectsRegression(t *testing.T) {
	for _, mode := range []string{"wall", "mono", "negative-wall", "negative-mono", "range"} {
		t.Run(mode, func(t *testing.T) {
			utc, mono := int64(123456), int64(20)
			c := &SystemClock{read: func() (int64, int64) { return utc, mono }}
			stamp, e := c.Now()
			if e != nil || stamp.Unix != 123 || stamp.MonoMS != 20 {
				t.Fatal(stamp, e)
			}
			u, m, e := c.Sample(context.Background())
			if e != nil || u != utc || m != mono {
				t.Fatal(u, m, e)
			}
			switch mode {
			case "wall":
				utc--
			case "mono":
				mono--
			case "negative-wall":
				utc = -1
			case "negative-mono":
				mono = -1
			case "range":
				utc = 9007199254740001
			}
			if _, _, e = c.Sample(context.Background()); e == nil {
				t.Fatal("invalid sample accepted")
			}
			utc, mono = 123457, 21
			if _, e = c.Now(); e == nil {
				t.Fatal("clock revived after regression")
			}
		})
	}
}
func TestClockCancellationAndInvalidInstances(t *testing.T) {
	c := NewSystemClock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, e := c.Sample(ctx); e == nil {
		t.Fatal("cancelled")
	}
	if _, _, e := c.Sample(nil); e == nil {
		t.Fatal("nil context")
	}
	if _, e := c.Now(); e != nil {
		t.Fatal(e)
	}
	var missing *SystemClock
	for _, x := range []*SystemClock{missing, {}} {
		if _, e := x.Now(); e == nil {
			t.Fatal("invalid clock")
		}
	}
	c = &SystemClock{read: func() (int64, int64) { panic("inert clock failure") }}
	if _, e := c.Now(); e == nil {
		t.Fatal("panic")
	}
}
