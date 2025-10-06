// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later

package agent

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestMessageContext_Text_Empty(t *testing.T) {
	agent, _ := NewAgent("test").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			text := msg.Text()
			if text != "" {
				t.Errorf("Text() = %v, want empty string", text)
			}
			return msg.Reply("ok")
		}).
		Build()

	// Message with no text parts
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{},
	)

	_, _ = agent.Process(context.Background(), msg)
}

func TestMessageContext_ContextID_Nil(t *testing.T) {
	agent, _ := NewAgent("test").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			contextID := msg.ContextID()
			if contextID != "" {
				t.Errorf("ContextID() = %v, want empty string", contextID)
			}
			return msg.Reply("ok")
		}).
		Build()

	// Message without context ID
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)
	msg.ContextID = nil

	_, _ = agent.Process(context.Background(), msg)
}

func TestMessageContext_Reply_Twice(t *testing.T) {
	agent, _ := NewAgent("test").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			// First reply should succeed
			if err := msg.Reply("first"); err != nil {
				t.Errorf("First Reply() error = %v", err)
			}

			// Second reply should fail
			err := msg.Reply("second")
			if err == nil {
				t.Error("Second Reply() should return error")
			}
			return nil
		}).
		Build()

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	_, _ = agent.Process(context.Background(), msg)
}

func TestMessageContext_ReplyWithParts_Twice(t *testing.T) {
	agent, _ := NewAgent("test").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			parts := []types.Part{types.NewTextPart("first")}

			// First reply should succeed
			if err := msg.ReplyWithParts(parts); err != nil {
				t.Errorf("First ReplyWithParts() error = %v", err)
			}

			// Second reply should fail
			err := msg.ReplyWithParts(parts)
			if err == nil {
				t.Error("Second ReplyWithParts() should return error")
			}
			return nil
		}).
		Build()

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	_, _ = agent.Process(context.Background(), msg)
}

func TestMessageContext_Response_PreservesContextID(t *testing.T) {
	contextID := "ctx-preserve"

	agent, _ := NewAgent("test").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			return msg.Reply("response")
		}).
		Build()

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)
	msg.ContextID = &contextID

	response, err := agent.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if response.ContextID == nil {
		t.Fatal("Response ContextID should not be nil")
	}

	if *response.ContextID != contextID {
		t.Errorf("Response ContextID = %v, want %v", *response.ContextID, contextID)
	}
}
