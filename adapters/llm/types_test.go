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

package llm

import (
	"testing"
)

func TestMessageRole_String(t *testing.T) {
	tests := []struct {
		name string
		role MessageRole
		want string
	}{
		{"user role", RoleUser, "user"},
		{"assistant role", RoleAssistant, "assistant"},
		{"system role", RoleSystem, "system"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.role) != tt.want {
				t.Errorf("Role = %v, want %v", tt.role, tt.want)
			}
		})
	}
}

func TestNewCompletionRequest(t *testing.T) {
	req := &CompletionRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	if req.Model != "gpt-4" {
		t.Errorf("Model = %v, want gpt-4", req.Model)
	}

	if len(req.Messages) != 1 {
		t.Errorf("Messages length = %v, want 1", len(req.Messages))
	}

	if req.Messages[0].Role != RoleUser {
		t.Errorf("Message role = %v, want user", req.Messages[0].Role)
	}
}

func TestCompletionResponse_Validation(t *testing.T) {
	resp := &CompletionResponse{
		ID:           "resp-123",
		Model:        "gpt-4",
		Content:      "Hello, how can I help you?",
		FinishReason: "stop",
		Usage: &Usage{
			PromptTokens:     10,
			CompletionTokens: 8,
			TotalTokens:      18,
		},
	}

	if resp.ID == "" {
		t.Error("ID should not be empty")
	}

	if resp.Usage.TotalTokens != 18 {
		t.Errorf("TotalTokens = %v, want 18", resp.Usage.TotalTokens)
	}
}
