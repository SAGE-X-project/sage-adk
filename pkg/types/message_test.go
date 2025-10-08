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
	"strings"
	"testing"
	"time"
)

func TestMessageRole_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		role  MessageRole
		valid bool
	}{
		{"user role is valid", MessageRoleUser, true},
		{"agent role is valid", MessageRoleAgent, true},
		{"empty role is invalid", MessageRole(""), false},
		{"unknown role is invalid", MessageRole("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.role.IsValid()
			if got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestTextPart_GetKind(t *testing.T) {
	part := &TextPart{
		Kind: string(PartKindText),
		Text: "Hello",
	}

	if got := part.GetKind(); got != string(PartKindText) {
		t.Errorf("GetKind() = %v, want %v", got, string(PartKindText))
	}
}

func TestTextPart_Validate(t *testing.T) {
	tests := []struct {
		name    string
		part    *TextPart
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid text part",
			part: &TextPart{
				Kind: string(PartKindText),
				Text: "Hello, world!",
			},
			wantErr: false,
		},
		{
			name: "empty text",
			part: &TextPart{
				Kind: string(PartKindText),
				Text: "",
			},
			wantErr: true,
			errMsg:  "text cannot be empty",
		},
		{
			name: "wrong kind",
			part: &TextPart{
				Kind: "wrong",
				Text: "Hello",
			},
			wantErr: true,
			errMsg:  "kind must be 'text'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.part.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestFilePart_GetKind(t *testing.T) {
	part := &FilePart{
		Kind: string(PartKindFile),
		File: &FileWithBytes{
			Name:     "test.txt",
			MimeType: "text/plain",
			Bytes:    []byte("content"),
		},
	}

	if got := part.GetKind(); got != string(PartKindFile) {
		t.Errorf("GetKind() = %v, want %v", got, string(PartKindFile))
	}
}

func TestFilePart_Validate(t *testing.T) {
	tests := []struct {
		name    string
		part    *FilePart
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid file with bytes",
			part: &FilePart{
				Kind: string(PartKindFile),
				File: &FileWithBytes{
					Name:     "test.txt",
					MimeType: "text/plain",
					Bytes:    []byte("content"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid file with URI",
			part: &FilePart{
				Kind: string(PartKindFile),
				File: &FileWithURI{
					Name:     "test.txt",
					MimeType: "text/plain",
					URI:      "https://example.com/file.txt",
				},
			},
			wantErr: false,
		},
		{
			name: "missing file content",
			part: &FilePart{
				Kind: string(PartKindFile),
				File: nil,
			},
			wantErr: true,
			errMsg:  "file content is required",
		},
		{
			name: "wrong kind",
			part: &FilePart{
				Kind: "wrong",
				File: &FileWithBytes{
					Name:     "test.txt",
					MimeType: "text/plain",
					Bytes:    []byte("content"),
				},
			},
			wantErr: true,
			errMsg:  "kind must be 'file'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.part.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestDataPart_GetKind(t *testing.T) {
	part := &DataPart{
		Kind: string(PartKindData),
		Data: map[string]interface{}{"key": "value"},
	}

	if got := part.GetKind(); got != string(PartKindData) {
		t.Errorf("GetKind() = %v, want %v", got, string(PartKindData))
	}
}

func TestDataPart_Validate(t *testing.T) {
	tests := []struct {
		name    string
		part    *DataPart
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid data part",
			part: &DataPart{
				Kind: string(PartKindData),
				Data: map[string]interface{}{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "nil data",
			part: &DataPart{
				Kind: string(PartKindData),
				Data: nil,
			},
			wantErr: true,
			errMsg:  "data cannot be nil",
		},
		{
			name: "wrong kind",
			part: &DataPart{
				Kind: "wrong",
				Data: map[string]interface{}{"key": "value"},
			},
			wantErr: true,
			errMsg:  "kind must be 'data'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.part.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestMessage_Validate(t *testing.T) {
	tests := []struct {
		name    string
		msg     *Message
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid A2A message",
			msg: &Message{
				MessageID: "msg-123",
				Role:      MessageRoleUser,
				Parts: []Part{
					&TextPart{
						Kind: string(PartKindText),
						Text: "Hello",
					},
				},
				Kind: "message",
			},
			wantErr: false,
		},
		{
			name: "valid SAGE message",
			msg: &Message{
				MessageID: "msg-456",
				Role:      MessageRoleAgent,
				Parts: []Part{
					&TextPart{
						Kind: string(PartKindText),
						Text: "Response",
					},
				},
				Kind: "message",
				Security: &SecurityMetadata{
					Mode:      ProtocolModeSAGE,
					AgentDID:  "did:sage:eth:0x123",
					Nonce:     "nonce-789",
					Timestamp: time.Now(),
				},
			},
			wantErr: false,
		},
		{
			name: "empty MessageID",
			msg: &Message{
				MessageID: "",
				Role:      MessageRoleUser,
				Parts: []Part{
					&TextPart{
						Kind: string(PartKindText),
						Text: "Hello",
					},
				},
			},
			wantErr: true,
			errMsg:  "MessageID is required",
		},
		{
			name: "invalid role",
			msg: &Message{
				MessageID: "msg-123",
				Role:      "invalid",
				Parts: []Part{
					&TextPart{
						Kind: string(PartKindText),
						Text: "Hello",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid role",
		},
		{
			name: "empty parts",
			msg: &Message{
				MessageID: "msg-123",
				Role:      MessageRoleUser,
				Parts:     []Part{},
			},
			wantErr: true,
			errMsg:  "at least one part is required",
		},
		{
			name: "SAGE without AgentDID",
			msg: &Message{
				MessageID: "msg-789",
				Role:      MessageRoleUser,
				Parts: []Part{
					&TextPart{
						Kind: string(PartKindText),
						Text: "Hello",
					},
				},
				Security: &SecurityMetadata{
					Mode: ProtocolModeSAGE,
				},
			},
			wantErr: true,
			errMsg:  "AgentDID required for SAGE mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestMessage_JSON(t *testing.T) {
	msg := &Message{
		MessageID: "msg-123",
		Role:      MessageRoleUser,
		Parts: []Part{
			&TextPart{
				Kind: string(PartKindText),
				Text: "Hello",
			},
		},
		Kind: "message",
	}

	// Marshal to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Unmarshal from JSON
	var got Message
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// Verify fields
	if got.MessageID != msg.MessageID {
		t.Errorf("MessageID = %v, want %v", got.MessageID, msg.MessageID)
	}
	if got.Role != msg.Role {
		t.Errorf("Role = %v, want %v", got.Role, msg.Role)
	}
	if len(got.Parts) != len(msg.Parts) {
		t.Errorf("Parts length = %v, want %v", len(got.Parts), len(msg.Parts))
	}
}
