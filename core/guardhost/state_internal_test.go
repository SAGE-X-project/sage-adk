//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later
package guardhost

import (
	"context"
	"testing"
)

func TestAbsentPorts(t *testing.T) {
	var none *state
	var port any = none
	for name, c := range map[string]struct {
		v    any
		want bool
	}{
		"nil":           {nil, true},
		"typed-nil":     {port, true},
		"nil-func":      {(func())(nil), true},
		"value":         {struct{}{}, false},
		"pointer":       {&state{}, false},
		"non-nil-slice": {[]byte{}, false},
	} {
		if absent(c.v) != c.want {
			t.Fatalf("%s: absent = %v", name, !c.want)
		}
	}
}

// The caller never commits through the issuer's sender; delivery is owned by
// the native connection.
func TestCallerIssuerSenderRefusesCommit(t *testing.T) {
	if (noSender{}).Commit(context.Background(), "id", []byte("intent")) == nil {
		t.Fatal("sender committed")
	}
}
