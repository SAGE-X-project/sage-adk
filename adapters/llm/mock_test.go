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
	"context"
	"testing"
)

func TestNewMockProvider(t *testing.T) {
	responses := []string{"response1", "response2"}
	provider := NewMockProvider("test", responses)

	if provider == nil {
		t.Fatal("NewMockProvider() should not return nil")
	}

	if provider.Name() != "test" {
		t.Errorf("Name() = %v, want test", provider.Name())
	}
}

func TestMockProvider_Complete_Success(t *testing.T) {
	responses := []string{"Hello, World!"}
	provider := NewMockProvider("test", responses)

	req := &CompletionRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens: 100,
	}

	resp, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Content != "Hello, World!" {
		t.Errorf("Content = %v, want 'Hello, World!'", resp.Content)
	}

	if resp.Model != "test-model" {
		t.Errorf("Model = %v, want test-model", resp.Model)
	}

	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %v, want stop", resp.FinishReason)
	}
}

func TestMockProvider_Complete_MultipleResponses(t *testing.T) {
	responses := []string{"First", "Second", "Third"}
	provider := NewMockProvider("test", responses)

	req := &CompletionRequest{
		Model:    "test-model",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	}

	// First call
	resp1, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("First Complete() error = %v", err)
	}
	if resp1.Content != "First" {
		t.Errorf("First response = %v, want First", resp1.Content)
	}

	// Second call
	resp2, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Second Complete() error = %v", err)
	}
	if resp2.Content != "Second" {
		t.Errorf("Second response = %v, want Second", resp2.Content)
	}

	// Third call
	resp3, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Third Complete() error = %v", err)
	}
	if resp3.Content != "Third" {
		t.Errorf("Third response = %v, want Third", resp3.Content)
	}
}

func TestMockProvider_Complete_NoMoreResponses(t *testing.T) {
	responses := []string{"Only one"}
	provider := NewMockProvider("test", responses)

	req := &CompletionRequest{
		Model:    "test-model",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	}

	// First call succeeds
	_, err := provider.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("First Complete() error = %v", err)
	}

	// Second call should fail
	_, err = provider.Complete(context.Background(), req)
	if err == nil {
		t.Error("Complete() should return error when no more responses")
	}
}

func TestMockProvider_SupportsStreaming(t *testing.T) {
	provider := NewMockProvider("test", []string{"response"})

	// Phase 1: Streaming not supported
	if provider.SupportsStreaming() {
		t.Error("SupportsStreaming() should return false in Phase 1")
	}
}

func TestMockProvider_Stream_NotImplemented(t *testing.T) {
	provider := NewMockProvider("test", []string{"response"})

	req := &CompletionRequest{
		Model:    "test-model",
		Messages: []Message{{Role: RoleUser, Content: "test"}},
	}

	fn := func(chunk string) error {
		return nil
	}

	// Phase 1: Streaming not implemented
	err := provider.Stream(context.Background(), req, fn)
	if err == nil {
		t.Error("Stream() should return error (not implemented)")
	}
}
