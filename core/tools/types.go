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
	"encoding/json"
)

// Tool represents a callable function/tool that an agent can use.
type Tool interface {
	// Name returns the unique name of the tool.
	Name() string

	// Description returns a description of what the tool does.
	Description() string

	// Parameters returns the JSON schema for the tool's parameters.
	Parameters() *ParameterSchema

	// Execute runs the tool with the given parameters.
	Execute(ctx context.Context, params map[string]interface{}) (*Result, error)
}

// ParameterSchema defines the JSON schema for tool parameters.
type ParameterSchema struct {
	Type       string                      `json:"type"`
	Properties map[string]*PropertySchema  `json:"properties,omitempty"`
	Required   []string                    `json:"required,omitempty"`
}

// PropertySchema defines a single parameter property.
type PropertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}

// Result represents the result of a tool execution.
type Result struct {
	// Success indicates whether the tool execution succeeded.
	Success bool `json:"success"`

	// Output contains the tool's output (success case).
	Output interface{} `json:"output,omitempty"`

	// Error contains the error message (failure case).
	Error string `json:"error,omitempty"`

	// Metadata contains additional execution metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolFunc is a function type that can be used as a tool.
type ToolFunc func(ctx context.Context, params map[string]interface{}) (*Result, error)

// FunctionTool is a simple implementation of Tool using a function.
type FunctionTool struct {
	name        string
	description string
	parameters  *ParameterSchema
	execute     ToolFunc
}

// NewFunctionTool creates a new tool from a function.
func NewFunctionTool(name, description string, parameters *ParameterSchema, fn ToolFunc) Tool {
	return &FunctionTool{
		name:        name,
		description: description,
		parameters:  parameters,
		execute:     fn,
	}
}

// Name returns the tool name.
func (t *FunctionTool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *FunctionTool) Description() string {
	return t.description
}

// Parameters returns the parameter schema.
func (t *FunctionTool) Parameters() *ParameterSchema {
	return t.parameters
}

// Execute runs the tool.
func (t *FunctionTool) Execute(ctx context.Context, params map[string]interface{}) (*Result, error) {
	return t.execute(ctx, params)
}

// Registry manages a collection of tools.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return ErrNilTool
	}
	if tool.Name() == "" {
		return ErrEmptyToolName
	}
	if _, exists := r.tools[tool.Name()]; exists {
		return ErrToolAlreadyExists
	}
	r.tools[tool.Name()] = tool
	return nil
}

// Unregister removes a tool from the registry.
func (r *Registry) Unregister(name string) error {
	if _, exists := r.tools[name]; !exists {
		return ErrToolNotFound
	}
	delete(r.tools, name)
	return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, ErrToolNotFound
	}
	return tool, nil
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Has checks if a tool is registered.
func (r *Registry) Has(name string) bool {
	_, exists := r.tools[name]
	return exists
}

// Count returns the number of registered tools.
func (r *Registry) Count() int {
	return len(r.tools)
}

// Execute runs a tool with the given parameters.
func (r *Registry) Execute(ctx context.Context, name string, params map[string]interface{}) (*Result, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	return tool.Execute(ctx, params)
}

// ToLLMFormat converts tools to LLM function calling format.
func (r *Registry) ToLLMFormat() []map[string]interface{} {
	tools := r.List()
	result := make([]map[string]interface{}, len(tools))

	for i, tool := range tools {
		result[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters":  tool.Parameters(),
			},
		}
	}

	return result
}

// SuccessResult creates a successful result.
func SuccessResult(output interface{}) *Result {
	return &Result{
		Success: true,
		Output:  output,
	}
}

// ErrorResult creates an error result.
func ErrorResult(err error) *Result {
	return &Result{
		Success: false,
		Error:   err.Error(),
	}
}

// ErrorResultWithMessage creates an error result with a custom message.
func ErrorResultWithMessage(message string) *Result {
	return &Result{
		Success: false,
		Error:   message,
	}
}

// UnmarshalParams unmarshals parameters into a struct.
func UnmarshalParams(params map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
