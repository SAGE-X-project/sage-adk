// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import (
	"context"
	"testing"
)

type boundaryStore struct {
	fail   bool
	inputs [][]byte
}

func (s *boundaryStore) Create(_ context.Context, _ string, b [][]byte) error {
	s.inputs = clone(b)
	return nil
}
func (s *boundaryStore) Load(context.Context, string) ([][]byte, error) {
	if s.fail {
		return nil, ErrCapture
	}
	return clone(s.inputs), nil
}

type boundaryPolicy struct {
	calls  int
	digest string
}

func (p *boundaryPolicy) Bindings(context.Context, string, string) (string, []byte, []byte, error) {
	p.calls++
	return p.digest, nil, nil, nil
}
func (p *boundaryPolicy) Authorize(context.Context, string, string, []byte) error {
	p.calls++
	return nil
}
func (p *boundaryPolicy) ApproveIntent(context.Context, []byte) error { p.calls++; return nil }

type boundarySigner struct{ calls int }

func (s *boundarySigner) Sign(context.Context, string, []byte) ([]byte, error) {
	s.calls++
	return nil, nil
}

// Each callback is a separately callable core port. Missing original custody
// must prevent delegation even when an earlier check had succeeded.
func TestEachPortRefusesMissingOriginal(t *testing.T) {
	ctx := context.Background()
	store := &boundaryStore{}
	host, _ := NewHost(store)
	r, e := host.Capture(ctx, [][]byte{[]byte("inert")})
	if e != nil {
		t.Fatal(e)
	}
	delegate := &boundaryPolicy{digest: r.Digest()}
	p := &boundPolicy{request: r, delegate: delegate}
	approve := &boundIssuancePolicy{boundPolicy: *p, delegate: delegate}
	key := &boundarySigner{}
	signer := &boundSigner{request: r, delegate: key}
	if _, _, _, e = p.Bindings(ctx, "fixture", "different-request"); e == nil || delegate.calls != 0 {
		t.Fatal("wrong request reached policy")
	}
	store.fail = true
	if e = p.Authorize(ctx, "fixture", "read", nil); e == nil {
		t.Fatal("authorization with missing original")
	}
	if e = approve.ApproveIntent(ctx, nil); e == nil {
		t.Fatal("approval with missing original")
	}
	if _, e = signer.Sign(ctx, "fixture", nil); e == nil {
		t.Fatal("signing with missing original")
	}
	if delegate.calls != 0 || key.calls != 0 {
		t.Fatal("delegated after custody failure")
	}
}
