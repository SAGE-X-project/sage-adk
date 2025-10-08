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
	"fmt"
	"time"
)

// CalculatorTool creates a calculator tool for basic arithmetic.
func CalculatorTool() Tool {
	return NewFunctionTool(
		"calculator",
		"Performs basic arithmetic operations (add, subtract, multiply, divide)",
		&ParameterSchema{
			Type: "object",
			Properties: map[string]*PropertySchema{
				"operation": {
					Type:        "string",
					Description: "The arithmetic operation to perform",
					Enum:        []string{"add", "subtract", "multiply", "divide"},
				},
				"a": {
					Type:        "number",
					Description: "The first number",
				},
				"b": {
					Type:        "number",
					Description: "The second number",
				},
			},
			Required: []string{"operation", "a", "b"},
		},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			operation, ok := params["operation"].(string)
			if !ok {
				return ErrorResultWithMessage("operation must be a string"), nil
			}

			a, ok := params["a"].(float64)
			if !ok {
				return ErrorResultWithMessage("a must be a number"), nil
			}

			b, ok := params["b"].(float64)
			if !ok {
				return ErrorResultWithMessage("b must be a number"), nil
			}

			var result float64
			switch operation {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return ErrorResultWithMessage("division by zero"), nil
				}
				result = a / b
			default:
				return ErrorResultWithMessage(fmt.Sprintf("unknown operation: %s", operation)), nil
			}

			return SuccessResult(result), nil
		},
	)
}

// CurrentTimeTool creates a tool that returns the current time.
func CurrentTimeTool() Tool {
	return NewFunctionTool(
		"current_time",
		"Returns the current date and time",
		&ParameterSchema{
			Type: "object",
			Properties: map[string]*PropertySchema{
				"format": {
					Type:        "string",
					Description: "The time format (RFC3339, Unix, or Human)",
					Enum:        []string{"RFC3339", "Unix", "Human"},
					Default:     "RFC3339",
				},
				"timezone": {
					Type:        "string",
					Description: "The timezone (e.g., UTC, America/New_York, Asia/Tokyo)",
					Default:     "UTC",
				},
			},
			Required: []string{},
		},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			format := "RFC3339"
			if f, ok := params["format"].(string); ok {
				format = f
			}

			timezone := "UTC"
			if tz, ok := params["timezone"].(string); ok {
				timezone = tz
			}

			// Load timezone
			loc, err := time.LoadLocation(timezone)
			if err != nil {
				return ErrorResultWithMessage(fmt.Sprintf("invalid timezone: %s", timezone)), nil
			}

			now := time.Now().In(loc)

			var output string
			switch format {
			case "RFC3339":
				output = now.Format(time.RFC3339)
			case "Unix":
				output = fmt.Sprintf("%d", now.Unix())
			case "Human":
				output = now.Format("Monday, January 2, 2006 at 3:04:05 PM MST")
			default:
				return ErrorResultWithMessage(fmt.Sprintf("unknown format: %s", format)), nil
			}

			return &Result{
				Success: true,
				Output:  output,
				Metadata: map[string]interface{}{
					"timestamp": now.Unix(),
					"timezone":  timezone,
					"format":    format,
				},
			}, nil
		},
	)
}

// EchoTool creates a simple echo tool for testing.
func EchoTool() Tool {
	return NewFunctionTool(
		"echo",
		"Echoes back the input message",
		&ParameterSchema{
			Type: "object",
			Properties: map[string]*PropertySchema{
				"message": {
					Type:        "string",
					Description: "The message to echo back",
				},
			},
			Required: []string{"message"},
		},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			message, ok := params["message"].(string)
			if !ok {
				return ErrorResultWithMessage("message must be a string"), nil
			}

			return SuccessResult(message), nil
		},
	)
}

// StringLengthTool creates a tool that returns the length of a string.
func StringLengthTool() Tool {
	return NewFunctionTool(
		"string_length",
		"Returns the length of a string",
		&ParameterSchema{
			Type: "object",
			Properties: map[string]*PropertySchema{
				"text": {
					Type:        "string",
					Description: "The text to measure",
				},
			},
			Required: []string{"text"},
		},
		func(ctx context.Context, params map[string]interface{}) (*Result, error) {
			text, ok := params["text"].(string)
			if !ok {
				return ErrorResultWithMessage("text must be a string"), nil
			}

			return &Result{
				Success: true,
				Output:  len(text),
				Metadata: map[string]interface{}{
					"text": text,
				},
			}, nil
		},
	)
}

// RegisterBuiltinTools registers all builtin tools to a registry.
func RegisterBuiltinTools(registry *Registry) error {
	tools := []Tool{
		CalculatorTool(),
		CurrentTimeTool(),
		EchoTool(),
		StringLengthTool(),
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
