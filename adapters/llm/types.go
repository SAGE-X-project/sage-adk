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
)

// MessageRole represents the role of a message sender.
type MessageRole string

const (
	// RoleUser indicates a message from the user.
	RoleUser MessageRole = "user"

	// RoleAssistant indicates a message from the AI assistant.
	RoleAssistant MessageRole = "assistant"

	// RoleSystem indicates a system message.
	RoleSystem MessageRole = "system"
)

// Message represents a single message in a conversation.
type Message struct {
	// Role is the sender of the message.
	Role MessageRole `json:"role"`

	// Content is the message content.
	Content string `json:"content"`
}

// CompletionRequest represents a request to an LLM provider.
type CompletionRequest struct {
	// Model is the model name to use.
	Model string `json:"model"`

	// Messages is the conversation history.
	Messages []Message `json:"messages"`

	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0 to 2.0).
	Temperature float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling (0.0 to 1.0).
	TopP float64 `json:"top_p,omitempty"`

	// Stream enables streaming responses.
	Stream bool `json:"stream,omitempty"`

	// Metadata contains provider-specific metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CompletionResponse represents a response from an LLM provider.
type CompletionResponse struct {
	// ID is the unique response identifier.
	ID string `json:"id"`

	// Model is the model that generated the response.
	Model string `json:"model"`

	// Content is the generated content.
	Content string `json:"content"`

	// FinishReason indicates why generation stopped.
	FinishReason string `json:"finish_reason"`

	// Usage contains token usage information.
	Usage *Usage `json:"usage,omitempty"`

	// Metadata contains provider-specific metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	// PromptTokens is the number of tokens in the prompt.
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the completion.
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total number of tokens used.
	TotalTokens int `json:"total_tokens"`
}

// StreamFunc is a callback function for streaming responses.
// It receives chunks of the response as they arrive.
type StreamFunc func(chunk string) error

// Provider defines the interface for LLM providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// Complete generates a completion for the given request.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Stream generates a streaming completion.
	Stream(ctx context.Context, req *CompletionRequest, fn StreamFunc) error

	// SupportsStreaming returns true if the provider supports streaming.
	SupportsStreaming() bool
}

// AdvancedProvider extends Provider with advanced features.
type AdvancedProvider interface {
	Provider

	// SupportsFunctionCalling returns true if the provider supports function calling.
	SupportsFunctionCalling() bool

	// CompleteWithTools generates a completion with tool/function calling support.
	CompleteWithTools(ctx context.Context, req *CompletionRequestWithTools) (*CompletionResponseWithTools, error)

	// CountTokens estimates the number of tokens in text.
	CountTokens(text string) int

	// GetTokenLimit returns the maximum token limit for the model.
	GetTokenLimit(model string) int
}
