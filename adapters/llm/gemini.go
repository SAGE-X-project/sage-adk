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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	geminiAPIURL = "https://generativelanguage.googleapis.com/v1beta/models"
)

// GeminiProvider implements the Provider interface for Google Gemini.
type GeminiProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// GeminiConfig contains Gemini-specific configuration.
type GeminiConfig struct {
	// APIKey is the Google AI API key.
	// If empty, uses GEMINI_API_KEY or GOOGLE_API_KEY environment variable.
	APIKey string

	// Model is the model to use (e.g., "gemini-pro", "gemini-pro-vision").
	// Default: "gemini-pro"
	Model string

	// HTTPClient is the HTTP client to use (optional).
	HTTPClient *http.Client
}

// Gemini creates a new Gemini provider with optional configuration.
//
// If no config is provided, uses environment variables:
//   - GEMINI_API_KEY or GOOGLE_API_KEY: API key (required)
//   - GEMINI_MODEL: Model name (optional, default: gemini-pro)
//
// Example:
//
//	// From environment
//	provider := llm.Gemini()
//
//	// With explicit config
//	provider := llm.Gemini(&llm.GeminiConfig{
//	    APIKey: "AIza...",
//	    Model:  "gemini-pro",
//	})
func Gemini(config ...*GeminiConfig) Provider {
	var cfg *GeminiConfig
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	} else {
		cfg = &GeminiConfig{}
	}

	// Get API key from config or environment
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}

	// Get model from config or environment
	model := cfg.Model
	if model == "" {
		model = os.Getenv("GEMINI_MODEL")
	}
	if model == "" {
		model = "gemini-pro" // Default model
	}

	// Get HTTP client
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &GeminiProvider{
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
	}
}

// Name returns the provider name.
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// Complete generates a completion for the given request.
func (p *GeminiProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if req == nil {
		return nil, errors.New("completion request is nil")
	}

	// Convert to Gemini format
	geminiReq := p.buildGeminiRequest(req)

	// Make API request
	respBody, err := p.makeRequest(ctx, geminiReq, false)
	if err != nil {
		return nil, err
	}

	// Parse response
	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to standard response
	return p.convertResponse(&geminiResp), nil
}

// Stream generates a streaming completion.
func (p *GeminiProvider) Stream(ctx context.Context, req *CompletionRequest, fn StreamFunc) error {
	if req == nil {
		return errors.New("completion request is nil")
	}
	if fn == nil {
		return errors.New("stream function is nil")
	}

	// Convert to Gemini format
	geminiReq := p.buildGeminiRequest(req)

	// Create HTTP request for streaming
	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	model := p.getModel(req)
	url := fmt.Sprintf("%s/%s:streamGenerateContent?key=%s&alt=sse", geminiAPIURL, model, p.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return convertGeminiError(resp.StatusCode, body)
	}

	// Process SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE event
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Parse JSON event
		var event geminiResponse
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// Extract text from candidates
		if len(event.Candidates) > 0 && len(event.Candidates[0].Content.Parts) > 0 {
			if text, ok := event.Candidates[0].Content.Parts[0]["text"].(string); ok && text != "" {
				if err := fn(text); err != nil {
					return err
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream reading error: %w", err)
	}

	return nil
}

// SupportsStreaming returns true (Gemini supports streaming).
func (p *GeminiProvider) SupportsStreaming() bool {
	return true
}

// SupportsFunctionCalling returns true as Gemini supports function calling.
func (p *GeminiProvider) SupportsFunctionCalling() bool {
	return true
}

// CompleteWithTools generates a completion with tool/function calling support.
func (p *GeminiProvider) CompleteWithTools(ctx context.Context, req *CompletionRequestWithTools) (*CompletionResponseWithTools, error) {
	if req == nil {
		return nil, errors.New("completion request is nil")
	}

	// Build base request
	geminiReq := p.buildGeminiRequest(&req.CompletionRequest)

	// Add tools if provided
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]interface{}{
				"function_declarations": []map[string]interface{}{
					{
						"name":        tool.Function.Name,
						"description": tool.Function.Description,
						"parameters":  tool.Function.Parameters,
					},
				},
			}
		}
		geminiReq.Tools = tools
	}

	// Determine model
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Build URL
	endpoint := fmt.Sprintf("/%s:generateContent", model)
	url := geminiAPIURL + endpoint + "?key=" + p.apiKey

	// Marshal request
	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, convertGeminiError(resp.StatusCode, body)
	}

	// Decode response
	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, errors.New("no candidates in response")
	}

	candidate := geminiResp.Candidates[0]

	// Build result
	result := &CompletionResponseWithTools{
		CompletionResponse: CompletionResponse{
			Model:        model,
			Content:      extractGeminiText(candidate.Content.Parts),
			FinishReason: candidate.FinishReason,
			Usage: &Usage{
				PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
				CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
			},
		},
	}

	// Extract function calls from parts
	for _, part := range candidate.Content.Parts {
		if fc, ok := part["functionCall"].(map[string]interface{}); ok {
			name, _ := fc["name"].(string)
			args := fc["args"]

			// Marshal args to JSON string
			argsJSON, err := json.Marshal(args)
			if err != nil {
				continue
			}

			if result.ToolCalls == nil {
				result.ToolCalls = make([]*ToolCall, 0)
			}

			result.ToolCalls = append(result.ToolCalls, &ToolCall{
				ID:   fmt.Sprintf("call_%s", name), // Gemini doesn't provide IDs
				Type: ToolTypeFunction,
				Function: &FunctionCall{
					Name:      name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	return result, nil
}

// CountTokens estimates the number of tokens in text.
func (p *GeminiProvider) CountTokens(text string) int {
	counter := NewSimpleTokenCounter()
	return counter.CountTokens(text)
}

// GetTokenLimit returns the maximum token limit for the model.
func (p *GeminiProvider) GetTokenLimit(model string) int {
	return GetModelTokenLimit(model)
}

// buildGeminiRequest converts our standard request to Gemini format.
func (p *GeminiProvider) buildGeminiRequest(req *CompletionRequest) *geminiRequest {
	var contents []geminiContent

	// Gemini uses a different format - it has role + parts structure
	// System messages are handled via systemInstruction field
	var systemInstruction *geminiContent

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			// System message goes to systemInstruction
			systemInstruction = &geminiContent{
				Parts: []map[string]interface{}{{"text": msg.Content}},
			}
		} else {
			// Convert role to Gemini format
			role := "user"
			if msg.Role == RoleAssistant {
				role = "model" // Gemini uses "model" instead of "assistant"
			}

			contents = append(contents, geminiContent{
				Role:  role,
				Parts: []map[string]interface{}{{"text": msg.Content}},
			})
		}
	}

	geminiReq := &geminiRequest{
		Contents: contents,
	}

	// Set system instruction if present
	if systemInstruction != nil {
		geminiReq.SystemInstruction = systemInstruction
	}

	// Set generation config
	genConfig := &geminiGenerationConfig{}

	if req.Temperature > 0 {
		genConfig.Temperature = req.Temperature
	}
	if req.TopP > 0 {
		genConfig.TopP = req.TopP
	}
	if req.MaxTokens > 0 {
		genConfig.MaxOutputTokens = req.MaxTokens
	}

	geminiReq.GenerationConfig = genConfig

	return geminiReq
}

// makeRequest makes an HTTP request to Gemini API.
func (p *GeminiProvider) makeRequest(ctx context.Context, geminiReq *geminiRequest, stream bool) ([]byte, error) {
	reqBody, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	model := p.model
	endpoint := "generateContent"
	if stream {
		endpoint = "streamGenerateContent"
	}

	url := fmt.Sprintf("%s/%s:%s?key=%s", geminiAPIURL, model, endpoint, p.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, convertGeminiError(resp.StatusCode, body)
	}

	return body, nil
}

// getModel returns the model to use for the request.
func (p *GeminiProvider) getModel(req *CompletionRequest) string {
	if req.Model != "" {
		return req.Model
	}
	return p.model
}

// convertResponse converts Gemini response to standard format.
func (p *GeminiProvider) convertResponse(resp *geminiResponse) *CompletionResponse {
	var content string
	var finishReason string

	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		content = extractGeminiText(candidate.Content.Parts)
		finishReason = candidate.FinishReason
	}

	// Calculate usage
	var usage *Usage
	if resp.UsageMetadata != nil {
		usage = &Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return &CompletionResponse{
		ID:           "", // Gemini doesn't provide an ID
		Model:        p.model,
		Content:      content,
		FinishReason: finishReason,
		Usage:        usage,
	}
}

// convertGeminiError converts Gemini API errors to user-friendly messages.
func convertGeminiError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("API error (status %d)", statusCode)
	}

	switch statusCode {
	case 400:
		if strings.Contains(errResp.Error.Message, "API key") {
			return errors.New("invalid API key")
		}
		return fmt.Errorf("invalid request: %s", errResp.Error.Message)
	case 403:
		return errors.New("API key lacks permissions or quota exceeded")
	case 429:
		return errors.New("rate limit exceeded")
	case 500, 502, 503:
		return errors.New("Gemini service unavailable")
	default:
		if errResp.Error.Message != "" {
			return fmt.Errorf("Gemini API error: %s", errResp.Error.Message)
		}
		return fmt.Errorf("Gemini API error (status %d)", statusCode)
	}
}

// Gemini API request/response types

type geminiRequest struct {
	Contents          []geminiContent          `json:"contents"`
	SystemInstruction *geminiContent           `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig  `json:"generationConfig,omitempty"`
	Tools             []map[string]interface{} `json:"tools,omitempty"`
}

type geminiContent struct {
	Role  string                   `json:"role,omitempty"`
	Parts []map[string]interface{} `json:"parts"`
}

type geminiPart map[string]interface{}

// extractGeminiText extracts text from parts array.
func extractGeminiText(parts []map[string]interface{}) string {
	for _, part := range parts {
		if text, ok := part["text"].(string); ok && text != "" {
			return text
		}
	}
	return ""
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate   `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}
