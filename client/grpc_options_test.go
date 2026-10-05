// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"testing"
	"time"
)

func TestGRPCSharedOptions(t *testing.T) {
	c, err := NewGRPCClient("127.0.0.1:1", WithTimeout(2*time.Second), WithRetry(0, time.Millisecond, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.config.timeout != 2*time.Second || c.config.maxRetries != 0 {
		t.Fatal("gRPC options were not applied")
	}
	var _ MessageStream = (*grpcMessageStream)(nil)
}
