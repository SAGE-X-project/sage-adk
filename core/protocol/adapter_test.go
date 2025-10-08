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

package protocol

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestProtocolMode_String(t *testing.T) {
	tests := []struct {
		name string
		mode ProtocolMode
		want string
	}{
		{"auto mode", ProtocolAuto, "auto"},
		{"a2a mode", ProtocolA2A, "a2a"},
		{"sage mode", ProtocolSAGE, "sage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectProtocol_SAGE(t *testing.T) {
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)
	msg.Security = &types.SecurityMetadata{
		Mode: types.ProtocolModeSAGE,
	}

	got := DetectProtocol(msg)
	if got != ProtocolSAGE {
		t.Errorf("DetectProtocol() = %v, want %v", got, ProtocolSAGE)
	}
}

func TestDetectProtocol_A2A(t *testing.T) {
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	got := DetectProtocol(msg)
	if got != ProtocolA2A {
		t.Errorf("DetectProtocol() = %v, want %v", got, ProtocolA2A)
	}
}

func TestDetectProtocol_NoSecurity(t *testing.T) {
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)
	msg.Security = nil

	got := DetectProtocol(msg)
	if got != ProtocolA2A {
		t.Errorf("DetectProtocol() = %v, want %v", got, ProtocolA2A)
	}
}

func TestMockAdapter_Name(t *testing.T) {
	adapter := NewMockAdapter("test-adapter")

	got := adapter.Name()
	want := "test-adapter"
	if got != want {
		t.Errorf("Name() = %v, want %v", got, want)
	}
}

func TestMockAdapter_SendMessage(t *testing.T) {
	adapter := NewMockAdapter("test")
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	err := adapter.SendMessage(context.Background(), msg)
	if err != nil {
		t.Errorf("SendMessage() error = %v", err)
	}

	if len(adapter.SentMessages) != 1 {
		t.Errorf("SentMessages count = %v, want 1", len(adapter.SentMessages))
	}
}

func TestMockAdapter_ReceiveMessage(t *testing.T) {
	adapter := NewMockAdapter("test")
	expected := types.NewMessage(
		types.MessageRoleAgent,
		[]types.Part{types.NewTextPart("response")},
	)
	adapter.ReceivedMessages = []*types.Message{expected}

	got, err := adapter.ReceiveMessage(context.Background())
	if err != nil {
		t.Errorf("ReceiveMessage() error = %v", err)
	}

	if got.MessageID != expected.MessageID {
		t.Errorf("ReceiveMessage() MessageID = %v, want %v", got.MessageID, expected.MessageID)
	}
}

func TestMockAdapter_ReceiveMessage_Empty(t *testing.T) {
	adapter := NewMockAdapter("test")

	_, err := adapter.ReceiveMessage(context.Background())
	if err == nil {
		t.Error("ReceiveMessage() should return error when no messages")
	}
}

func TestMockAdapter_Verify(t *testing.T) {
	adapter := NewMockAdapter("test")
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	err := adapter.Verify(context.Background(), msg)
	if err != nil {
		t.Errorf("Verify() error = %v", err)
	}
}

func TestMockAdapter_SupportsStreaming(t *testing.T) {
	tests := []struct {
		name      string
		streaming bool
	}{
		{"supports streaming", true},
		{"no streaming", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewMockAdapter("test")
			adapter.Streaming = tt.streaming

			got := adapter.SupportsStreaming()
			if got != tt.streaming {
				t.Errorf("SupportsStreaming() = %v, want %v", got, tt.streaming)
			}
		})
	}
}

func TestMockAdapter_Stream(t *testing.T) {
	adapter := NewMockAdapter("test")
	adapter.Streaming = true

	called := false
	fn := func(chunk string) error {
		called = true
		return nil
	}

	err := adapter.Stream(context.Background(), fn)
	if err != nil {
		t.Errorf("Stream() error = %v", err)
	}

	if !called {
		t.Error("Stream() should call the stream function")
	}
}

func TestMockAdapter_Stream_NotSupported(t *testing.T) {
	adapter := NewMockAdapter("test")
	adapter.Streaming = false

	fn := func(chunk string) error {
		return nil
	}

	err := adapter.Stream(context.Background(), fn)
	if err == nil {
		t.Error("Stream() should return error when streaming not supported")
	}
}
