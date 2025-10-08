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
	"strings"
	"unicode"
)

// TokenCounter provides token counting functionality.
type TokenCounter interface {
	// CountTokens estimates the number of tokens in text.
	CountTokens(text string) int

	// CountMessagesTokens estimates tokens for a list of messages.
	CountMessagesTokens(messages []Message) int
}

// SimpleTokenCounter is a simple token counter using approximation.
// For production, consider using tiktoken or similar libraries.
type SimpleTokenCounter struct {
	// TokensPerWord is the average tokens per word (default: 1.3).
	TokensPerWord float64
}

// NewSimpleTokenCounter creates a new simple token counter.
func NewSimpleTokenCounter() *SimpleTokenCounter {
	return &SimpleTokenCounter{
		TokensPerWord: 1.3, // Average for English text
	}
}

// CountTokens estimates the number of tokens in text.
func (tc *SimpleTokenCounter) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	// Count words
	words := 0
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			if inWord {
				words++
				inWord = false
			}
		} else {
			inWord = true
		}
	}

	// Count last word if text doesn't end with space
	if inWord {
		words++
	}

	// Estimate tokens
	tokens := float64(words) * tc.TokensPerWord
	return int(tokens + 0.5) // Round to nearest integer
}

// CountMessagesTokens estimates tokens for a list of messages.
func (tc *SimpleTokenCounter) CountMessagesTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		// Add message content tokens
		total += tc.CountTokens(msg.Content)

		// Add overhead for message structure (role, formatting, etc.)
		// OpenAI uses about 4 tokens per message for formatting
		total += 4
	}

	// Add base overhead for conversation
	total += 2

	return total
}

// CharacterBasedTokenCounter counts tokens based on character count.
// Useful for approximate estimation without word boundaries.
type CharacterBasedTokenCounter struct {
	// CharsPerToken is the average characters per token (default: 4).
	CharsPerToken float64
}

// NewCharacterBasedTokenCounter creates a new character-based token counter.
func NewCharacterBasedTokenCounter() *CharacterBasedTokenCounter {
	return &CharacterBasedTokenCounter{
		CharsPerToken: 4.0, // Average for English text
	}
}

// CountTokens estimates the number of tokens in text.
func (tc *CharacterBasedTokenCounter) CountTokens(text string) int {
	if text == "" {
		return 0
	}

	chars := len([]rune(text)) // Count Unicode characters
	tokens := float64(chars) / tc.CharsPerToken
	result := int(tokens + 0.5) // Round to nearest integer
	if result == 0 && chars > 0 {
		return 1 // Minimum 1 token for non-empty text
	}
	return result
}

// CountMessagesTokens estimates tokens for a list of messages.
func (tc *CharacterBasedTokenCounter) CountMessagesTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += tc.CountTokens(msg.Content)
		total += 4 // Message overhead
	}
	total += 2 // Base overhead
	return total
}

// ModelTokenLimits contains token limits for different models.
var ModelTokenLimits = map[string]int{
	// OpenAI models
	"gpt-4":                  8192,
	"gpt-4-32k":              32768,
	"gpt-4-turbo":            128000,
	"gpt-4-turbo-preview":    128000,
	"gpt-3.5-turbo":          4096,
	"gpt-3.5-turbo-16k":      16384,
	"gpt-3.5-turbo-1106":     16385,

	// Anthropic Claude models
	"claude-3-opus":          200000,
	"claude-3-sonnet":        200000,
	"claude-3-haiku":         200000,
	"claude-2.1":             200000,
	"claude-2":               100000,
	"claude-instant-1.2":     100000,

	// Google models
	"gemini-pro":             32760,
	"gemini-1.5-pro":         1048576, // 1M tokens
	"gemini-1.5-flash":       1048576,
}

// GetModelTokenLimit returns the token limit for a model.
func GetModelTokenLimit(model string) int {
	// Try exact match
	if limit, ok := ModelTokenLimits[model]; ok {
		return limit
	}

	// Try prefix match for versioned models
	for modelPrefix, limit := range ModelTokenLimits {
		if strings.HasPrefix(model, modelPrefix) {
			return limit
		}
	}

	// Default to conservative limit
	return 4096
}

// TokenBudget helps manage token usage within limits.
type TokenBudget struct {
	counter   TokenCounter
	maxTokens int
	used      int
}

// NewTokenBudget creates a new token budget.
func NewTokenBudget(counter TokenCounter, maxTokens int) *TokenBudget {
	return &TokenBudget{
		counter:   counter,
		maxTokens: maxTokens,
		used:      0,
	}
}

// CanAdd checks if text can be added without exceeding budget.
func (tb *TokenBudget) CanAdd(text string) bool {
	tokens := tb.counter.CountTokens(text)
	return tb.used+tokens <= tb.maxTokens
}

// Add adds text to the budget and returns the token count.
func (tb *TokenBudget) Add(text string) int {
	tokens := tb.counter.CountTokens(text)
	tb.used += tokens
	return tokens
}

// Remaining returns the remaining tokens in the budget.
func (tb *TokenBudget) Remaining() int {
	remaining := tb.maxTokens - tb.used
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Used returns the number of tokens used.
func (tb *TokenBudget) Used() int {
	return tb.used
}

// Reset resets the token budget.
func (tb *TokenBudget) Reset() {
	tb.used = 0
}

// TruncateMessages truncates messages to fit within token limit.
func TruncateMessages(messages []Message, counter TokenCounter, maxTokens int) []Message {
	if len(messages) == 0 {
		return messages
	}

	// Always keep system message (first message) if present
	startIdx := 0
	systemTokens := 0
	if messages[0].Role == RoleSystem {
		systemTokens = counter.CountTokens(messages[0].Content) + 4
		startIdx = 1
	}

	// Calculate tokens for each message
	budget := maxTokens - systemTokens - 2 // Reserve for base overhead

	// Find which messages fit from newest to oldest
	toKeep := make([]Message, 0)
	used := 0
	for i := len(messages) - 1; i >= startIdx; i-- {
		msgTokens := counter.CountTokens(messages[i].Content) + 4
		if used+msgTokens > budget {
			break
		}
		used += msgTokens
		toKeep = append(toKeep, messages[i])
	}

	// Build result: system message (if exists) + messages in correct order
	result := make([]Message, 0, len(messages))
	if startIdx == 1 {
		result = append(result, messages[0]) // Keep system message
	}

	// Reverse toKeep to restore chronological order
	for i := len(toKeep) - 1; i >= 0; i-- {
		result = append(result, toKeep[i])
	}

	return result
}
