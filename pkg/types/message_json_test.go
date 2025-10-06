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
	"encoding/json"
	"testing"
)

func TestFilePart_JSONRoundTrip_Bytes(t *testing.T) {
	part := &FilePart{
		Kind: string(PartKindFile),
		File: &FileWithBytes{
			Name:     "test.txt",
			MimeType: "text/plain",
			Bytes:    []byte("content"),
		},
	}

	// Marshal
	data, err := json.Marshal(part)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Unmarshal
	var got FilePart
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Verify
	fileBytes, ok := got.File.(*FileWithBytes)
	if !ok {
		t.Fatal("File should be FileWithBytes")
	}

	if fileBytes.Name != "test.txt" {
		t.Errorf("Name = %v, want test.txt", fileBytes.Name)
	}
}

func TestFilePart_JSONRoundTrip_URI(t *testing.T) {
	part := &FilePart{
		Kind: string(PartKindFile),
		File: &FileWithURI{
			Name:     "test.txt",
			MimeType: "text/plain",
			URI:      "https://example.com/file.txt",
		},
	}

	// Marshal
	data, err := json.Marshal(part)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Unmarshal
	var got FilePart
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Verify
	fileURI, ok := got.File.(*FileWithURI)
	if !ok {
		t.Fatal("File should be FileWithURI")
	}

	if fileURI.URI != "https://example.com/file.txt" {
		t.Errorf("URI = %v, want https://example.com/file.txt", fileURI.URI)
	}
}

func TestFilePart_UnmarshalJSON_Error(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "invalid json",
			json: `{"kind":"file","file":invalid}`,
		},
		{
			name: "invalid file content",
			json: `{"kind":"file","file":{"invalid":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var part FilePart
			if err := json.Unmarshal([]byte(tt.json), &part); err == nil {
				t.Error("Expected unmarshal error, got nil")
			}
		})
	}
}

func TestMessage_UnmarshalJSON_MultiplePartTypes(t *testing.T) {
	jsonData := `{
		"messageId": "msg-123",
		"role": "user",
		"kind": "message",
		"parts": [
			{
				"kind": "text",
				"text": "Hello"
			},
			{
				"kind": "file",
				"file": {
					"name": "test.txt",
					"mimeType": "text/plain",
					"bytes": "Y29udGVudA=="
				}
			},
			{
				"kind": "data",
				"data": {"key": "value"}
			}
		]
	}`

	var msg Message
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if len(msg.Parts) != 3 {
		t.Fatalf("Parts length = %v, want 3", len(msg.Parts))
	}

	// Check text part
	textPart, ok := msg.Parts[0].(*TextPart)
	if !ok {
		t.Error("First part should be TextPart")
	}
	if textPart.Text != "Hello" {
		t.Errorf("Text = %v, want Hello", textPart.Text)
	}

	// Check file part
	filePart, ok := msg.Parts[1].(*FilePart)
	if !ok {
		t.Error("Second part should be FilePart")
	}
	if filePart.File == nil {
		t.Error("File should not be nil")
	}

	// Check data part
	dataPart, ok := msg.Parts[2].(*DataPart)
	if !ok {
		t.Error("Third part should be DataPart")
	}
	if dataPart.Data == nil {
		t.Error("Data should not be nil")
	}
}

func TestArtifact_UnmarshalJSON(t *testing.T) {
	jsonData := `{
		"artifactId": "artifact-123",
		"name": "result.txt",
		"parts": [
			{
				"kind": "text",
				"text": "Result"
			}
		]
	}`

	var artifact Artifact
	if err := json.Unmarshal([]byte(jsonData), &artifact); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if artifact.ArtifactID != "artifact-123" {
		t.Errorf("ArtifactID = %v, want artifact-123", artifact.ArtifactID)
	}

	if len(artifact.Parts) != 1 {
		t.Fatalf("Parts length = %v, want 1", len(artifact.Parts))
	}
}

func TestUnmarshalPart_UnsupportedKind(t *testing.T) {
	jsonData := `{"kind":"unsupported","data":"test"}`

	var msg Message
	fullJSON := `{
		"messageId": "msg-123",
		"role": "user",
		"kind": "message",
		"parts": [` + jsonData + `]
	}`

	if err := json.Unmarshal([]byte(fullJSON), &msg); err == nil {
		t.Error("Expected unmarshal error for unsupported part kind")
	}
}
