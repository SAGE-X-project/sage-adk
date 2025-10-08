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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGemini_Name(t *testing.T) {
	provider := Gemini(&GeminiConfig{
		APIKey: "test-key",
	})

	if provider.Name() != "gemini" {
		t.Errorf("Name() = %v, want gemini", provider.Name())
	}
}

func TestGemini_SupportsStreaming(t *testing.T) {
	provider := Gemini(&GeminiConfig{
		APIKey: "test-key",
	})

	if !provider.SupportsStreaming() {
		t.Error("SupportsStreaming() = false, want true")
	}
}

func TestGemini_BuildRequest(t *testing.T) {
	provider := &GeminiProvider{
		apiKey: "test-key",
		model:  "gemini-pro",
	}

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	geminiReq := provider.buildGeminiRequest(req)

	// Verify contents
	if len(geminiReq.Contents) != 1 {
		t.Errorf("Contents length = %v, want 1", len(geminiReq.Contents))
	}
	if geminiReq.Contents[0].Role != "user" {
		t.Errorf("Content role = %v, want user", geminiReq.Contents[0].Role)
	}
	if len(geminiReq.Contents[0].Parts) != 1 {
		t.Errorf("Parts length = %v, want 1", len(geminiReq.Contents[0].Parts))
	}
	if text, ok := geminiReq.Contents[0].Parts[0]["text"].(string); !ok || text != "Hello" {
		t.Errorf("Part text = %v, want Hello", geminiReq.Contents[0].Parts[0]["text"])
	}
}

func TestGemini_BuildRequest_WithSystemMessage(t *testing.T) {
	provider := &GeminiProvider{
		apiKey: "test-key",
		model:  "gemini-pro",
	}

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "You are a helpful assistant."},
			{Role: RoleUser, Content: "Hello"},
		},
	}

	geminiReq := provider.buildGeminiRequest(req)

	// Verify system instruction is separated
	if geminiReq.SystemInstruction == nil {
		t.Fatal("SystemInstruction is nil")
	}
	if len(geminiReq.SystemInstruction.Parts) != 1 {
		t.Errorf("SystemInstruction parts length = %v, want 1", len(geminiReq.SystemInstruction.Parts))
	}
	if text, ok := geminiReq.SystemInstruction.Parts[0]["text"].(string); !ok || text != "You are a helpful assistant." {
		t.Errorf("SystemInstruction text = %v, want You are a helpful assistant.", geminiReq.SystemInstruction.Parts[0]["text"])
	}

	// Verify only user message in contents
	if len(geminiReq.Contents) != 1 {
		t.Errorf("Contents length = %v, want 1", len(geminiReq.Contents))
	}
	if geminiReq.Contents[0].Role != "user" {
		t.Errorf("Content role = %v, want user", geminiReq.Contents[0].Role)
	}
}

func TestGemini_BuildRequest_AssistantRole(t *testing.T) {
	provider := &GeminiProvider{
		apiKey: "test-key",
		model:  "gemini-pro",
	}

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
			{Role: RoleAssistant, Content: "Hi there!"},
			{Role: RoleUser, Content: "How are you?"},
		},
	}

	geminiReq := provider.buildGeminiRequest(req)

	// Verify assistant role is converted to "model"
	if len(geminiReq.Contents) != 3 {
		t.Errorf("Contents length = %v, want 3", len(geminiReq.Contents))
	}
	if geminiReq.Contents[0].Role != "user" {
		t.Errorf("Contents[0] role = %v, want user", geminiReq.Contents[0].Role)
	}
	if geminiReq.Contents[1].Role != "model" {
		t.Errorf("Contents[1] role = %v, want model", geminiReq.Contents[1].Role)
	}
	if geminiReq.Contents[2].Role != "user" {
		t.Errorf("Contents[2] role = %v, want user", geminiReq.Contents[2].Role)
	}
}

func TestGemini_BuildRequest_WithParameters(t *testing.T) {
	provider := &GeminiProvider{
		apiKey: "test-key",
		model:  "gemini-pro",
	}

	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		Temperature: 0.8,
		TopP:        0.9,
		MaxTokens:   1000,
	}

	geminiReq := provider.buildGeminiRequest(req)

	if geminiReq.GenerationConfig == nil {
		t.Fatal("GenerationConfig is nil")
	}
	if geminiReq.GenerationConfig.Temperature != 0.8 {
		t.Errorf("Temperature = %v, want 0.8", geminiReq.GenerationConfig.Temperature)
	}
	if geminiReq.GenerationConfig.TopP != 0.9 {
		t.Errorf("TopP = %v, want 0.9", geminiReq.GenerationConfig.TopP)
	}
	if geminiReq.GenerationConfig.MaxOutputTokens != 1000 {
		t.Errorf("MaxOutputTokens = %v, want 1000", geminiReq.GenerationConfig.MaxOutputTokens)
	}
}

func TestGemini_ConvertResponse(t *testing.T) {
	provider := &GeminiProvider{
		apiKey: "test-key",
		model:  "gemini-pro",
	}

	geminiResp := &geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []map[string]interface{}{
						{"text": "Hello! How can I help you?"},
					},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: &geminiUsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 20,
			TotalTokenCount:      30,
		},
	}

	resp := provider.convertResponse(geminiResp)

	if resp.Content != "Hello! How can I help you?" {
		t.Errorf("Content = %v, want Hello! How can I help you?", resp.Content)
	}
	if resp.FinishReason != "STOP" {
		t.Errorf("FinishReason = %v, want STOP", resp.FinishReason)
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

func TestGemini_Complete_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL contains API key
		if !strings.Contains(r.URL.String(), "key=test-key") {
			t.Errorf("URL does not contain API key: %s", r.URL.String())
		}

		// Verify request body
		var req geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if len(req.Contents) != 1 {
			t.Errorf("Contents length = %v, want 1", len(req.Contents))
		}

		// Send mock response
		resp := geminiResponse{
			Candidates: []geminiCandidate{
				{
					Content: geminiContent{
						Parts: []map[string]interface{}{
							{"text": "Hello! How can I help you?"},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &geminiUsageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 20,
				TotalTokenCount:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with mock server
	provider := &GeminiProvider{
		apiKey:     "test-key",
		model:      "gemini-pro",
		httpClient: server.Client(),
	}

	// Build request
	req := &CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	geminiReq := provider.buildGeminiRequest(req)

	// Make request to mock server
	reqBody, _ := json.Marshal(geminiReq)
	url := server.URL + "?key=test-key"
	httpReq, _ := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := provider.httpClient.Do(httpReq)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer httpResp.Body.Close()

	// Parse response
	var geminiResp geminiResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&geminiResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	resp := provider.convertResponse(&geminiResp)

	// Verify response
	if resp.Content != "Hello! How can I help you?" {
		t.Errorf("Content = %v, want Hello! How can I help you?", resp.Content)
	}
	if resp.FinishReason != "STOP" {
		t.Errorf("FinishReason = %v, want STOP", resp.FinishReason)
	}
}

func TestGemini_Complete_NilRequest(t *testing.T) {
	provider := Gemini(&GeminiConfig{
		APIKey: "test-key",
	})

	_, err := provider.Complete(context.Background(), nil)
	if err == nil {
		t.Error("Complete() with nil request should return error")
	}
}

func TestGemini_Stream_NilRequest(t *testing.T) {
	provider := Gemini(&GeminiConfig{
		APIKey: "test-key",
	})

	err := provider.Stream(context.Background(), nil, func(chunk string) error {
		return nil
	})
	if err == nil {
		t.Error("Stream() with nil request should return error")
	}
}

func TestGemini_Stream_NilStreamFunc(t *testing.T) {
	provider := Gemini(&GeminiConfig{
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

func TestGemini_ConvertError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		wantErr    string
	}{
		{
			name:       "400 Invalid API Key",
			statusCode: 400,
			body:       []byte(`{"error":{"code":400,"message":"API key not valid","status":"INVALID_ARGUMENT"}}`),
			wantErr:    "invalid API key",
		},
		{
			name:       "403 Permission Denied",
			statusCode: 403,
			body:       []byte(`{"error":{"code":403,"message":"Permission denied","status":"PERMISSION_DENIED"}}`),
			wantErr:    "API key lacks permissions",
		},
		{
			name:       "429 Rate Limit",
			statusCode: 429,
			body:       []byte(`{"error":{"code":429,"message":"Rate limit exceeded","status":"RESOURCE_EXHAUSTED"}}`),
			wantErr:    "rate limit exceeded",
		},
		{
			name:       "500 Server Error",
			statusCode: 500,
			body:       []byte(`{"error":{"code":500,"message":"Internal server error","status":"INTERNAL"}}`),
			wantErr:    "Gemini service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := convertGeminiError(tt.statusCode, tt.body)
			if err == nil {
				t.Fatal("convertGeminiError() should return error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error = %v, want to contain %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestGemini_FromEnvironment(t *testing.T) {
	// Save original env vars
	origGeminiKey := os.Getenv("GEMINI_API_KEY")
	origGoogleKey := os.Getenv("GOOGLE_API_KEY")
	origModel := os.Getenv("GEMINI_MODEL")
	defer func() {
		os.Setenv("GEMINI_API_KEY", origGeminiKey)
		os.Setenv("GOOGLE_API_KEY", origGoogleKey)
		os.Setenv("GEMINI_MODEL", origModel)
	}()

	// Set test env vars
	os.Setenv("GEMINI_API_KEY", "env-test-key")
	os.Setenv("GEMINI_MODEL", "gemini-pro-vision")

	provider := Gemini()

	p, ok := provider.(*GeminiProvider)
	if !ok {
		t.Fatal("Provider is not *GeminiProvider")
	}

	if p.apiKey != "env-test-key" {
		t.Errorf("apiKey = %v, want env-test-key", p.apiKey)
	}
	if p.model != "gemini-pro-vision" {
		t.Errorf("model = %v, want gemini-pro-vision", p.model)
	}
}

func TestGemini_GoogleAPIKeyFallback(t *testing.T) {
	// Save original env vars
	origGeminiKey := os.Getenv("GEMINI_API_KEY")
	origGoogleKey := os.Getenv("GOOGLE_API_KEY")
	defer func() {
		os.Setenv("GEMINI_API_KEY", origGeminiKey)
		os.Setenv("GOOGLE_API_KEY", origGoogleKey)
	}()

	// Clear GEMINI_API_KEY and set GOOGLE_API_KEY
	os.Unsetenv("GEMINI_API_KEY")
	os.Setenv("GOOGLE_API_KEY", "google-key")

	provider := Gemini()

	p, ok := provider.(*GeminiProvider)
	if !ok {
		t.Fatal("Provider is not *GeminiProvider")
	}

	if p.apiKey != "google-key" {
		t.Errorf("apiKey = %v, want google-key (from GOOGLE_API_KEY)", p.apiKey)
	}
}

func TestGemini_ConfigOverridesEnvironment(t *testing.T) {
	// Save original env vars
	origKey := os.Getenv("GEMINI_API_KEY")
	origModel := os.Getenv("GEMINI_MODEL")
	defer func() {
		os.Setenv("GEMINI_API_KEY", origKey)
		os.Setenv("GEMINI_MODEL", origModel)
	}()

	// Set env vars
	os.Setenv("GEMINI_API_KEY", "env-key")
	os.Setenv("GEMINI_MODEL", "env-model")

	// Config should override env vars
	provider := Gemini(&GeminiConfig{
		APIKey: "config-key",
		Model:  "config-model",
	})

	p, ok := provider.(*GeminiProvider)
	if !ok {
		t.Fatal("Provider is not *GeminiProvider")
	}

	if p.apiKey != "config-key" {
		t.Errorf("apiKey = %v, want config-key", p.apiKey)
	}
	if p.model != "config-model" {
		t.Errorf("model = %v, want config-model", p.model)
	}
}

func TestGemini_DefaultModel(t *testing.T) {
	provider := Gemini(&GeminiConfig{
		APIKey: "test-key",
	})

	p, ok := provider.(*GeminiProvider)
	if !ok {
		t.Fatal("Provider is not *GeminiProvider")
	}

	if p.model != "gemini-pro" {
		t.Errorf("model = %v, want gemini-pro (default)", p.model)
	}
}
