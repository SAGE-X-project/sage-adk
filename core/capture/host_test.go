// SPDX-License-Identifier: LGPL-3.0-or-later
package capture_test

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage-adk/core/capture"
)

type memoryStore struct {
	data                                     map[string][][]byte
	failCreate, failLoad, panicLoad, corrupt bool
}

func (s *memoryStore) Create(_ context.Context, id string, b [][]byte) error {
	if s.failCreate {
		return errors.New("unavailable")
	}
	if s.data == nil {
		s.data = map[string][][]byte{}
	}
	s.data[id] = copyInputs(b)
	return nil
}
func (s *memoryStore) Load(_ context.Context, id string) ([][]byte, error) {
	if s.panicLoad {
		panic("provider failed")
	}
	if s.failLoad {
		return nil, errors.New("unavailable")
	}
	if s.corrupt {
		return [][]byte{[]byte("different")}, nil
	}
	return copyInputs(s.data[id]), nil
}
func copyInputs(b [][]byte) [][]byte {
	c := make([][]byte, len(b))
	for i := range b {
		c[i] = append([]byte(nil), b[i]...)
	}
	return c
}

// Independent framing calculation: no core commitment helper is used here.
func commitment(b [][]byte) string {
	h := sha256.New()
	h.Write([]byte("sage-original|0.10.0\x00"))
	var n [8]byte
	binary.BigEndian.PutUint32(n[:4], uint32(len(b)))
	h.Write(n[:4])
	for _, v := range b {
		binary.BigEndian.PutUint64(n[:], uint64(len(v)))
		h.Write(n[:])
		h.Write(v)
	}
	return hex.EncodeToString(h.Sum(nil))
}
func TestExactCapture(t *testing.T) {
	ctx := context.Background()
	s := &memoryStore{}
	h, err := capture.NewHost(s)
	if err != nil {
		t.Fatal(err)
	}
	inputs := [][]byte{[]byte(" 요청 \n"), []byte("e\u0301"), {}, []byte("é")}
	expected := copyInputs(inputs)
	r, err := h.Capture(ctx, inputs)
	if err != nil {
		t.Fatal(err)
	}
	id, err := uuid.Parse(r.ID())
	if err != nil || id.Version() != 4 || id.String() != r.ID() {
		t.Fatal("fresh canonical UUIDv4 required")
	}
	if r.Digest() != commitment(expected) {
		t.Fatal("ordered exact-byte commitment differs")
	}
	inputs[0][0] = 'x'
	got, err := r.Inputs(ctx)
	if err != nil || !reflect.DeepEqual(got, expected) {
		t.Fatalf("capture alias: %v", err)
	}
	got[0][0] = 'y'
	again, err := r.Inputs(ctx)
	if err != nil || !reflect.DeepEqual(again, expected) {
		t.Fatal("load alias")
	}
	other, err := h.Capture(ctx, expected)
	if err != nil || other.ID() == r.ID() {
		t.Fatal("request IDs reused")
	}
	reordered := copyInputs(expected)
	reordered[1], reordered[3] = reordered[3], reordered[1]
	next, err := h.Capture(ctx, reordered)
	if err != nil || next.Digest() == r.Digest() {
		t.Fatal("order lost")
	}
}
func TestCaptureFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		s      *memoryStore
		inputs [][]byte
	}{
		{"create", &memoryStore{failCreate: true}, nil}, {"load", &memoryStore{failLoad: true}, nil}, {"panic", &memoryStore{panicLoad: true}, nil}, {"readback", &memoryStore{corrupt: true}, nil},
		{"utf8", &memoryStore{}, [][]byte{{255}}}, {"count", &memoryStore{}, make([][]byte, 1025)}, {"size", &memoryStore{}, [][]byte{make([]byte, (1<<20)+1)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := capture.NewHost(tc.s)
			if r, e := h.Capture(context.Background(), tc.inputs); e == nil || r != nil {
				t.Fatal("accepted invalid capture")
			}
		})
	}
	if _, err := capture.NewHost(nil); err == nil {
		t.Fatal("nil store")
	}
	var typed *memoryStore
	if _, err := capture.NewHost(typed); err == nil {
		t.Fatal("typed nil store")
	}
	h, _ := capture.NewHost(&memoryStore{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.Capture(ctx, nil); err == nil {
		t.Fatal("cancelled capture")
	}
	if _, err := h.Capture(nil, nil); err == nil {
		t.Fatal("nil context")
	}
	var zero capture.Request
	if _, err := zero.Inputs(context.Background()); err == nil {
		t.Fatal("zero capability")
	}
}
func TestChangedOrUnavailableOriginal(t *testing.T) {
	s := &memoryStore{}
	h, _ := capture.NewHost(s)
	r, err := h.Capture(context.Background(), [][]byte{[]byte("original")})
	if err != nil {
		t.Fatal(err)
	}
	s.corrupt = true
	if _, err := r.Inputs(context.Background()); err == nil {
		t.Fatal("changed original accepted")
	}
	s.corrupt = false
	s.failLoad = true
	if _, err := r.Inputs(context.Background()); err == nil {
		t.Fatal("missing original accepted")
	}
}

func TestRestoreRequiresProtectedCheckpoint(t *testing.T) {
	ctx := context.Background()
	s := &memoryStore{}
	h, _ := capture.NewHost(s)
	r, e := h.Capture(ctx, [][]byte{[]byte("original")})
	if e != nil {
		t.Fatal(e)
	}
	restored, e := h.Restore(ctx, r.ID(), r.Digest())
	if e != nil || restored.ID() != r.ID() || restored.Digest() != r.Digest() {
		t.Fatal("checkpoint restore")
	}
	if _, e = h.Restore(ctx, r.ID(), commitment([][]byte{[]byte("different")})); e == nil {
		t.Fatal("unbound checkpoint")
	}
	s.data["invalid"] = [][]byte{[]byte("original")}
	if _, e = h.Restore(ctx, "invalid", r.Digest()); e == nil {
		t.Fatal("invalid checkpoint ID")
	}
	if _, e = h.Restore(nil, r.ID(), r.Digest()); e == nil {
		t.Fatal("nil context")
	}
	var missing *capture.Host
	if _, e = missing.Restore(ctx, r.ID(), r.Digest()); e == nil {
		t.Fatal("nil host")
	}
	var request *capture.Request
	if request.ID() != "" || request.Digest() != "" {
		t.Fatal("nil request metadata")
	}
	if _, e = missing.Capture(ctx, nil); e == nil {
		t.Fatal("nil host capture")
	}
	s.data[r.ID()] = [][]byte{{255}}
	if _, e = r.Inputs(ctx); e == nil {
		t.Fatal("invalid retained UTF8")
	}
}

type valueStore struct{ *memoryStore }

func TestValueStoreAndExactLimits(t *testing.T) {
	h, e := capture.NewHost(valueStore{&memoryStore{}})
	if e != nil {
		t.Fatal(e)
	}
	b := make([][]byte, 1024)
	b[0] = make([]byte, 1<<20)
	r, e := h.Capture(context.Background(), b)
	if e != nil || r.Digest() != commitment(b) {
		t.Fatal("exact permitted bounds refused")
	}
}
