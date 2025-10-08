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

package a2a

import (
	"testing"

	"github.com/sage-x-project/sage-adk/pkg/types"
	a2a "trpc.group/trpc-go/trpc-a2a-go/protocol"
)

func TestToA2AMessage_Success(t *testing.T) {
	contextID := "ctx-123"
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("Hello")},
	)
	msg.ContextID = &contextID

	a2aMsg, err := toA2AMessage(msg)
	if err != nil {
		t.Fatalf("toA2AMessage() error = %v", err)
	}

	if a2aMsg.MessageID != msg.MessageID {
		t.Errorf("MessageID = %v, want %v", a2aMsg.MessageID, msg.MessageID)
	}

	if a2aMsg.Role != a2a.MessageRole(msg.Role) {
		t.Errorf("Role = %v, want %v", a2aMsg.Role, msg.Role)
	}

	if a2aMsg.ContextID == nil || *a2aMsg.ContextID != *msg.ContextID {
		t.Errorf("ContextID = %v, want %v", a2aMsg.ContextID, msg.ContextID)
	}

	if len(a2aMsg.Parts) != len(msg.Parts) {
		t.Errorf("Parts length = %v, want %v", len(a2aMsg.Parts), len(msg.Parts))
	}
}

func TestFromA2AMessage_Success(t *testing.T) {
	contextID := "ctx-456"
	a2aMsg := a2a.NewMessage(
		a2a.MessageRoleAgent,
		[]a2a.Part{a2a.NewTextPart("Response")},
	)
	a2aMsg.ContextID = &contextID

	msg, err := fromA2AMessage(&a2aMsg)
	if err != nil {
		t.Fatalf("fromA2AMessage() error = %v", err)
	}

	if msg.MessageID != a2aMsg.MessageID {
		t.Errorf("MessageID = %v, want %v", msg.MessageID, a2aMsg.MessageID)
	}

	if string(msg.Role) != string(a2aMsg.Role) {
		t.Errorf("Role = %v, want %v", msg.Role, a2aMsg.Role)
	}

	if msg.ContextID == nil || *msg.ContextID != *a2aMsg.ContextID {
		t.Errorf("ContextID = %v, want %v", msg.ContextID, a2aMsg.ContextID)
	}

	if len(msg.Parts) != len(a2aMsg.Parts) {
		t.Errorf("Parts length = %v, want %v", len(msg.Parts), len(a2aMsg.Parts))
	}
}

func TestToA2AParts_TextPart(t *testing.T) {
	parts := []types.Part{
		types.NewTextPart("Hello"),
		types.NewTextPart("World"),
	}

	a2aParts, err := toA2AParts(parts)
	if err != nil {
		t.Fatalf("toA2AParts() error = %v", err)
	}

	if len(a2aParts) != len(parts) {
		t.Errorf("Parts length = %v, want %v", len(a2aParts), len(parts))
	}

	for i, p := range a2aParts {
		textPart, ok := p.(a2a.TextPart)
		if !ok {
			t.Fatalf("Part %d is not TextPart", i)
		}

		origPart := parts[i].(*types.TextPart)
		if textPart.Text != origPart.Text {
			t.Errorf("Part %d Text = %v, want %v", i, textPart.Text, origPart.Text)
		}
	}
}

func TestFromA2AParts_TextPart(t *testing.T) {
	a2aParts := []a2a.Part{
		a2a.NewTextPart("Hello"),
		a2a.NewTextPart("World"),
	}

	parts, err := fromA2AParts(a2aParts)
	if err != nil {
		t.Fatalf("fromA2AParts() error = %v", err)
	}

	if len(parts) != len(a2aParts) {
		t.Errorf("Parts length = %v, want %v", len(parts), len(a2aParts))
	}

	for i, p := range parts {
		textPart, ok := p.(*types.TextPart)
		if !ok {
			t.Fatalf("Part %d is not *TextPart", i)
		}

		origPart := a2aParts[i].(a2a.TextPart)
		if textPart.Text != origPart.Text {
			t.Errorf("Part %d Text = %v, want %v", i, textPart.Text, origPart.Text)
		}
	}
}

func TestToA2AParts_FilePart(t *testing.T) {
	name := "test.txt"
	mimeType := "text/plain"
	content := []byte("test content")

	parts := []types.Part{
		&types.FilePart{
			Kind: string(types.PartKindFile),
			File: &types.FileWithBytes{
				Name:     name,
				MimeType: mimeType,
				Bytes:    content,
			},
		},
	}

	a2aParts, err := toA2AParts(parts)
	if err != nil {
		t.Fatalf("toA2AParts() error = %v", err)
	}

	if len(a2aParts) != 1 {
		t.Fatalf("Parts length = %v, want 1", len(a2aParts))
	}

	filePart, ok := a2aParts[0].(a2a.FilePart)
	if !ok {
		t.Fatalf("Part is not FilePart")
	}

	fileWithBytes, ok := filePart.File.(*a2a.FileWithBytes)
	if !ok {
		t.Fatalf("File is not *FileWithBytes")
	}

	if fileWithBytes.Name == nil || *fileWithBytes.Name != name {
		t.Errorf("Name = %v, want %v", fileWithBytes.Name, name)
	}
}

func TestToA2AParts_DataPart(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	parts := []types.Part{
		&types.DataPart{
			Kind: string(types.PartKindData),
			Data: data,
		},
	}

	a2aParts, err := toA2AParts(parts)
	if err != nil {
		t.Fatalf("toA2AParts() error = %v", err)
	}

	if len(a2aParts) != 1 {
		t.Fatalf("Parts length = %v, want 1", len(a2aParts))
	}

	dataPart, ok := a2aParts[0].(a2a.DataPart)
	if !ok {
		t.Fatalf("Part is not DataPart")
	}

	dataMap, ok := dataPart.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Data is not map[string]interface{}")
	}

	if dataMap["key"] != "value" {
		t.Errorf("Data[key] = %v, want value", dataMap["key"])
	}
}

func TestRoundTrip_Message(t *testing.T) {
	contextID := "ctx-roundtrip"
	original := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{
			types.NewTextPart("Test message"),
			&types.DataPart{
				Kind: string(types.PartKindData),
				Data: map[string]interface{}{"test": "data"},
			},
		},
	)
	original.ContextID = &contextID

	// Convert to A2A
	a2aMsg, err := toA2AMessage(original)
	if err != nil {
		t.Fatalf("toA2AMessage() error = %v", err)
	}

	// Convert back to sage-adk
	result, err := fromA2AMessage(&a2aMsg)
	if err != nil {
		t.Fatalf("fromA2AMessage() error = %v", err)
	}

	// Compare
	if result.MessageID != original.MessageID {
		t.Errorf("MessageID = %v, want %v", result.MessageID, original.MessageID)
	}

	if result.Role != original.Role {
		t.Errorf("Role = %v, want %v", result.Role, original.Role)
	}

	if len(result.Parts) != len(original.Parts) {
		t.Errorf("Parts length = %v, want %v", len(result.Parts), len(original.Parts))
	}
}
