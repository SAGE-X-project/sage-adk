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

package types

import (
	"strings"
	"testing"
)

func TestGenerateMessageID(t *testing.T) {
	id := GenerateMessageID()
	if !strings.HasPrefix(id, "msg-") {
		t.Errorf("MessageID = %v, should start with 'msg-'", id)
	}

	// Generate another to ensure uniqueness
	id2 := GenerateMessageID()
	if id == id2 {
		t.Error("Generated IDs should be unique")
	}
}

func TestGenerateContextID(t *testing.T) {
	id := GenerateContextID()
	if !strings.HasPrefix(id, "ctx-") {
		t.Errorf("ContextID = %v, should start with 'ctx-'", id)
	}
}

func TestGenerateTaskID(t *testing.T) {
	id := GenerateTaskID()
	if !strings.HasPrefix(id, "task-") {
		t.Errorf("TaskID = %v, should start with 'task-'", id)
	}
}

func TestGenerateArtifactID(t *testing.T) {
	id := GenerateArtifactID()
	if !strings.HasPrefix(id, "artifact-") {
		t.Errorf("ArtifactID = %v, should start with 'artifact-'", id)
	}
}

func TestGenerateAgentID(t *testing.T) {
	id := GenerateAgentID()
	if !strings.HasPrefix(id, "agent-") {
		t.Errorf("AgentID = %v, should start with 'agent-'", id)
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce := GenerateNonce()
	if !strings.HasPrefix(nonce, "nonce-") {
		t.Errorf("Nonce = %v, should start with 'nonce-'", nonce)
	}
}

func TestNewTextPart(t *testing.T) {
	text := "Hello, world!"
	part := NewTextPart(text)

	if part.Kind != string(PartKindText) {
		t.Errorf("Kind = %v, want %v", part.Kind, string(PartKindText))
	}
	if part.Text != text {
		t.Errorf("Text = %v, want %v", part.Text, text)
	}
	if part.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}

func TestNewFilePartWithBytes(t *testing.T) {
	name := "test.txt"
	mimeType := "text/plain"
	bytes := []byte("content")

	part := NewFilePartWithBytes(name, mimeType, bytes)

	if part.Kind != string(PartKindFile) {
		t.Errorf("Kind = %v, want %v", part.Kind, string(PartKindFile))
	}

	fileWithBytes, ok := part.File.(*FileWithBytes)
	if !ok {
		t.Fatal("File should be FileWithBytes")
	}

	if fileWithBytes.Name != name {
		t.Errorf("Name = %v, want %v", fileWithBytes.Name, name)
	}
	if fileWithBytes.MimeType != mimeType {
		t.Errorf("MimeType = %v, want %v", fileWithBytes.MimeType, mimeType)
	}
	if string(fileWithBytes.Bytes) != string(bytes) {
		t.Errorf("Bytes = %v, want %v", fileWithBytes.Bytes, bytes)
	}
}

func TestNewFilePartWithURI(t *testing.T) {
	name := "test.txt"
	mimeType := "text/plain"
	uri := "https://example.com/file.txt"

	part := NewFilePartWithURI(name, mimeType, uri)

	if part.Kind != string(PartKindFile) {
		t.Errorf("Kind = %v, want %v", part.Kind, string(PartKindFile))
	}

	fileWithURI, ok := part.File.(*FileWithURI)
	if !ok {
		t.Fatal("File should be FileWithURI")
	}

	if fileWithURI.Name != name {
		t.Errorf("Name = %v, want %v", fileWithURI.Name, name)
	}
	if fileWithURI.MimeType != mimeType {
		t.Errorf("MimeType = %v, want %v", fileWithURI.MimeType, mimeType)
	}
	if fileWithURI.URI != uri {
		t.Errorf("URI = %v, want %v", fileWithURI.URI, uri)
	}
}

func TestNewDataPart(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	part := NewDataPart(data)

	if part.Kind != string(PartKindData) {
		t.Errorf("Kind = %v, want %v", part.Kind, string(PartKindData))
	}
	if part.Data == nil {
		t.Error("Data should not be nil")
	}
}

func TestNewMessage(t *testing.T) {
	parts := []Part{NewTextPart("Hello")}
	msg := NewMessage(MessageRoleUser, parts)

	if msg.MessageID == "" {
		t.Error("MessageID should be generated")
	}
	if !strings.HasPrefix(msg.MessageID, "msg-") {
		t.Errorf("MessageID = %v, should start with 'msg-'", msg.MessageID)
	}
	if msg.Role != MessageRoleUser {
		t.Errorf("Role = %v, want %v", msg.Role, MessageRoleUser)
	}
	if len(msg.Parts) != len(parts) {
		t.Errorf("Parts length = %v, want %v", len(msg.Parts), len(parts))
	}
	if msg.Kind != "message" {
		t.Errorf("Kind = %v, want 'message'", msg.Kind)
	}
	if msg.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}

func TestNewMessageWithContext(t *testing.T) {
	parts := []Part{NewTextPart("Hello")}
	taskID := "task-123"
	contextID := "ctx-456"

	msg := NewMessageWithContext(MessageRoleUser, parts, &taskID, &contextID)

	if msg.MessageID == "" {
		t.Error("MessageID should be generated")
	}
	if msg.TaskID == nil || *msg.TaskID != taskID {
		t.Errorf("TaskID = %v, want %v", msg.TaskID, taskID)
	}
	if msg.ContextID == nil || *msg.ContextID != contextID {
		t.Errorf("ContextID = %v, want %v", msg.ContextID, contextID)
	}
}

func TestNewArtifact(t *testing.T) {
	name := "result.txt"
	description := "Test result"
	parts := []Part{NewTextPart("Result content")}

	artifact := NewArtifact(name, description, parts)

	if artifact.ArtifactID == "" {
		t.Error("ArtifactID should be generated")
	}
	if !strings.HasPrefix(artifact.ArtifactID, "artifact-") {
		t.Errorf("ArtifactID = %v, should start with 'artifact-'", artifact.ArtifactID)
	}
	if artifact.Name != name {
		t.Errorf("Name = %v, want %v", artifact.Name, name)
	}
	if artifact.Description != description {
		t.Errorf("Description = %v, want %v", artifact.Description, description)
	}
	if len(artifact.Parts) != len(parts) {
		t.Errorf("Parts length = %v, want %v", len(artifact.Parts), len(parts))
	}
	if artifact.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}
