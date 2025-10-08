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
	"encoding/json"
	"fmt"
)

// FunctionCall represents a function call made by the LLM.
type FunctionCall struct {
	// Name is the function name to call.
	Name string `json:"name"`

	// Arguments is the JSON-encoded function arguments.
	Arguments string `json:"arguments"`
}

// ParsedArguments parses the arguments into a map.
func (fc *FunctionCall) ParsedArguments() (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(fc.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse function arguments: %w", err)
	}
	return args, nil
}

// Function defines a function that can be called by the LLM.
type Function struct {
	// Name is the function name.
	Name string `json:"name"`

	// Description describes what the function does.
	Description string `json:"description"`

	// Parameters defines the function parameters using JSON Schema.
	Parameters *FunctionParameters `json:"parameters"`
}

// FunctionParameters defines function parameters using JSON Schema.
type FunctionParameters struct {
	// Type is the schema type (usually "object").
	Type string `json:"type"`

	// Properties defines the parameter properties.
	Properties map[string]*PropertySchema `json:"properties"`

	// Required lists the required parameter names.
	Required []string `json:"required,omitempty"`
}

// PropertySchema defines a property schema.
type PropertySchema struct {
	// Type is the property type (string, number, boolean, etc.).
	Type string `json:"type"`

	// Description describes the property.
	Description string `json:"description,omitempty"`

	// Enum lists allowed values (for string type).
	Enum []string `json:"enum,omitempty"`

	// Items defines array item schema (for array type).
	Items *PropertySchema `json:"items,omitempty"`

	// Properties defines nested object properties (for object type).
	Properties map[string]*PropertySchema `json:"properties,omitempty"`
}

// FunctionChoice controls function calling behavior.
type FunctionChoice string

const (
	// FunctionChoiceNone disables function calling.
	FunctionChoiceNone FunctionChoice = "none"

	// FunctionChoiceAuto lets the model decide.
	FunctionChoiceAuto FunctionChoice = "auto"

	// FunctionChoiceRequired forces the model to call a function.
	FunctionChoiceRequired FunctionChoice = "required"
)

// ToolType represents the type of tool.
type ToolType string

const (
	// ToolTypeFunction represents a function tool.
	ToolTypeFunction ToolType = "function"
)

// Tool represents a tool available to the LLM.
type Tool struct {
	// Type is the tool type.
	Type ToolType `json:"type"`

	// Function is the function definition.
	Function *Function `json:"function"`
}

// ToolCall represents a tool call made by the LLM.
type ToolCall struct {
	// ID is the tool call identifier.
	ID string `json:"id"`

	// Type is the tool type.
	Type ToolType `json:"type"`

	// Function is the function call.
	Function *FunctionCall `json:"function"`
}

// ToolCallResult represents the result of executing a tool call.
type ToolCallResult struct {
	// ToolCallID is the ID of the tool call this result is for.
	ToolCallID string `json:"tool_call_id"`

	// Role should be "tool".
	Role MessageRole `json:"role"`

	// Content is the tool execution result.
	Content string `json:"content"`

	// Name is the tool name.
	Name string `json:"name,omitempty"`
}

// MessageWithToolCalls extends Message to support tool calls.
type MessageWithToolCalls struct {
	Message

	// ToolCalls contains function/tool calls made by the assistant.
	ToolCalls []*ToolCall `json:"tool_calls,omitempty"`
}

// CompletionRequestWithTools extends CompletionRequest to support function calling.
type CompletionRequestWithTools struct {
	CompletionRequest

	// Tools is the list of tools available to the model.
	Tools []*Tool `json:"tools,omitempty"`

	// ToolChoice controls function calling behavior.
	ToolChoice interface{} `json:"tool_choice,omitempty"` // Can be string or object
}

// CompletionResponseWithTools extends CompletionResponse to support function calling.
type CompletionResponseWithTools struct {
	CompletionResponse

	// ToolCalls contains function/tool calls made by the assistant.
	ToolCalls []*ToolCall `json:"tool_calls,omitempty"`
}

// NewFunction creates a new function definition.
func NewFunction(name, description string, parameters *FunctionParameters) *Function {
	return &Function{
		Name:        name,
		Description: description,
		Parameters:  parameters,
	}
}

// NewFunctionParameters creates new function parameters.
func NewFunctionParameters() *FunctionParameters {
	return &FunctionParameters{
		Type:       "object",
		Properties: make(map[string]*PropertySchema),
		Required:   make([]string, 0),
	}
}

// AddProperty adds a property to the function parameters.
func (fp *FunctionParameters) AddProperty(name, propType, description string, required bool) *FunctionParameters {
	fp.Properties[name] = &PropertySchema{
		Type:        propType,
		Description: description,
	}
	if required {
		fp.Required = append(fp.Required, name)
	}
	return fp
}

// AddEnumProperty adds an enum property to the function parameters.
func (fp *FunctionParameters) AddEnumProperty(name, description string, enum []string, required bool) *FunctionParameters {
	fp.Properties[name] = &PropertySchema{
		Type:        "string",
		Description: description,
		Enum:        enum,
	}
	if required {
		fp.Required = append(fp.Required, name)
	}
	return fp
}

// NewTool creates a new tool from a function.
func NewTool(function *Function) *Tool {
	return &Tool{
		Type:     ToolTypeFunction,
		Function: function,
	}
}
