// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestLoadConfig_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentPath := filepath.Join(tempDir, "nonexistent.yaml")

	cfg, err := loadConfig(nonExistentPath)
	if err != nil {
		t.Fatalf("loadConfig should return default config when file not found, got error: %v", err)
	}

	if cfg == nil {
		t.Error("Expected default config, got nil")
	}

	// Verify it's the correct type
	var _ *config.Config = cfg
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create a minimal valid config file (without LLM to avoid API key requirement)
	configContent := `
agent:
  name: test-agent
  version: 1.0.0

storage:
  type: memory
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config, got nil")
	}

	if cfg.Agent.Name != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", cfg.Agent.Name)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.yaml")

	// Create an invalid YAML file
	invalidContent := "this is: not: valid: yaml::"
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = loadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

// Note: configureLLM and configureStorage tests are skipped because they
// require concrete *builder.Builder type and have external dependencies
// (environment variables, actual LLM providers). These are better tested
// in integration tests.

func TestDefaultMessageHandler(t *testing.T) {
	handler := defaultMessageHandler()
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	// Test with mock message context
	mockCtx := &mockMessageContext{
		text: "test message",
	}

	err := handler(nil, mockCtx)
	if err != nil {
		t.Errorf("Handler should not error with valid message: %v", err)
	}

	if !mockCtx.replyCalled {
		t.Error("Expected Reply to be called")
	}

	if !strings.Contains(mockCtx.replyText, "Echo:") {
		t.Errorf("Expected echo response, got: %s", mockCtx.replyText)
	}
}

func TestDefaultMessageHandler_EmptyMessage(t *testing.T) {
	handler := defaultMessageHandler()

	mockCtx := &mockMessageContext{
		text: "",
	}

	err := handler(nil, mockCtx)
	if err == nil {
		t.Error("Expected error for empty message")
	}

	if !strings.Contains(err.Error(), "empty message") {
		t.Errorf("Expected 'empty message' error, got: %v", err)
	}
}

// Mock types for testing

type mockMessageContext struct {
	text        string
	replyCalled bool
	replyText   string
}

func (m *mockMessageContext) Text() string {
	return m.text
}

func (m *mockMessageContext) Parts() []types.Part {
	return nil
}

func (m *mockMessageContext) ContextID() string {
	return "test-context-id"
}

func (m *mockMessageContext) MessageID() string {
	return "test-msg-id"
}

func (m *mockMessageContext) Reply(text string) error {
	m.replyCalled = true
	m.replyText = text
	return nil
}

func (m *mockMessageContext) ReplyWithParts(parts []types.Part) error {
	return nil
}
