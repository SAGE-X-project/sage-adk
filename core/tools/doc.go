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

// Package tools provides a framework for defining and managing agent tools.
//
// Tools are external functions that agents can call to perform specific tasks.
// This package provides:
//   - Tool interface for defining callable functions
//   - Registry for managing collections of tools
//   - Parameter schemas using JSON Schema
//   - Result types for tool execution
//
// Example:
//
//	// Define a simple calculator tool
//	calculator := tools.NewFunctionTool(
//	    "calculator",
//	    "Performs basic arithmetic operations",
//	    &tools.ParameterSchema{
//	        Type: "object",
//	        Properties: map[string]*tools.PropertySchema{
//	            "operation": {
//	                Type:        "string",
//	                Description: "The operation to perform",
//	                Enum:        []string{"add", "subtract", "multiply", "divide"},
//	            },
//	            "a": {Type: "number", Description: "First number"},
//	            "b": {Type: "number", Description: "Second number"},
//	        },
//	        Required: []string{"operation", "a", "b"},
//	    },
//	    func(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
//	        op := params["operation"].(string)
//	        a := params["a"].(float64)
//	        b := params["b"].(float64)
//
//	        var result float64
//	        switch op {
//	        case "add":
//	            result = a + b
//	        case "subtract":
//	            result = a - b
//	        case "multiply":
//	            result = a * b
//	        case "divide":
//	            if b == 0 {
//	                return tools.ErrorResultWithMessage("division by zero"), nil
//	            }
//	            result = a / b
//	        }
//
//	        return tools.SuccessResult(result), nil
//	    },
//	)
//
//	// Register tool
//	registry := tools.NewRegistry()
//	registry.Register(calculator)
//
//	// Execute tool
//	result, err := registry.Execute(ctx, "calculator", map[string]interface{}{
//	    "operation": "add",
//	    "a":         5.0,
//	    "b":         3.0,
//	})
package tools
