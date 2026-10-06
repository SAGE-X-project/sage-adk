// SPDX-License-Identifier: LGPL-3.0-or-later
package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/core/capture"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

type inputStore struct {
	original                 [][]byte
	persisted, fail, changed bool
}

func (s *inputStore) Create(context.Context, string, [][]byte) error {
	if s.fail {
		return errors.New("unavailable")
	}
	s.persisted = true
	s.original = [][]byte{[]byte(" raw\n")}
	return nil
}
func (s *inputStore) Load(context.Context, string) ([][]byte, error) {
	if s.changed {
		return [][]byte{[]byte("changed")}, nil
	}
	return s.original, nil
}
func TestProcessOriginalOrdering(t *testing.T) {
	s := &inputStore{}
	h, _ := capture.NewHost(s)
	handled := 0
	a, e := agent.NewAgent("capture-fixture").OnMessage(func(_ context.Context, m agent.MessageContext) error {
		if !s.persisted || m.Text() != "decoded" {
			t.Fatal("handler ran before capture")
		}
		handled++
		return m.Reply("fixed")
	}).Build()
	if e != nil {
		t.Fatal(e)
	}
	r, response, e := agent.ProcessOriginal(context.Background(), a, h, [][]byte{[]byte(" raw\n")}, func(_ context.Context, b [][]byte) (*types.Message, error) {
		if !s.persisted {
			t.Fatal("decoder before capture")
		}
		b[0][0] = 'x'
		return types.NewMessage(types.MessageRoleUser, []types.Part{types.NewTextPart("decoded")}), nil
	})
	if e != nil || r == nil || response == nil || handled != 1 {
		t.Fatalf("process: %v", e)
	}
	b, e := r.Inputs(context.Background())
	if e != nil || string(b[0]) != " raw\n" {
		t.Fatal("decoder changed original")
	}
}
func TestProcessOriginalRefusesBeforeHandler(t *testing.T) {
	for _, mode := range []string{"store", "decode", "nil", "invalid", "panic", "changed", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			s := &inputStore{fail: mode == "store"}
			h, _ := capture.NewHost(s)
			handled, decoded := 0, 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			a, _ := agent.NewAgent("capture-fixture").OnMessage(func(context.Context, agent.MessageContext) error { handled++; return nil }).Build()
			_, _, e := agent.ProcessOriginal(ctx, a, h, [][]byte{[]byte(" raw\n")}, func(context.Context, [][]byte) (*types.Message, error) {
				decoded++
				switch mode {
				case "decode":
					return nil, errors.New("decode failed")
				case "nil":
					return nil, nil
				case "invalid":
					return &types.Message{}, nil
				case "panic":
					panic("decoder failed")
				case "changed":
					s.changed = true
				case "cancel":
					cancel()
				}
				return types.NewMessage(types.MessageRoleUser, []types.Part{types.NewTextPart("fixed")}), nil
			})
			if e == nil || handled != 0 {
				t.Fatal("invalid original processed")
			}
			if mode == "store" && decoded != 0 {
				t.Fatal("decoder before durable capture")
			}
		})
	}
	if _, _, e := agent.ProcessOriginal(nil, nil, nil, nil, nil); e == nil {
		t.Fatal("missing services")
	}
}
