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

package tools

import (
	"context"
	"errors"
	"testing"
)

func TestNewFunctionTool(t *testing.T) {
	tool := NewFunctionTool(
		"test_tool",
		"A test tool",
		&ParameterSchema{Type: "object"},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			return SuccessResult("ok"), nil
		},
	)

	if tool.Name() != "test_tool" {
		t.Errorf("Name() = %v, want test_tool", tool.Name())
	}
	if tool.Description() != "A test tool" {
		t.Errorf("Description() = %v, want A test tool", tool.Description())
	}
	if tool.Parameters() == nil {
		t.Error("Parameters() returned nil")
	}
}

func TestFunctionTool_Execute(t *testing.T) {
	called := false
	tool := NewFunctionTool(
		"test",
		"test",
		&ParameterSchema{Type: "object"},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			called = true
			return SuccessResult(params["input"]), nil
		},
	)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"input": "hello",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !called {
		t.Error("Tool function was not called")
	}
	if !result.Success {
		t.Error("Result.Success = false, want true")
	}
	if result.Output != "hello" {
		t.Errorf("Result.Output = %v, want hello", result.Output)
	}
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	tool := createTestTool("tool1")

	err := registry.Register(tool)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("Count() = %d, want 1", registry.Count())
	}

	if !registry.Has("tool1") {
		t.Error("Has(tool1) = false, want true")
	}
}

func TestRegistry_Register_NilTool(t *testing.T) {
	registry := NewRegistry()
	err := registry.Register(nil)

	if err != ErrNilTool {
		t.Errorf("Register(nil) error = %v, want ErrNilTool", err)
	}
}

func TestRegistry_Register_EmptyName(t *testing.T) {
	registry := NewRegistry()
	tool := NewFunctionTool(
		"",
		"test",
		&ParameterSchema{Type: "object"},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			return SuccessResult("ok"), nil
		},
	)

	err := registry.Register(tool)
	if err != ErrEmptyToolName {
		t.Errorf("Register() error = %v, want ErrEmptyToolName", err)
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	registry := NewRegistry()
	tool1 := createTestTool("tool1")
	tool2 := createTestTool("tool1") // Same name

	registry.Register(tool1)
	err := registry.Register(tool2)

	if err != ErrToolAlreadyExists {
		t.Errorf("Register(duplicate) error = %v, want ErrToolAlreadyExists", err)
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	tool := createTestTool("tool1")
	registry.Register(tool)

	retrieved, err := registry.Get("tool1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if retrieved.Name() != "tool1" {
		t.Errorf("Retrieved tool name = %v, want tool1", retrieved.Name())
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Get("nonexistent")
	if err != ErrToolNotFound {
		t.Errorf("Get(nonexistent) error = %v, want ErrToolNotFound", err)
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()
	tool := createTestTool("tool1")
	registry.Register(tool)

	err := registry.Unregister("tool1")
	if err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	if registry.Count() != 0 {
		t.Errorf("Count() = %d, want 0", registry.Count())
	}

	if registry.Has("tool1") {
		t.Error("Has(tool1) = true after unregister, want false")
	}
}

func TestRegistry_Unregister_NotFound(t *testing.T) {
	registry := NewRegistry()

	err := registry.Unregister("nonexistent")
	if err != ErrToolNotFound {
		t.Errorf("Unregister(nonexistent) error = %v, want ErrToolNotFound", err)
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()
	tool1 := createTestTool("tool1")
	tool2 := createTestTool("tool2")
	tool3 := createTestTool("tool3")

	registry.Register(tool1)
	registry.Register(tool2)
	registry.Register(tool3)

	tools := registry.List()
	if len(tools) != 3 {
		t.Errorf("List() length = %d, want 3", len(tools))
	}

	// Check all tools are present
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}

	if !names["tool1"] || !names["tool2"] || !names["tool3"] {
		t.Error("List() missing expected tools")
	}
}

func TestRegistry_Execute(t *testing.T) {
	registry := NewRegistry()

	tool := NewFunctionTool(
		"echo",
		"Echoes input",
		&ParameterSchema{Type: "object"},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			return SuccessResult(params["message"]), nil
		},
	)

	registry.Register(tool)

	result, err := registry.Execute(context.Background(), "echo", map[string]interface{}{
		"message": "hello world",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Error("Result.Success = false, want true")
	}
	if result.Output != "hello world" {
		t.Errorf("Result.Output = %v, want hello world", result.Output)
	}
}

func TestRegistry_Execute_NotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Execute(context.Background(), "nonexistent", nil)
	if err != ErrToolNotFound {
		t.Errorf("Execute(nonexistent) error = %v, want ErrToolNotFound", err)
	}
}

func TestRegistry_ToLLMFormat(t *testing.T) {
	registry := NewRegistry()

	tool := NewFunctionTool(
		"calculator",
		"Performs calculations",
		&ParameterSchema{
			Type: "object",
			Properties: map[string]*PropertySchema{
				"operation": {Type: "string", Description: "The operation"},
			},
			Required: []string{"operation"},
		},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			return SuccessResult(0), nil
		},
	)

	registry.Register(tool)

	format := registry.ToLLMFormat()

	if len(format) != 1 {
		t.Fatalf("ToLLMFormat() length = %d, want 1", len(format))
	}

	toolDef := format[0]
	if toolDef["type"] != "function" {
		t.Errorf("Tool type = %v, want function", toolDef["type"])
	}

	fn := toolDef["function"].(map[string]interface{})
	if fn["name"] != "calculator" {
		t.Errorf("Function name = %v, want calculator", fn["name"])
	}
	if fn["description"] != "Performs calculations" {
		t.Errorf("Function description = %v, want Performs calculations", fn["description"])
	}
}

func TestSuccessResult(t *testing.T) {
	result := SuccessResult("test output")

	if !result.Success {
		t.Error("Success = false, want true")
	}
	if result.Output != "test output" {
		t.Errorf("Output = %v, want test output", result.Output)
	}
	if result.Error != "" {
		t.Errorf("Error = %v, want empty string", result.Error)
	}
}

func TestErrorResult(t *testing.T) {
	err := errors.New("test error")
	result := ErrorResult(err)

	if result.Success {
		t.Error("Success = true, want false")
	}
	if result.Error != "test error" {
		t.Errorf("Error = %v, want test error", result.Error)
	}
	if result.Output != nil {
		t.Errorf("Output = %v, want nil", result.Output)
	}
}

func TestErrorResultWithMessage(t *testing.T) {
	result := ErrorResultWithMessage("custom error")

	if result.Success {
		t.Error("Success = true, want false")
	}
	if result.Error != "custom error" {
		t.Errorf("Error = %v, want custom error", result.Error)
	}
}

func TestUnmarshalParams(t *testing.T) {
	type TestParams struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	params := map[string]interface{}{
		"name":  "John Doe",
		"age":   float64(30), // JSON numbers are float64
		"email": "john@example.com",
	}

	var result TestParams
	err := UnmarshalParams(params, &result)

	if err != nil {
		t.Fatalf("UnmarshalParams() error = %v", err)
	}

	if result.Name != "John Doe" {
		t.Errorf("Name = %v, want John Doe", result.Name)
	}
	if result.Age != 30 {
		t.Errorf("Age = %v, want 30", result.Age)
	}
	if result.Email != "john@example.com" {
		t.Errorf("Email = %v, want john@example.com", result.Email)
	}
}

// Helper function to create a test tool
func createTestTool(name string) Tool {
	return NewFunctionTool(
		name,
		"A test tool",
		&ParameterSchema{Type: "object"},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			return SuccessResult("ok"), nil
		},
	)
}
