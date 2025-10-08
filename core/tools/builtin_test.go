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
	"testing"
)

func TestCalculatorTool_Add(t *testing.T) {
	tool := CalculatorTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"operation": "add",
		"a":         5.0,
		"b":         3.0,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true: %s", result.Error)
	}
	if result.Output != 8.0 {
		t.Errorf("Output = %v, want 8.0", result.Output)
	}
}

func TestCalculatorTool_Subtract(t *testing.T) {
	tool := CalculatorTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"operation": "subtract",
		"a":         10.0,
		"b":         4.0,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if result.Output != 6.0 {
		t.Errorf("Output = %v, want 6.0", result.Output)
	}
}

func TestCalculatorTool_Multiply(t *testing.T) {
	tool := CalculatorTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"operation": "multiply",
		"a":         6.0,
		"b":         7.0,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if result.Output != 42.0 {
		t.Errorf("Output = %v, want 42.0", result.Output)
	}
}

func TestCalculatorTool_Divide(t *testing.T) {
	tool := CalculatorTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"operation": "divide",
		"a":         20.0,
		"b":         5.0,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if result.Output != 4.0 {
		t.Errorf("Output = %v, want 4.0", result.Output)
	}
}

func TestCalculatorTool_DivideByZero(t *testing.T) {
	tool := CalculatorTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"operation": "divide",
		"a":         10.0,
		"b":         0.0,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success {
		t.Error("Success = true for division by zero, want false")
	}
	if result.Error == "" {
		t.Error("Error message is empty for division by zero")
	}
}

func TestCalculatorTool_InvalidOperation(t *testing.T) {
	tool := CalculatorTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"operation": "modulo",
		"a":         10.0,
		"b":         3.0,
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success {
		t.Error("Success = true for invalid operation, want false")
	}
}

func TestCurrentTimeTool_RFC3339(t *testing.T) {
	tool := CurrentTimeTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"format":   "RFC3339",
		"timezone": "UTC",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true: %s", result.Error)
	}
	if result.Output == nil {
		t.Error("Output is nil")
	}
	if result.Metadata == nil {
		t.Error("Metadata is nil")
	}
}

func TestCurrentTimeTool_Unix(t *testing.T) {
	tool := CurrentTimeTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"format": "Unix",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if result.Output == nil {
		t.Error("Output is nil")
	}
}

func TestCurrentTimeTool_Human(t *testing.T) {
	tool := CurrentTimeTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"format":   "Human",
		"timezone": "America/New_York",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true")
	}
}

func TestCurrentTimeTool_InvalidTimezone(t *testing.T) {
	tool := CurrentTimeTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"timezone": "Invalid/Timezone",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success {
		t.Error("Success = true for invalid timezone, want false")
	}
}

func TestEchoTool(t *testing.T) {
	tool := EchoTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"message": "Hello, World!",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if result.Output != "Hello, World!" {
		t.Errorf("Output = %v, want Hello, World!", result.Output)
	}
}

func TestEchoTool_EmptyMessage(t *testing.T) {
	tool := EchoTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"message": "",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if result.Output != "" {
		t.Errorf("Output = %v, want empty string", result.Output)
	}
}

func TestStringLengthTool(t *testing.T) {
	tool := StringLengthTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"text": "Hello, World!",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if result.Output != 13 {
		t.Errorf("Output = %v, want 13", result.Output)
	}
}

func TestStringLengthTool_EmptyString(t *testing.T) {
	tool := StringLengthTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"text": "",
	})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Success {
		t.Errorf("Success = false, want true")
	}
	if result.Output != 0 {
		t.Errorf("Output = %v, want 0", result.Output)
	}
}

func TestRegisterBuiltinTools(t *testing.T) {
	registry := NewRegistry()
	err := RegisterBuiltinTools(registry)

	if err != nil {
		t.Fatalf("RegisterBuiltinTools() error = %v", err)
	}

	expectedTools := []string{"calculator", "current_time", "echo", "string_length"}
	for _, name := range expectedTools {
		if !registry.Has(name) {
			t.Errorf("Registry missing tool: %s", name)
		}
	}

	if registry.Count() != len(expectedTools) {
		t.Errorf("Registry count = %d, want %d", registry.Count(), len(expectedTools))
	}
}

func TestBuiltinTools_Integration(t *testing.T) {
	registry := NewRegistry()
	RegisterBuiltinTools(registry)

	// Test calculator
	calcResult, err := registry.Execute(context.Background(), "calculator", map[string]interface{}{
		"operation": "multiply",
		"a":         6.0,
		"b":         7.0,
	})
	if err != nil || !calcResult.Success || calcResult.Output != 42.0 {
		t.Error("Calculator tool integration test failed")
	}

	// Test echo
	echoResult, err := registry.Execute(context.Background(), "echo", map[string]interface{}{
		"message": "test",
	})
	if err != nil || !echoResult.Success || echoResult.Output != "test" {
		t.Error("Echo tool integration test failed")
	}

	// Test string_length
	lenResult, err := registry.Execute(context.Background(), "string_length", map[string]interface{}{
		"text": "hello",
	})
	if err != nil || !lenResult.Success || lenResult.Output != 5 {
		t.Error("String length tool integration test failed")
	}
}
