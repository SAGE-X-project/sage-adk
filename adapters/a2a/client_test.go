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
	a2aprotocol "trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

func TestNewClient(t *testing.T) {
	client, err := NewClient("http://localhost:8080/")
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	if client.client == nil {
		t.Fatal("NewClient() created client with nil underlying client")
	}
}

func TestNewClient_InvalidURL(t *testing.T) {
	_, err := NewClient("://invalid-url")
	if err == nil {
		t.Error("NewClient() with invalid URL should return error")
	}
}

func TestConvertMessageToA2A(t *testing.T) {
	tests := []struct {
		name string
		msg  *types.Message
		want a2aprotocol.Message
	}{
		{
			name: "text message",
			msg: &types.Message{
				Role:  types.MessageRoleUser,
				Parts: []types.Part{types.NewTextPart("Hello")},
			},
			want: a2aprotocol.Message{
				Role: a2aprotocol.MessageRoleUser,
				Parts: []a2aprotocol.Part{
					a2aprotocol.TextPart{
						Kind: "text",
						Text: "Hello",
					},
				},
			},
		},
		{
			name: "message with file bytes",
			msg: &types.Message{
				Role: types.MessageRoleUser,
				Parts: []types.Part{
					types.NewFilePartWithBytes("test.txt", "text/plain", []byte("test")),
				},
			},
			want: a2aprotocol.Message{
				Role: a2aprotocol.MessageRoleUser,
				Parts: []a2aprotocol.Part{
					a2aprotocol.FilePart{
						Kind: "file",
						File: &a2aprotocol.FileWithBytes{
							Name:     stringPtr("test.txt"),
							MimeType: stringPtr("text/plain"),
							Bytes:    "dGVzdA==", // base64 encoded "test"
						},
					},
				},
			},
		},
		{
			name: "message with data part",
			msg: &types.Message{
				Role: types.MessageRoleAgent,
				Parts: []types.Part{
					types.NewDataPart(map[string]interface{}{
						"key": "value",
					}),
				},
			},
			want: a2aprotocol.Message{
				Role: a2aprotocol.MessageRoleAgent,
				Parts: []a2aprotocol.Part{
					a2aprotocol.DataPart{
						Kind: "data",
						Data: map[string]interface{}{
							"key": "value",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertMessageToA2A(tt.msg)

			if got.Role != tt.want.Role {
				t.Errorf("Role = %v, want %v", got.Role, tt.want.Role)
			}

			if len(got.Parts) != len(tt.want.Parts) {
				t.Errorf("Parts length = %d, want %d", len(got.Parts), len(tt.want.Parts))
			}

			for i, part := range got.Parts {
				wantPart := tt.want.Parts[i]
				if part.GetKind() != wantPart.GetKind() {
					t.Errorf("Part[%d].Kind = %v, want %v", i, part.GetKind(), wantPart.GetKind())
				}
			}
		})
	}
}

func TestConvertMessageFromA2A(t *testing.T) {
	tests := []struct {
		name string
		msg  *a2aprotocol.Message
		want *types.Message
	}{
		{
			name: "text message",
			msg: &a2aprotocol.Message{
				Role: a2aprotocol.MessageRoleAgent,
				Parts: []a2aprotocol.Part{
					a2aprotocol.TextPart{
						Kind: "text",
						Text: "Hello back",
					},
				},
			},
			want: &types.Message{
				Role:  types.MessageRoleAgent,
				Parts: []types.Part{types.NewTextPart("Hello back")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertMessageFromA2A(tt.msg)

			if got.Role != tt.want.Role {
				t.Errorf("Role = %v, want %v", got.Role, tt.want.Role)
			}

			if len(got.Parts) != len(tt.want.Parts) {
				t.Errorf("Parts length = %d, want %d", len(got.Parts), len(tt.want.Parts))
			}
		})
	}
}

func TestConvertPartToA2A(t *testing.T) {
	tests := []struct {
		name string
		part types.Part
		want string // part kind
	}{
		{
			name: "text part",
			part: types.NewTextPart("test"),
			want: "text",
		},
		{
			name: "file part with bytes",
			part: types.NewFilePartWithBytes("test.txt", "text/plain", []byte("test")),
			want: "file",
		},
		{
			name: "file part with URI",
			part: types.NewFilePartWithURI("file.pdf", "application/pdf", "http://example.com/file.pdf"),
			want: "file",
		},
		{
			name: "data part",
			part: types.NewDataPart(map[string]interface{}{"key": "value"}),
			want: "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertPartToA2A(tt.part)
			if got.GetKind() != tt.want {
				t.Errorf("Kind = %v, want %v", got.GetKind(), tt.want)
			}
		})
	}
}

func TestConvertPartFromA2A(t *testing.T) {
	tests := []struct {
		name string
		part a2aprotocol.Part
	}{
		{
			name: "text part",
			part: a2aprotocol.TextPart{
				Kind: "text",
				Text: "test",
			},
		},
		{
			name: "file part",
			part: a2aprotocol.FilePart{
				Kind: "file",
				File: &a2aprotocol.FileWithBytes{
					Name:     stringPtr("test.txt"),
					MimeType: stringPtr("text/plain"),
					Bytes:    "dGVzdA==",
				},
			},
		},
		{
			name: "data part",
			part: a2aprotocol.DataPart{
				Kind: "data",
				Data: map[string]interface{}{"key": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertPartFromA2A(tt.part)
			if got == nil {
				t.Error("convertPartFromA2A() returned nil")
			}
		})
	}
}
