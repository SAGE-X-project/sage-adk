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
				Role: string(types.MessageRoleUser),
				Parts: []a2aprotocol.Part{
					{
						Type: a2aprotocol.PartTypeText,
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
					types.NewFilePart(types.FileWithBytes{
						Name:     "test.txt",
						MimeType: "text/plain",
						Data:     "dGVzdA==",
					}),
				},
			},
			want: a2aprotocol.Message{
				Role: string(types.MessageRoleUser),
				Parts: []a2aprotocol.Part{
					{
						Type: a2aprotocol.PartTypeFile,
						File: &a2aprotocol.FileWithBytes{
							Name:     "test.txt",
							MimeType: "text/plain",
							Data:     "dGVzdA==",
						},
					},
				},
			},
		},
		{
			name: "message with data part",
			msg: &types.Message{
				Role: types.MessageRoleAssistant,
				Parts: []types.Part{
					types.NewDataPart(map[string]interface{}{
						"key": "value",
					}),
				},
			},
			want: a2aprotocol.Message{
				Role: string(types.MessageRoleAssistant),
				Parts: []a2aprotocol.Part{
					{
						Type: a2aprotocol.PartTypeData,
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
				if part.Type != wantPart.Type {
					t.Errorf("Part[%d].Type = %v, want %v", i, part.Type, wantPart.Type)
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
				Role: string(types.MessageRoleAssistant),
				Parts: []a2aprotocol.Part{
					{
						Type: a2aprotocol.PartTypeText,
						Text: "Hello back",
					},
				},
			},
			want: &types.Message{
				Role:  types.MessageRoleAssistant,
				Parts: []types.Part{types.NewTextPart("Hello back")},
			},
		},
		{
			name: "message with file bytes",
			msg: &a2aprotocol.Message{
				Role: string(types.MessageRoleAssistant),
				Parts: []a2aprotocol.Part{
					{
						Type: a2aprotocol.PartTypeFile,
						File: &a2aprotocol.FileWithBytes{
							Name:     "response.json",
							MimeType: "application/json",
							Data:     "eyJrZXkiOiJ2YWx1ZSJ9",
						},
					},
				},
			},
			want: &types.Message{
				Role: types.MessageRoleAssistant,
				Parts: []types.Part{
					types.NewFilePart(types.FileWithBytes{
						Name:     "response.json",
						MimeType: "application/json",
						Data:     "eyJrZXkiOiJ2YWx1ZSJ9",
					}),
				},
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
		want string // part type
	}{
		{
			name: "text part",
			part: types.NewTextPart("test"),
			want: a2aprotocol.PartTypeText,
		},
		{
			name: "file part with bytes",
			part: types.NewFilePart(types.FileWithBytes{
				Name:     "test.txt",
				MimeType: "text/plain",
				Data:     "dGVzdA==",
			}),
			want: a2aprotocol.PartTypeFile,
		},
		{
			name: "file part with URI",
			part: types.NewFilePart(types.FileWithURI{
				URI:      "http://example.com/file.pdf",
				MimeType: "application/pdf",
			}),
			want: a2aprotocol.PartTypeFile,
		},
		{
			name: "data part",
			part: types.NewDataPart(map[string]interface{}{"key": "value"}),
			want: a2aprotocol.PartTypeData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertPartToA2A(tt.part)
			if got.Type != tt.want {
				t.Errorf("Type = %v, want %v", got.Type, tt.want)
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
			part: a2aprotocol.Part{
				Type: a2aprotocol.PartTypeText,
				Text: "test",
			},
		},
		{
			name: "file part",
			part: a2aprotocol.Part{
				Type: a2aprotocol.PartTypeFile,
				File: &a2aprotocol.FileWithBytes{
					Name:     "test.txt",
					MimeType: "text/plain",
					Data:     "dGVzdA==",
				},
			},
		},
		{
			name: "data part",
			part: a2aprotocol.Part{
				Type: a2aprotocol.PartTypeData,
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
