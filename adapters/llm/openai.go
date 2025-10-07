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
	"errors"
	"io"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements the Provider interface for OpenAI.
type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// OpenAIConfig contains OpenAI-specific configuration.
type OpenAIConfig struct {
	// APIKey is the OpenAI API key.
	// If empty, uses OPENAI_API_KEY environment variable.
	APIKey string

	// Model is the model to use (e.g., "gpt-4", "gpt-3.5-turbo").
	// Default: "gpt-4"
	Model string

	// BaseURL is the API base URL (for custom endpoints).
	// Default: https://api.openai.com/v1
	BaseURL string
}

// OpenAI creates a new OpenAI provider with optional configuration.
//
// If no config is provided, uses environment variables:
//   - OPENAI_API_KEY: API key (required)
//   - OPENAI_MODEL: Model name (optional, default: gpt-4)
//
// Example:
//
//	// From environment
//	provider := llm.OpenAI()
//
//	// With explicit config
//	provider := llm.OpenAI(&llm.OpenAIConfig{
//	    APIKey: "sk-...",
//	    Model:  "gpt-4",
//	})
func OpenAI(config ...*OpenAIConfig) Provider {
	var cfg *OpenAIConfig
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	} else {
		cfg = &OpenAIConfig{}
	}

	// Get API key from config or environment
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	// Get model from config or environment
	model := cfg.Model
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
	}
	if model == "" {
		model = "gpt-4" // Default model
	}

	// Create OpenAI client
	clientConfig := openai.DefaultConfig(apiKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAIProvider{
		client: client,
		model:  model,
	}
}

// Name returns the provider name.
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Complete generates a completion for the given request.
func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if req == nil {
		return nil, errors.New("completion request is nil")
	}

	// Convert messages to OpenAI format
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Determine model
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Create request
	chatReq := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}

	// Set optional parameters
	if req.MaxTokens > 0 {
		chatReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		chatReq.Temperature = float32(req.Temperature)
	}
	if req.TopP > 0 {
		chatReq.TopP = float32(req.TopP)
	}

	// Call OpenAI API
	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, convertOpenAIError(err)
	}

	// Check if we got choices
	if len(resp.Choices) == 0 {
		return nil, errors.New("no completion choices returned")
	}

	// Convert response
	choice := resp.Choices[0]
	return &CompletionResponse{
		ID:           resp.ID,
		Model:        resp.Model,
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
		Usage: &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// Stream generates a streaming completion.
func (p *OpenAIProvider) Stream(ctx context.Context, req *CompletionRequest, fn StreamFunc) error {
	if req == nil {
		return errors.New("completion request is nil")
	}
	if fn == nil {
		return errors.New("stream function is nil")
	}

	// Convert messages to OpenAI format
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Determine model
	model := req.Model
	if model == "" {
		model = p.model
	}

	// Create request
	chatReq := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	// Set optional parameters
	if req.MaxTokens > 0 {
		chatReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		chatReq.Temperature = float32(req.Temperature)
	}
	if req.TopP > 0 {
		chatReq.TopP = float32(req.TopP)
	}

	// Create stream
	stream, err := p.client.CreateChatCompletionStream(ctx, chatReq)
	if err != nil {
		return convertOpenAIError(err)
	}
	defer stream.Close()

	// Process stream
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return convertOpenAIError(err)
		}

		// Check if we got choices
		if len(response.Choices) == 0 {
			continue
		}

		// Get delta content
		delta := response.Choices[0].Delta.Content

		// Call stream function with chunk
		if delta != "" {
			if err := fn(delta); err != nil {
				return err
			}
		}
	}

	return nil
}

// SupportsStreaming returns true (OpenAI supports streaming).
func (p *OpenAIProvider) SupportsStreaming() bool {
	return true
}

// convertOpenAIError converts OpenAI errors to user-friendly messages.
func convertOpenAIError(err error) error {
	if err == nil {
		return nil
	}

	// Check for specific error types
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.HTTPStatusCode {
		case 401:
			return errors.New("invalid API key")
		case 429:
			return errors.New("rate limit exceeded")
		case 500, 502, 503:
			return errors.New("OpenAI service unavailable")
		default:
			return errors.New("OpenAI API error: " + apiErr.Message)
		}
	}

	return err
}
