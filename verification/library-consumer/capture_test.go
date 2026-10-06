// SPDX-License-Identifier: LGPL-3.0-or-later
package consumer_test

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/capture"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestPublicDurableOriginalCapture(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "originals")
	store, err := capture.OpenFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	host, err := capture.NewHost(store)
	if err != nil {
		t.Fatal(err)
	}
	inputs := [][]byte{[]byte(" 고정 요청\n"), []byte("e\u0301")}
	handled := 0
	a, err := agent.NewAgent("capture-consumer").OnMessage(func(_ context.Context, m agent.MessageContext) error { handled++; return m.Reply("inert response") }).Build()
	if err != nil {
		t.Fatal(err)
	}
	request, response, err := agent.ProcessOriginal(ctx, a, host, inputs, func(_ context.Context, retained [][]byte) (*types.Message, error) {
		if string(retained[0]) != string(inputs[0]) {
			t.Fatal("original bytes unavailable")
		}
		return types.NewMessage(types.MessageRoleUser, []types.Part{types.NewTextPart("decoded fixed request")}), nil
	})
	if err != nil || request == nil || response == nil || handled != 1 {
		t.Fatalf("capture processing: %v", err)
	}
	h := sha256.New()
	h.Write([]byte("sage-original|0.10.0\x00"))
	var n [8]byte
	binary.BigEndian.PutUint32(n[:4], 2)
	h.Write(n[:4])
	for _, b := range inputs {
		binary.BigEndian.PutUint64(n[:], uint64(len(b)))
		h.Write(n[:])
		h.Write(b)
	}
	digest := hex.EncodeToString(h.Sum(nil))
	if request.Digest() != digest {
		t.Fatal("independent commitment mismatch")
	}
	id := request.ID()
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := capture.OpenFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	host, err = capture.NewHost(reopened)
	if err != nil {
		t.Fatal(err)
	}
	// These values stand for a protected host checkpoint in this local fixture.
	restored, err := host.Restore(ctx, id, digest)
	if err != nil {
		t.Fatal(err)
	}
	got, err := restored.Inputs(ctx)
	if err != nil || string(got[0]) != string(inputs[0]) || string(got[1]) != string(inputs[1]) {
		t.Fatal("disk restart lost exact original")
	}
}
