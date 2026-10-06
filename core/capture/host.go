// SPDX-License-Identifier: LGPL-3.0-or-later
package capture

import (
	"context"
	"reflect"

	"github.com/google/uuid"
	adkerrors "github.com/sage-x-project/sage-adk/pkg/errors"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// ErrCapture indicates unavailable or inconsistent original input. Processing
// must stop; no reconstructed message may substitute for the original bytes.
var ErrCapture = &adkerrors.Error{Category: adkerrors.CategorySecurity, Code: "ORIGINAL_CAPTURE_INVALID", Message: "original request capture unavailable or inconsistent"}

// Host owns the trusted store. Keep this capability outside models and plugins.
type Host struct{ store Store }

// NewHost binds a host to a required protected store without performing I/O.
func NewHost(store Store) (*Host, error) {
	if absent(store) {
		return nil, ErrCapture
	}
	return &Host{store: store}, nil
}
func absent(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
func active(ctx context.Context) bool { return ctx != nil && ctx.Err() == nil }
func clone(b [][]byte) [][]byte {
	c := make([][]byte, len(b))
	for i := range b {
		c[i] = append([]byte(nil), b[i]...)
	}
	return c
}

// Capture assigns a fresh request ID, persists a defensive copy before any
// transformation, and checks readback. A failed attempt is never overwritten or
// automatically retried. The caller must not concurrently mutate supplied slices.
func (h *Host) Capture(ctx context.Context, inputs [][]byte) (request *Request, err error) {
	defer func() {
		if recover() != nil {
			request = nil
			err = ErrCapture
		}
	}()
	if h == nil || absent(h.store) || !active(ctx) {
		return nil, ErrCapture
	}
	// Validate bounds before allocating another copy.
	if _, e := g.OriginalCommitment(inputs); e != nil {
		return nil, ErrCapture
	}
	owned := clone(inputs)
	digest, e := g.OriginalCommitment(owned)
	if e != nil {
		return nil, ErrCapture
	}
	id, e := uuid.NewRandom()
	if e != nil {
		return nil, ErrCapture
	}
	if e = h.store.Create(ctx, id.String(), clone(owned)); e != nil || !active(ctx) {
		return nil, ErrCapture
	}
	r := &Request{host: h, id: id.String(), digest: digest}
	if _, e = r.Inputs(ctx); e != nil {
		return nil, ErrCapture
	}
	return r, nil
}

// Restore rebinds retained input after a host restart. Both ID and digest must
// come from the host's protected operation checkpoint, never model arguments or
// an unverified wire envelope. It grants no new approval and does not replace
// the core's durable issuance fence or Client journal.
func (h *Host) Restore(ctx context.Context, id, digest string) (*Request, error) {
	if h == nil || absent(h.store) || !active(ctx) {
		return nil, ErrCapture
	}
	if _, err := g.NewRootCapture(nil, id); err != nil {
		return nil, ErrCapture
	}
	r := &Request{host: h, id: id, digest: digest}
	_, err := r.Inputs(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil
}
