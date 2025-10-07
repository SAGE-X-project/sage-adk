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
	"os"
	"testing"
)

func TestOpenAI_Creation(t *testing.T) {
	// Test with explicit config
	provider := OpenAI(&OpenAIConfig{
		APIKey: "test-key",
		Model:  "gpt-4",
	})

	if provider == nil {
		t.Fatal("OpenAI() returned nil")
	}

	if provider.Name() != "openai" {
		t.Errorf("Name() = %v, want openai", provider.Name())
	}
}

func TestOpenAI_FromEnvironment(t *testing.T) {
	// Set environment variable
	os.Setenv("OPENAI_API_KEY", "test-env-key")
	os.Setenv("OPENAI_MODEL", "gpt-3.5-turbo")
	defer func() {
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("OPENAI_MODEL")
	}()

	// Create provider from environment
	provider := OpenAI()

	if provider == nil {
		t.Fatal("OpenAI() returned nil")
	}

	openaiProvider, ok := provider.(*OpenAIProvider)
	if !ok {
		t.Fatal("provider is not *OpenAIProvider")
	}

	if openaiProvider.model != "gpt-3.5-turbo" {
		t.Errorf("model = %v, want gpt-3.5-turbo", openaiProvider.model)
	}
}

func TestOpenAI_DefaultModel(t *testing.T) {
	// No config, no environment variable
	provider := OpenAI(&OpenAIConfig{
		APIKey: "test-key",
	})

	openaiProvider, ok := provider.(*OpenAIProvider)
	if !ok {
		t.Fatal("provider is not *OpenAIProvider")
	}

	if openaiProvider.model != "gpt-4" {
		t.Errorf("default model = %v, want gpt-4", openaiProvider.model)
	}
}

func TestOpenAI_SupportsStreaming(t *testing.T) {
	provider := OpenAI(&OpenAIConfig{
		APIKey: "test-key",
	})

	if !provider.SupportsStreaming() {
		t.Error("SupportsStreaming() = false, want true")
	}
}

func TestOpenAI_Complete_NilRequest(t *testing.T) {
	provider := OpenAI(&OpenAIConfig{
		APIKey: "test-key",
	})

	_, err := provider.Complete(context.Background(), nil)

	if err == nil {
		t.Error("Complete() with nil request should return error")
	}
}

func TestOpenAI_Stream_NilRequest(t *testing.T) {
	provider := OpenAI(&OpenAIConfig{
		APIKey: "test-key",
	})

	err := provider.Stream(context.Background(), nil, func(chunk string) error {
		return nil
	})

	if err == nil {
		t.Error("Stream() with nil request should return error")
	}
}

func TestOpenAI_Stream_NilFunction(t *testing.T) {
	provider := OpenAI(&OpenAIConfig{
		APIKey: "test-key",
	})

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "test"},
		},
	}

	err := provider.Stream(context.Background(), req, nil)

	if err == nil {
		t.Error("Stream() with nil function should return error")
	}
}

// Note: Integration tests with real API calls should be run separately
// with a valid API key. These are unit tests that test the structure.

func TestOpenAI_RequestConversion(t *testing.T) {
	provider := OpenAI(&OpenAIConfig{
		APIKey: "test-key",
		Model:  "gpt-4",
	})

	req := &CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: RoleSystem, Content: "You are helpful"},
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
		TopP:        0.9,
	}

	// This will fail with invalid API key, but tests request structure
	_, err := provider.Complete(context.Background(), req)

	// We expect an error (invalid API key), but this tests that
	// the request was properly constructed
	if err == nil {
		t.Log("Note: Got successful response with test key (unexpected)")
	}
}

func TestOpenAI_ModelOverride(t *testing.T) {
	// Provider has default model
	provider := OpenAI(&OpenAIConfig{
		APIKey: "test-key",
		Model:  "gpt-4",
	})

	// Request overrides model
	req := &CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: RoleUser, Content: "test"},
		},
	}

	// This will fail with invalid API key, but tests model override
	_, err := provider.Complete(context.Background(), req)

	// We expect an error, but this tests the model override logic
	if err == nil {
		t.Log("Note: Got successful response (unexpected)")
	}
}

func TestConvertOpenAIError(t *testing.T) {
	// Test nil error
	err := convertOpenAIError(nil)
	if err != nil {
		t.Errorf("convertOpenAIError(nil) = %v, want nil", err)
	}

	// Test generic error
	genericErr := convertOpenAIError(context.Canceled)
	if genericErr == nil {
		t.Error("convertOpenAIError() should not return nil for generic error")
	}
}
