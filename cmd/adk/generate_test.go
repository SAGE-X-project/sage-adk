// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGenerate_Provider(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	err := runGenerate(nil, []string{"provider", "test"})
	if err != nil {
		t.Fatalf("runGenerate provider failed: %v", err)
	}

	// Verify provider file was created
	providerFile := "test_provider.go"
	if _, err := os.Stat(providerFile); os.IsNotExist(err) {
		t.Error("Provider file was not created")
	}
}

func TestRunGenerate_Middleware(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	err := runGenerate(nil, []string{"middleware", "auth"})
	if err != nil {
		t.Fatalf("runGenerate middleware failed: %v", err)
	}

	// Verify middleware file was created
	middlewareFile := "auth_middleware.go"
	if _, err := os.Stat(middlewareFile); os.IsNotExist(err) {
		t.Error("Middleware file was not created")
	}
}

func TestRunGenerate_Adapter(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	err := runGenerate(nil, []string{"adapter", "custom"})
	if err != nil {
		t.Fatalf("runGenerate adapter failed: %v", err)
	}

	// Verify adapter directory and file were created
	adapterDir := filepath.Join("adapters", "custom")
	adapterFile := filepath.Join(adapterDir, "adapter.go")

	if _, err := os.Stat(adapterFile); os.IsNotExist(err) {
		t.Error("Adapter file was not created")
	}
}

func TestRunGenerate_UnsupportedType(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	err := runGenerate(nil, []string{"unknown", "test"})
	if err == nil {
		t.Error("Expected error for unsupported type")
	}

	if !strings.Contains(err.Error(), "unknown generate type") {
		t.Errorf("Expected 'unknown generate type' error, got: %v", err)
	}
}

func TestGenerateProvider(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	name := "mytest"
	err := generateProvider(name)
	if err != nil {
		t.Fatalf("generateProvider failed: %v", err)
	}

	// Verify file was created
	filename := name + "_provider.go"
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read provider file: %v", err)
	}

	contentStr := string(content)

	// Verify expected content
	expectedPatterns := []string{
		"package llm",
		"type " + name + "Provider struct",
		"type " + name + "Config struct",
		"func New" + name,
		"func (p *" + name + "Provider) Name() string",
		"func (p *" + name + "Provider) Generate",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf("Expected pattern not found in provider: %s", pattern)
		}
	}
}

func TestGenerateMiddleware(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	name := "logging"
	err := generateMiddleware(name)
	if err != nil {
		t.Fatalf("generateMiddleware failed: %v", err)
	}

	// Verify file was created
	filename := name + "_middleware.go"
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read middleware file: %v", err)
	}

	contentStr := string(content)

	// Verify expected content
	expectedPatterns := []string{
		"package middleware",
		"func " + name,
		"return func(next Handler) Handler",
		"// TODO: Pre-processing logic here",
		"// TODO: Post-processing logic here",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf("Expected pattern not found in middleware: %s", pattern)
		}
	}
}

func TestGenerateAdapter(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	name := "custom"
	err := generateAdapter(name)
	if err != nil {
		t.Fatalf("generateAdapter failed: %v", err)
	}

	// Verify directory was created
	adapterDir := filepath.Join("adapters", name)
	if _, err := os.Stat(adapterDir); os.IsNotExist(err) {
		t.Fatal("Adapter directory was not created")
	}

	// Verify file was created
	adapterFile := filepath.Join(adapterDir, "adapter.go")
	content, err := os.ReadFile(adapterFile)
	if err != nil {
		t.Fatalf("Failed to read adapter file: %v", err)
	}

	contentStr := string(content)

	// Verify expected content
	expectedPatterns := []string{
		"package " + name,
		"type Adapter struct",
		"type Config struct",
		"func NewAdapter",
		"func (a *Adapter) Name() string",
		"func (a *Adapter) SendMessage",
		"func (a *Adapter) ReceiveMessage",
		"func (a *Adapter) Verify",
		"func (a *Adapter) SupportsStreaming",
		"func (a *Adapter) Stream",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf("Expected pattern not found in adapter: %s", pattern)
		}
	}
}
