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
	"testing"
)

func TestFunctionCall_ParsedArguments(t *testing.T) {
	fc := &FunctionCall{
		Name:      "get_weather",
		Arguments: `{"location": "San Francisco", "unit": "celsius"}`,
	}

	args, err := fc.ParsedArguments()
	if err != nil {
		t.Fatalf("ParsedArguments() error = %v", err)
	}

	if args["location"] != "San Francisco" {
		t.Errorf("location = %v, want San Francisco", args["location"])
	}
	if args["unit"] != "celsius" {
		t.Errorf("unit = %v, want celsius", args["unit"])
	}
}

func TestFunctionCall_ParsedArguments_Invalid(t *testing.T) {
	fc := &FunctionCall{
		Name:      "test",
		Arguments: `invalid json`,
	}

	_, err := fc.ParsedArguments()
	if err == nil {
		t.Error("ParsedArguments() should return error for invalid JSON")
	}
}

func TestNewFunction(t *testing.T) {
	params := NewFunctionParameters()
	params.AddProperty("location", "string", "The city and state", true)

	fn := NewFunction("get_weather", "Get the current weather", params)

	if fn.Name != "get_weather" {
		t.Errorf("Name = %s, want get_weather", fn.Name)
	}
	if fn.Description != "Get the current weather" {
		t.Errorf("Description = %s, want Get the current weather", fn.Description)
	}
	if fn.Parameters == nil {
		t.Error("Parameters should not be nil")
	}
}

func TestNewFunctionParameters(t *testing.T) {
	params := NewFunctionParameters()

	if params.Type != "object" {
		t.Errorf("Type = %s, want object", params.Type)
	}
	if params.Properties == nil {
		t.Error("Properties should not be nil")
	}
	if params.Required == nil {
		t.Error("Required should not be nil")
	}
}

func TestFunctionParameters_AddProperty(t *testing.T) {
	params := NewFunctionParameters()
	params.AddProperty("location", "string", "The location", true)
	params.AddProperty("unit", "string", "The unit", false)

	if len(params.Properties) != 2 {
		t.Errorf("Properties count = %d, want 2", len(params.Properties))
	}

	if params.Properties["location"].Type != "string" {
		t.Errorf("location type = %s, want string", params.Properties["location"].Type)
	}

	if len(params.Required) != 1 {
		t.Errorf("Required count = %d, want 1", len(params.Required))
	}
	if params.Required[0] != "location" {
		t.Errorf("Required[0] = %s, want location", params.Required[0])
	}
}

func TestFunctionParameters_AddEnumProperty(t *testing.T) {
	params := NewFunctionParameters()
	params.AddEnumProperty("unit", "Temperature unit", []string{"celsius", "fahrenheit"}, true)

	if len(params.Properties) != 1 {
		t.Errorf("Properties count = %d, want 1", len(params.Properties))
	}

	prop := params.Properties["unit"]
	if prop.Type != "string" {
		t.Errorf("unit type = %s, want string", prop.Type)
	}
	if len(prop.Enum) != 2 {
		t.Errorf("Enum count = %d, want 2", len(prop.Enum))
	}

	if len(params.Required) != 1 {
		t.Errorf("Required count = %d, want 1", len(params.Required))
	}
}

func TestNewTool(t *testing.T) {
	params := NewFunctionParameters()
	fn := NewFunction("test", "Test function", params)
	tool := NewTool(fn)

	if tool.Type != ToolTypeFunction {
		t.Errorf("Type = %v, want %v", tool.Type, ToolTypeFunction)
	}
	if tool.Function != fn {
		t.Error("Function should match the provided function")
	}
}

func TestToolCall_JSON(t *testing.T) {
	tc := &ToolCall{
		ID:   "call_123",
		Type: ToolTypeFunction,
		Function: &FunctionCall{
			Name:      "get_weather",
			Arguments: `{"location": "NYC"}`,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Unmarshal back
	var tc2 ToolCall
	if err := json.Unmarshal(data, &tc2); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if tc2.ID != tc.ID {
		t.Errorf("ID = %s, want %s", tc2.ID, tc.ID)
	}
	if tc2.Function.Name != tc.Function.Name {
		t.Errorf("Function.Name = %s, want %s", tc2.Function.Name, tc.Function.Name)
	}
}

func TestFunctionParameters_ChainedAddProperty(t *testing.T) {
	params := NewFunctionParameters().
		AddProperty("name", "string", "Name", true).
		AddProperty("age", "number", "Age", false).
		AddEnumProperty("status", "Status", []string{"active", "inactive"}, true)

	if len(params.Properties) != 3 {
		t.Errorf("Properties count = %d, want 3", len(params.Properties))
	}

	if len(params.Required) != 2 {
		t.Errorf("Required count = %d, want 2", len(params.Required))
	}
}

func TestPropertySchema_Nested(t *testing.T) {
	params := NewFunctionParameters()
	params.Properties["address"] = &PropertySchema{
		Type: "object",
		Properties: map[string]*PropertySchema{
			"street": {
				Type:        "string",
				Description: "Street address",
			},
			"city": {
				Type:        "string",
				Description: "City name",
			},
		},
	}

	if params.Properties["address"].Type != "object" {
		t.Error("address should be object type")
	}
	if len(params.Properties["address"].Properties) != 2 {
		t.Errorf("Nested properties count = %d, want 2",
			len(params.Properties["address"].Properties))
	}
}

func TestPropertySchema_Array(t *testing.T) {
	params := NewFunctionParameters()
	params.Properties["tags"] = &PropertySchema{
		Type: "array",
		Items: &PropertySchema{
			Type: "string",
		},
	}

	if params.Properties["tags"].Type != "array" {
		t.Error("tags should be array type")
	}
	if params.Properties["tags"].Items == nil {
		t.Error("array items should be defined")
	}
	if params.Properties["tags"].Items.Type != "string" {
		t.Error("array items should be string type")
	}
}
