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

func TestAgent_Process_Echo(t *testing.T) {
	agent, err := NewAgent("echo").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			return msg.Reply(msg.Text())
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Create test message
	inputText := "Hello, World!"
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart(inputText)},
	)

	// Process message
	response, err := agent.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if response == nil {
		t.Fatal("Process() should return response")
	}

	// Extract response text
	responseText := extractText(response)
	if responseText != inputText {
		t.Errorf("Response text = %v, want %v", responseText, inputText)
	}
}

func TestAgent_Process_Transform(t *testing.T) {
	agent, err := NewAgent("upper").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			text := msg.Text()
			return msg.Reply("ECHO: " + text)
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("hello")},
	)

	response, err := agent.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	responseText := extractText(response)
	want := "ECHO: hello"
	if responseText != want {
		t.Errorf("Response text = %v, want %v", responseText, want)
	}
}

func TestAgent_Process_MultipleParts(t *testing.T) {
	agent, err := NewAgent("multi").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			parts := msg.Parts()
			return msg.ReplyWithParts(parts)
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	parts := []types.Part{
		types.NewTextPart("Part 1"),
		types.NewTextPart("Part 2"),
	}

	msg := types.NewMessage(types.MessageRoleUser, parts)

	response, err := agent.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if len(response.Parts) != 2 {
		t.Errorf("Response parts count = %v, want 2", len(response.Parts))
	}
}

func TestMessageContext_ContextID(t *testing.T) {
	var receivedContextID string

	agent, err := NewAgent("ctx").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			receivedContextID = msg.ContextID()
			return msg.Reply("ok")
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	contextID := "ctx-123"
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)
	msg.ContextID = &contextID

	_, err = agent.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if receivedContextID != contextID {
		t.Errorf("ContextID = %v, want %v", receivedContextID, contextID)
	}
}

func TestMessageContext_MessageID(t *testing.T) {
	var receivedMessageID string

	agent, err := NewAgent("msgid").
		OnMessage(func(ctx context.Context, msg MessageContext) error {
			receivedMessageID = msg.MessageID()
			return msg.Reply("ok")
		}).
		Build()

	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)
	originalID := msg.MessageID

	_, err = agent.Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if receivedMessageID != originalID {
		t.Errorf("MessageID = %v, want %v", receivedMessageID, originalID)
	}
}

// Helper function to extract text from message
func extractText(msg *types.Message) string {
	for _, part := range msg.Parts {
		if textPart, ok := part.(*types.TextPart); ok {
			return textPart.Text
		}
	}
	return ""
}
