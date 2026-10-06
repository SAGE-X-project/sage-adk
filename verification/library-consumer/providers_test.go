// SPDX-License-Identifier: LGPL-3.0-or-later
package consumer_test

import (
	"context"
	"sync"
	"testing"

	p "github.com/sage-x-project/sage-adk/core/guardservices"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

func TestPublicSharedGuardClock(t *testing.T) {
	clock := p.NewSystemClock()
	var registryClock r.Clock = clock
	var clientClock g.ClientClock = clock
	stamp, e := registryClock.Now()
	if e != nil {
		t.Fatal(e)
	}
	u, m, e := clientClock.Sample(context.Background())
	if e != nil || u/1000 < stamp.Unix || m < stamp.MonoMS {
		t.Fatal("inconsistent public clock", e)
	}
	var wg sync.WaitGroup
	for n := 0; n < 16; n++ {
		wg.Go(func() {
			if _, _, e := clientClock.Sample(context.Background()); e != nil {
				t.Error(e)
			}
		})
	}
	wg.Wait()
}
