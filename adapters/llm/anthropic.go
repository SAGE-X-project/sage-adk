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
	anthropicAPIURL     = "https://api.anthropic.com/v1/messages"
	anthropicAPIVersion = "2023-06-01"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude.
type AnthropicProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// AnthropicConfig contains Anthropic-specific configuration.
type AnthropicConfig struct {
	// APIKey is the Anthropic API key.
	// If empty, uses ANTHROPIC_API_KEY environment variable.
	APIKey string

	// Model is the model to use (e.g., "claude-3-opus-20240229", "claude-3-sonnet-20240229").
	// Default: "claude-3-sonnet-20240229"
	Model string

	// HTTPClient is the HTTP client to use (optional).
	HTTPClient *http.Client
}

// Anthropic creates a new Anthropic provider with optional configuration.
//
// If no config is provided, uses environment variables:
//   - ANTHROPIC_API_KEY: API key (required)
//   - ANTHROPIC_MODEL: Model name (optional, default: claude-3-sonnet-20240229)
//
// Example:
//
//	// From environment
//	provider := llm.Anthropic()
//
//	// With explicit config
//	provider := llm.Anthropic(&llm.AnthropicConfig{
//	    APIKey: "sk-ant-...",
//	    Model:  "claude-3-opus-20240229",
//	})
func Anthropic(config ...*AnthropicConfig) Provider {
	var cfg *AnthropicConfig
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	} else {
		cfg = &AnthropicConfig{}
	}

	// Get API key from config or environment
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	// Get model from config or environment
	model := cfg.Model
	if model == "" {
		model = os.Getenv("ANTHROPIC_MODEL")
	}
	if model == "" {
		model = "claude-3-sonnet-20240229" // Default model
	}

	// Get HTTP client
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &AnthropicProvider{
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
	}
}

// Name returns the provider name.
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Complete generates a completion for the given request.
func (p *AnthropicProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if req == nil {
		return nil, errors.New("completion request is nil")
	}

	// Convert to Anthropic format
	anthropicReq := p.buildAnthropicRequest(req, false)

	// Make API request
	respBody, err := p.makeRequest(ctx, anthropicReq)
	if err != nil {
		return nil, err
	}

	// Parse response
	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to standard response
	return p.convertResponse(&anthropicResp), nil
}

// Stream generates a streaming completion.
func (p *AnthropicProvider) Stream(ctx context.Context, req *CompletionRequest, fn StreamFunc) error {
	if req == nil {
		return errors.New("completion request is nil")
	}
	if fn == nil {
		return errors.New("stream function is nil")
	}

	// Convert to Anthropic format with streaming enabled
	anthropicReq := p.buildAnthropicRequest(req, true)

	// Create HTTP request
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	// Make request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return convertAnthropicError(resp.StatusCode, body)
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

		// Check for stream end
		if data == "[DONE]" {
			break
		}

		// Parse JSON event
		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// Handle content block delta
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			if err := fn(event.Delta.Text); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream reading error: %w", err)
	}

	return nil
}

// SupportsStreaming returns true (Anthropic supports streaming).
func (p *AnthropicProvider) SupportsStreaming() bool {
	return true
}

// SupportsFunctionCalling returns true as Anthropic Claude supports tool use.
func (p *AnthropicProvider) SupportsFunctionCalling() bool {
	return true
}

// CompleteWithTools generates a completion with tool/function calling support.
func (p *AnthropicProvider) CompleteWithTools(ctx context.Context, req *CompletionRequestWithTools) (*CompletionResponseWithTools, error) {
	if req == nil {
		return nil, errors.New("completion request is nil")
	}

	// Build base request
	anthropicReq := p.buildAnthropicRequest(&req.CompletionRequest, false)

	// Add tools if provided
	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, len(req.Tools))
		for i, tool := range req.Tools {
			tools[i] = map[string]interface{}{
				"name":         tool.Function.Name,
				"description":  tool.Function.Description,
				"input_schema": tool.Function.Parameters,
			}
		}
		anthropicReq.Tools = tools
	}

	// Set tool choice if specified
	if req.ToolChoice != nil {
		anthropicReq.ToolChoice = req.ToolChoice
	}

	// Marshal request
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, convertAnthropicError(resp.StatusCode, body)
	}

	// Decode response
	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Build result
	result := &CompletionResponseWithTools{
		CompletionResponse: CompletionResponse{
			ID:           anthropicResp.ID,
			Model:        anthropicResp.Model,
			Content:      extractTextContent(anthropicResp.Content),
			FinishReason: anthropicResp.StopReason,
			Usage: &Usage{
				PromptTokens:     anthropicResp.Usage.InputTokens,
				CompletionTokens: anthropicResp.Usage.OutputTokens,
				TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
			},
		},
	}

	// Extract tool use from content
	for _, content := range anthropicResp.Content {
		if content.Type == "tool_use" {
			// Marshal input to JSON string
			argsJSON, err := json.Marshal(content.Input)
			if err != nil {
				continue
			}

			if result.ToolCalls == nil {
				result.ToolCalls = make([]*ToolCall, 0)
			}

			result.ToolCalls = append(result.ToolCalls, &ToolCall{
				ID:   content.ID,
				Type: ToolTypeFunction,
				Function: &FunctionCall{
					Name:      content.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	return result, nil
}

// CountTokens estimates the number of tokens in text.
func (p *AnthropicProvider) CountTokens(text string) int {
	counter := NewSimpleTokenCounter()
	return counter.CountTokens(text)
}

// GetTokenLimit returns the maximum token limit for the model.
func (p *AnthropicProvider) GetTokenLimit(model string) int {
	return GetModelTokenLimit(model)
}

// buildAnthropicRequest converts our standard request to Anthropic format.
func (p *AnthropicProvider) buildAnthropicRequest(req *CompletionRequest, stream bool) *anthropicRequest {
	// Separate system message from conversation
	var system string
	var messages []anthropicMessage

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			system = msg.Content
		} else {
			messages = append(messages, anthropicMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	// Determine model
	model := req.Model
	if model == "" {
		model = p.model
	}

	anthropicReq := &anthropicRequest{
		Model:    model,
		Messages: messages,
		Stream:   stream,
	}

	// Set system message if present
	if system != "" {
		anthropicReq.System = system
	}

	// Set optional parameters
	if req.MaxTokens > 0 {
		anthropicReq.MaxTokens = req.MaxTokens
	} else {
		anthropicReq.MaxTokens = 4096 // Anthropic requires max_tokens
	}

	if req.Temperature > 0 {
		anthropicReq.Temperature = req.Temperature
	}

	if req.TopP > 0 {
		anthropicReq.TopP = req.TopP
	}

	return anthropicReq
}

// makeRequest makes an HTTP request to Anthropic API.
func (p *AnthropicProvider) makeRequest(ctx context.Context, anthropicReq *anthropicRequest) ([]byte, error) {
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

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
		return nil, convertAnthropicError(resp.StatusCode, body)
	}

	return body, nil
}

// convertResponse converts Anthropic response to standard format.
func (p *AnthropicProvider) convertResponse(resp *anthropicResponse) *CompletionResponse {
	var content string
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	return &CompletionResponse{
		ID:           resp.ID,
		Model:        resp.Model,
		Content:      content,
		FinishReason: resp.StopReason,
		Usage: &Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// convertAnthropicError converts Anthropic API errors to user-friendly messages.
func convertAnthropicError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("API error (status %d)", statusCode)
	}

	switch statusCode {
	case 401:
		return errors.New("invalid API key")
	case 429:
		return errors.New("rate limit exceeded")
	case 500, 502, 503:
		return errors.New("Anthropic service unavailable")
	default:
		if errResp.Error.Message != "" {
			return fmt.Errorf("Anthropic API error: %s", errResp.Error.Message)
		}
		return fmt.Errorf("Anthropic API error (status %d)", statusCode)
	}
}

// Anthropic API request/response types

type anthropicRequest struct {
	Model       string                   `json:"model"`
	Messages    []anthropicMessage       `json:"messages"`
	System      string                   `json:"system,omitempty"`
	MaxTokens   int                      `json:"max_tokens"`
	Temperature float64                  `json:"temperature,omitempty"`
	TopP        float64                  `json:"top_p,omitempty"`
	Stream      bool                     `json:"stream,omitempty"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
	ToolChoice  interface{}              `json:"tool_choice,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID         string              `json:"id"`
	Type       string              `json:"type"`
	Role       string              `json:"role"`
	Content    []anthropicContent  `json:"content"`
	Model      string              `json:"model"`
	StopReason string              `json:"stop_reason"`
	Usage      anthropicUsage      `json:"usage"`
}

type anthropicContent struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// extractTextContent extracts text from content array.
func extractTextContent(contents []anthropicContent) string {
	for _, content := range contents {
		if content.Type == "text" && content.Text != "" {
			return content.Text
		}
	}
	return ""
}

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}
