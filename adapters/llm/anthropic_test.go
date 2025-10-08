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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAnthropic_Name(t *testing.T) {
	provider := Anthropic(&AnthropicConfig{
		APIKey: "test-key",
	})

	if provider.Name() != "anthropic" {
		t.Errorf("Name() = %v, want anthropic", provider.Name())
	}
}

func TestAnthropic_SupportsStreaming(t *testing.T) {
	provider := Anthropic(&AnthropicConfig{
		APIKey: "test-key",
	})

	if !provider.SupportsStreaming() {
		t.Error("SupportsStreaming() = false, want true")
	}
}

func TestAnthropic_Complete_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key header = %v, want test-key", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != anthropicAPIVersion {
			t.Errorf("anthropic-version header = %v, want %v", r.Header.Get("anthropic-version"), anthropicAPIVersion)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type header = %v, want application/json", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var req anthropicRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Model != "claude-3-sonnet-20240229" {
			t.Errorf("Model = %v, want claude-3-sonnet-20240229", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Errorf("Messages length = %v, want 1", len(req.Messages))
		}
		if req.Messages[0].Role != "user" {
			t.Errorf("Message role = %v, want user", req.Messages[0].Role)
		}
		if req.Messages[0].Content != "Hello" {
			t.Errorf("Message content = %v, want Hello", req.Messages[0].Content)
		}

		// Send mock response
		resp := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []anthropicContent{
				{Type: "text", Text: "Hello! How can I help you?"},
			},
			Model:      "claude-3-sonnet-20240229",
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server
	provider := &AnthropicProvider{
		apiKey:     "test-key",
		model:      "claude-3-sonnet-20240229",
		httpClient: server.Client(),
	}

	// Note: We use the mock server URL directly in the HTTP request
	// instead of trying to override the const anthropicAPIURL

	// Make request (we'll use a workaround for testing)
	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	// Build request
	anthropicReq := provider.buildAnthropicRequest(req, false)

	// Make request to mock server
	reqBody, _ := json.Marshal(anthropicReq)
	httpReq, _ := http.NewRequest("POST", server.URL, strings.NewReader(string(reqBody)))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", "test-key")
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	httpResp, err := provider.httpClient.Do(httpReq)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer httpResp.Body.Close()

	// Parse response
	var anthropicResp anthropicResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&anthropicResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	resp := provider.convertResponse(&anthropicResp)

	// Verify response
	if resp.ID != "msg_123" {
		t.Errorf("ID = %v, want msg_123", resp.ID)
	}
	if resp.Model != "claude-3-sonnet-20240229" {
		t.Errorf("Model = %v, want claude-3-sonnet-20240229", resp.Model)
	}
	if resp.Content != "Hello! How can I help you?" {
		t.Errorf("Content = %v, want Hello! How can I help you?", resp.Content)
	}
	if resp.FinishReason != "end_turn" {
		t.Errorf("FinishReason = %v, want end_turn", resp.FinishReason)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %v, want 10", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %v, want 20", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %v, want 30", resp.Usage.TotalTokens)
	}
}

func TestAnthropic_Complete_WithSystemMessage(t *testing.T) {
	provider := &AnthropicProvider{
		apiKey: "test-key",
		model:  "claude-3-sonnet-20240229",
	}

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "You are a helpful assistant."},
			{Role: RoleUser, Content: "Hello"},
		},
	}

	anthropicReq := provider.buildAnthropicRequest(req, false)

	// Verify system message is separated
	if anthropicReq.System != "You are a helpful assistant." {
		t.Errorf("System = %v, want You are a helpful assistant.", anthropicReq.System)
	}

	// Verify only user message in messages
	if len(anthropicReq.Messages) != 1 {
		t.Errorf("Messages length = %v, want 1", len(anthropicReq.Messages))
	}
	if anthropicReq.Messages[0].Role != "user" {
		t.Errorf("Message role = %v, want user", anthropicReq.Messages[0].Role)
	}
}

func TestAnthropic_Complete_WithTemperature(t *testing.T) {
	provider := &AnthropicProvider{
		apiKey: "test-key",
		model:  "claude-3-sonnet-20240229",
	}

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		Temperature: 0.8,
		TopP:        0.9,
		MaxTokens:   1000,
	}

	anthropicReq := provider.buildAnthropicRequest(req, false)

	if anthropicReq.Temperature != 0.8 {
		t.Errorf("Temperature = %v, want 0.8", anthropicReq.Temperature)
	}
	if anthropicReq.TopP != 0.9 {
		t.Errorf("TopP = %v, want 0.9", anthropicReq.TopP)
	}
	if anthropicReq.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %v, want 1000", anthropicReq.MaxTokens)
	}
}

func TestAnthropic_Complete_DefaultMaxTokens(t *testing.T) {
	provider := &AnthropicProvider{
		apiKey: "test-key",
		model:  "claude-3-sonnet-20240229",
	}

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	anthropicReq := provider.buildAnthropicRequest(req, false)

	// Anthropic requires max_tokens, should default to 4096
	if anthropicReq.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %v, want 4096 (default)", anthropicReq.MaxTokens)
	}
}

func TestAnthropic_Complete_NilRequest(t *testing.T) {
	provider := Anthropic(&AnthropicConfig{
		APIKey: "test-key",
	})

	_, err := provider.Complete(context.Background(), nil)
	if err == nil {
		t.Error("Complete() with nil request should return error")
	}
}

func TestAnthropic_Stream_NilRequest(t *testing.T) {
	provider := Anthropic(&AnthropicConfig{
		APIKey: "test-key",
	})

	err := provider.Stream(context.Background(), nil, func(chunk string) error {
		return nil
	})
	if err == nil {
		t.Error("Stream() with nil request should return error")
	}
}

func TestAnthropic_Stream_NilStreamFunc(t *testing.T) {
	provider := Anthropic(&AnthropicConfig{
		APIKey: "test-key",
	})

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	err := provider.Stream(context.Background(), req, nil)
	if err == nil {
		t.Error("Stream() with nil stream function should return error")
	}
}

func TestAnthropic_ConvertError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		wantErr    string
	}{
		{
			name:       "401 Unauthorized",
			statusCode: 401,
			body:       []byte(`{"error":{"type":"authentication_error","message":"Invalid API key"}}`),
			wantErr:    "invalid API key",
		},
		{
			name:       "429 Rate Limit",
			statusCode: 429,
			body:       []byte(`{"error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`),
			wantErr:    "rate limit exceeded",
		},
		{
			name:       "500 Server Error",
			statusCode: 500,
			body:       []byte(`{"error":{"type":"api_error","message":"Internal server error"}}`),
			wantErr:    "Anthropic service unavailable",
		},
		{
			name:       "400 Bad Request",
			statusCode: 400,
			body:       []byte(`{"error":{"type":"invalid_request_error","message":"Invalid model"}}`),
			wantErr:    "Invalid model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := convertAnthropicError(tt.statusCode, tt.body)
			if err == nil {
				t.Fatal("convertAnthropicError() should return error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error = %v, want to contain %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestAnthropic_FromEnvironment(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	origModel := os.Getenv("ANTHROPIC_MODEL")
	defer func() {
		os.Setenv("ANTHROPIC_API_KEY", origAPIKey)
		os.Setenv("ANTHROPIC_MODEL", origModel)
	}()

	// Set test env vars
	os.Setenv("ANTHROPIC_API_KEY", "env-test-key")
	os.Setenv("ANTHROPIC_MODEL", "claude-3-opus-20240229")

	provider := Anthropic()

	p, ok := provider.(*AnthropicProvider)
	if !ok {
		t.Fatal("Provider is not *AnthropicProvider")
	}

	if p.apiKey != "env-test-key" {
		t.Errorf("apiKey = %v, want env-test-key", p.apiKey)
	}
	if p.model != "claude-3-opus-20240229" {
		t.Errorf("model = %v, want claude-3-opus-20240229", p.model)
	}
}

func TestAnthropic_ConfigOverridesEnvironment(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	origModel := os.Getenv("ANTHROPIC_MODEL")
	defer func() {
		os.Setenv("ANTHROPIC_API_KEY", origAPIKey)
		os.Setenv("ANTHROPIC_MODEL", origModel)
	}()

	// Set env vars
	os.Setenv("ANTHROPIC_API_KEY", "env-key")
	os.Setenv("ANTHROPIC_MODEL", "env-model")

	// Config should override env vars
	provider := Anthropic(&AnthropicConfig{
		APIKey: "config-key",
		Model:  "config-model",
	})

	p, ok := provider.(*AnthropicProvider)
	if !ok {
		t.Fatal("Provider is not *AnthropicProvider")
	}

	if p.apiKey != "config-key" {
		t.Errorf("apiKey = %v, want config-key", p.apiKey)
	}
	if p.model != "config-model" {
		t.Errorf("model = %v, want config-model", p.model)
	}
}

func TestAnthropic_DefaultModel(t *testing.T) {
	provider := Anthropic(&AnthropicConfig{
		APIKey: "test-key",
	})

	p, ok := provider.(*AnthropicProvider)
	if !ok {
		t.Fatal("Provider is not *AnthropicProvider")
	}

	if p.model != "claude-3-sonnet-20240229" {
		t.Errorf("model = %v, want claude-3-sonnet-20240229 (default)", p.model)
	}
}
