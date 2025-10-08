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

func TestSimpleTokenCounter_CountTokens(t *testing.T) {
	counter := NewSimpleTokenCounter()

	tests := []struct {
		text     string
		minCount int
		maxCount int
	}{
		{"Hello world", 2, 4},
		{"This is a test", 4, 6},
		{"", 0, 0},
		{"One", 1, 2},
	}

	for _, tt := range tests {
		count := counter.CountTokens(tt.text)
		if count < tt.minCount || count > tt.maxCount {
			t.Errorf("CountTokens(%q) = %d, want between %d and %d",
				tt.text, count, tt.minCount, tt.maxCount)
		}
	}
}

func TestSimpleTokenCounter_CountMessagesTokens(t *testing.T) {
	counter := NewSimpleTokenCounter()

	messages := []Message{
		{Role: RoleSystem, Content: "You are a helpful assistant."},
		{Role: RoleUser, Content: "Hello!"},
		{Role: RoleAssistant, Content: "Hi there!"},
	}

	count := counter.CountMessagesTokens(messages)
	if count < 10 {
		t.Errorf("CountMessagesTokens() = %d, want at least 10", count)
	}
}

func TestCharacterBasedTokenCounter_CountTokens(t *testing.T) {
	counter := NewCharacterBasedTokenCounter()

	tests := []struct {
		text     string
		minCount int
		maxCount int
	}{
		{"Hello", 1, 2},
		{"Hello world", 2, 4},
		{"", 0, 0},
		{"A", 1, 1},
	}

	for _, tt := range tests {
		count := counter.CountTokens(tt.text)
		if count < tt.minCount || count > tt.maxCount {
			t.Errorf("CountTokens(%q) = %d, want between %d and %d",
				tt.text, count, tt.minCount, tt.maxCount)
		}
	}
}

func TestGetModelTokenLimit(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		{"gpt-4", 8192},
		{"gpt-4-turbo", 128000},
		{"gpt-3.5-turbo", 4096},
		{"claude-3-opus", 200000},
		{"gemini-1.5-pro", 1048576},
		{"unknown-model", 4096}, // default
	}

	for _, tt := range tests {
		limit := GetModelTokenLimit(tt.model)
		if limit != tt.expected {
			t.Errorf("GetModelTokenLimit(%q) = %d, want %d",
				tt.model, limit, tt.expected)
		}
	}
}

func TestGetModelTokenLimit_PrefixMatch(t *testing.T) {
	// Should match by prefix
	limit := GetModelTokenLimit("gpt-4-0613")
	if limit != 8192 {
		t.Errorf("GetModelTokenLimit(gpt-4-0613) = %d, want 8192", limit)
	}
}

func TestTokenBudget_CanAdd(t *testing.T) {
	counter := NewSimpleTokenCounter()
	budget := NewTokenBudget(counter, 10)

	if !budget.CanAdd("Hello") {
		t.Error("CanAdd(Hello) should return true")
	}

	budget.Add("Hello world test")

	if budget.CanAdd("This is a long text") {
		t.Error("CanAdd should return false when budget exceeded")
	}
}

func TestTokenBudget_Add(t *testing.T) {
	counter := NewSimpleTokenCounter()
	budget := NewTokenBudget(counter, 100)

	tokens := budget.Add("Hello world")
	if tokens < 1 {
		t.Errorf("Add() returned %d, want at least 1", tokens)
	}

	if budget.Used() < 1 {
		t.Errorf("Used() = %d, want at least 1", budget.Used())
	}
}

func TestTokenBudget_Remaining(t *testing.T) {
	counter := NewSimpleTokenCounter()
	budget := NewTokenBudget(counter, 100)

	initial := budget.Remaining()
	if initial != 100 {
		t.Errorf("Remaining() = %d, want 100", initial)
	}

	budget.Add("Hello world")

	remaining := budget.Remaining()
	if remaining >= 100 {
		t.Errorf("Remaining() = %d, want less than 100", remaining)
	}
}

func TestTokenBudget_Reset(t *testing.T) {
	counter := NewSimpleTokenCounter()
	budget := NewTokenBudget(counter, 100)

	budget.Add("Hello world")
	if budget.Used() == 0 {
		t.Error("Used() should be > 0 after Add()")
	}

	budget.Reset()
	if budget.Used() != 0 {
		t.Errorf("Used() = %d after Reset(), want 0", budget.Used())
	}
}

func TestTruncateMessages_EmptyMessages(t *testing.T) {
	counter := NewSimpleTokenCounter()
	messages := []Message{}

	result := TruncateMessages(messages, counter, 100)
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestTruncateMessages_KeepSystemMessage(t *testing.T) {
	counter := NewSimpleTokenCounter()
	messages := []Message{
		{Role: RoleSystem, Content: "You are a helpful assistant."},
		{Role: RoleUser, Content: "Hello!"},
		{Role: RoleAssistant, Content: "Hi!"},
	}

	result := TruncateMessages(messages, counter, 10)

	if len(result) == 0 {
		t.Fatal("result should not be empty")
	}

	// System message should always be kept
	if result[0].Role != RoleSystem {
		t.Error("First message should be system message")
	}
}

func TestTruncateMessages_KeepRecentMessages(t *testing.T) {
	counter := NewSimpleTokenCounter()
	messages := []Message{
		{Role: RoleSystem, Content: "System."},
		{Role: RoleUser, Content: "Message 1"},
		{Role: RoleAssistant, Content: "Reply 1"},
		{Role: RoleUser, Content: "Message 2"},
		{Role: RoleAssistant, Content: "Reply 2"},
		{Role: RoleUser, Content: "Message 3"},
	}

	// Small token limit - should only keep system + most recent
	result := TruncateMessages(messages, counter, 15)

	if len(result) == 0 {
		t.Fatal("result should not be empty")
	}

	// Should have system message + some recent messages
	if result[0].Role != RoleSystem {
		t.Error("First message should be system message")
	}

	// Last message should be the most recent user message
	lastMsg := result[len(result)-1]
	if lastMsg.Content != "Message 3" {
		t.Errorf("Last message = %q, want Message 3", lastMsg.Content)
	}
}

func TestTruncateMessages_NoSystemMessage(t *testing.T) {
	counter := NewSimpleTokenCounter()
	messages := []Message{
		{Role: RoleUser, Content: "Message 1"},
		{Role: RoleAssistant, Content: "Reply 1"},
		{Role: RoleUser, Content: "Message 2"},
	}

	result := TruncateMessages(messages, counter, 10)

	if len(result) == 0 {
		t.Fatal("result should not be empty")
	}

	// Should keep most recent messages
	if result[len(result)-1].Content != "Message 2" {
		t.Error("Should keep most recent message")
	}
}

func TestTokenBudget_Exceeding(t *testing.T) {
	counter := NewSimpleTokenCounter()
	budget := NewTokenBudget(counter, 5)

	// Add text that exceeds budget
	budget.Add("This is a very long text that exceeds the budget")

	// Remaining should return 0, not negative
	if budget.Remaining() != 0 {
		t.Errorf("Remaining() = %d, want 0", budget.Remaining())
	}
}
