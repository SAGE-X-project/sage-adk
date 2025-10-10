// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInit(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to temp directory
	os.Chdir(tempDir)

	// Set default flags
	initProtocol = "auto"
	initLLM = "openai"
	initStorage = "memory"

	// Test successful project creation
	projectName := "test-project"
	err := runInit(nil, []string{projectName})
	if err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Verify project directory was created
	projectPath := filepath.Join(tempDir, projectName)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("Project directory was not created")
	}

	// Verify files were created
	expectedFiles := []string{
		"main.go",
		"config.yaml",
		"go.mod",
		"README.md",
		".env.example",
		".gitignore",
	}

	for _, file := range expectedFiles {
		filePath := filepath.Join(projectPath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", file)
		}
	}
}

func TestRunInit_EmptyProjectName(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	err := runInit(nil, []string{""})
	if err == nil {
		t.Error("Expected error for empty project name")
	}
}

func TestCreateMainGo(t *testing.T) {
	tempDir := t.TempDir()

	// Set test configuration
	initLLM = "openai"
	initStorage = "memory"

	err := createMainGo(tempDir)
	if err != nil {
		t.Fatalf("createMainGo failed: %v", err)
	}

	// Verify file was created
	mainPath := filepath.Join(tempDir, "main.go")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Fatal("main.go was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	contentStr := string(content)

	// Check for expected patterns
	expectedPatterns := []string{
		"package main",
		"func main()",
		"builder.NewAgent",
		"OPENAI_API_KEY",
		"NewMemoryStorage",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf("Expected pattern not found in main.go: %s", pattern)
		}
	}
}

func TestCreateConfigYAML(t *testing.T) {
	tempDir := t.TempDir()

	// Set test configuration
	initProtocol = "auto"
	initLLM = "anthropic"
	initStorage = "redis"

	err := createConfigYAML(tempDir)
	if err != nil {
		t.Fatalf("createConfigYAML failed: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tempDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config.yaml: %v", err)
	}

	contentStr := string(content)

	// Verify configuration values
	if !strings.Contains(contentStr, "protocol: auto") {
		t.Error("Expected protocol: auto in config.yaml")
	}
	if !strings.Contains(contentStr, "provider: anthropic") {
		t.Error("Expected provider: anthropic in config.yaml")
	}
	if !strings.Contains(contentStr, "type: redis") {
		t.Error("Expected type: redis in config.yaml")
	}
}

func TestCreateGoMod(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	projectName := "test-project"

	// Create project directory first
	os.MkdirAll(projectName, 0755)

	err := createGoMod(projectName)
	if err != nil {
		t.Fatalf("createGoMod failed: %v", err)
	}

	// Verify file was created
	goModPath := filepath.Join(projectName, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	contentStr := string(content)

	// Verify module name
	if !strings.Contains(contentStr, "module test-project") {
		t.Error("Expected module name 'test-project' in go.mod")
	}

	// Verify dependency
	if !strings.Contains(contentStr, "github.com/sage-x-project/sage-adk") {
		t.Error("Expected sage-adk dependency in go.mod")
	}
}

func TestCreateREADME(t *testing.T) {
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	projectName := "my-agent"

	// Create project directory first
	os.MkdirAll(projectName, 0755)

	err := createREADME(projectName)
	if err != nil {
		t.Fatalf("createREADME failed: %v", err)
	}

	// Verify file was created
	readmePath := filepath.Join(projectName, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	contentStr := string(content)

	// Verify project name appears in README
	if !strings.Contains(contentStr, "# my-agent") {
		t.Error("Expected project name as title in README.md")
	}

	// Verify setup instructions
	if !strings.Contains(contentStr, "## Setup") {
		t.Error("Expected setup section in README.md")
	}
}

func TestCreateEnvExample(t *testing.T) {
	tempDir := t.TempDir()

	err := createEnvExample(tempDir)
	if err != nil {
		t.Fatalf("createEnvExample failed: %v", err)
	}

	// Verify file was created
	envPath := filepath.Join(tempDir, ".env.example")
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("Failed to read .env.example: %v", err)
	}

	contentStr := string(content)

	// Verify API key placeholders
	expectedKeys := []string{
		"OPENAI_API_KEY",
		"ANTHROPIC_API_KEY",
		"GEMINI_API_KEY",
		"REDIS_URL",
		"POSTGRES_URL",
	}

	for _, key := range expectedKeys {
		if !strings.Contains(contentStr, key) {
			t.Errorf("Expected environment variable %s in .env.example", key)
		}
	}
}

func TestCreateGitignore(t *testing.T) {
	tempDir := t.TempDir()

	err := createGitignore(tempDir)
	if err != nil {
		t.Fatalf("createGitignore failed: %v", err)
	}

	// Verify file was created
	gitignorePath := filepath.Join(tempDir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("Failed to read .gitignore: %v", err)
	}

	contentStr := string(content)

	// Verify common patterns
	expectedPatterns := []string{
		".env",
		"*.log",
		".DS_Store",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf("Expected pattern %s in .gitignore", pattern)
		}
	}
}

func TestGetMainGoTemplate_DifferentProviders(t *testing.T) {
	tests := []struct {
		llm     string
		storage string
		want    []string
	}{
		{
			llm:     "openai",
			storage: "memory",
			want:    []string{"OPENAI_API_KEY", "NewMemoryStorage", "NewOpenai"},
		},
		{
			llm:     "anthropic",
			storage: "redis",
			want:    []string{"ANTHROPIC_API_KEY", "NewRedisStorage", "NewAnthropic"},
		},
		{
			llm:     "gemini",
			storage: "postgres",
			want:    []string{"GEMINI_API_KEY", "NewPostgresStorage", "NewGemini"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.llm+"_"+tt.storage, func(t *testing.T) {
			initLLM = tt.llm
			initStorage = tt.storage

			template := getMainGoTemplate()

			for _, want := range tt.want {
				if !strings.Contains(template, want) {
					t.Errorf("Expected %s in template for %s/%s", want, tt.llm, tt.storage)
				}
			}
		})
	}
}
