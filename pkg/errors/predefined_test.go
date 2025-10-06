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

package errors

import (
	"testing"
)

func TestPredefinedErrors_Validation(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		category ErrorCategory
		code     string
	}{
		{"ErrInvalidInput", ErrInvalidInput, CategoryValidation, "INVALID_INPUT"},
		{"ErrMissingField", ErrMissingField, CategoryValidation, "MISSING_FIELD"},
		{"ErrInvalidFormat", ErrInvalidFormat, CategoryValidation, "INVALID_FORMAT"},
		{"ErrInvalidValue", ErrInvalidValue, CategoryValidation, "INVALID_VALUE"},
		{"ErrOutOfRange", ErrOutOfRange, CategoryValidation, "OUT_OF_RANGE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Category != tt.category {
				t.Errorf("Category = %v, want %v", tt.err.Category, tt.category)
			}
			if tt.err.Code != tt.code {
				t.Errorf("Code = %v, want %v", tt.err.Code, tt.code)
			}
		})
	}
}

func TestPredefinedErrors_Protocol(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		category ErrorCategory
		code     string
	}{
		{"ErrProtocolMismatch", ErrProtocolMismatch, CategoryProtocol, "PROTOCOL_MISMATCH"},
		{"ErrUnsupportedProtocol", ErrUnsupportedProtocol, CategoryProtocol, "UNSUPPORTED_PROTOCOL"},
		{"ErrMessageParsing", ErrMessageParsing, CategoryProtocol, "MESSAGE_PARSING_ERROR"},
		{"ErrInvalidProtocolVersion", ErrInvalidProtocolVersion, CategoryProtocol, "INVALID_PROTOCOL_VERSION"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Category != tt.category {
				t.Errorf("Category = %v, want %v", tt.err.Category, tt.category)
			}
			if tt.err.Code != tt.code {
				t.Errorf("Code = %v, want %v", tt.err.Code, tt.code)
			}
		})
	}
}

func TestPredefinedErrors_Security(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
	}{
		{"ErrSignatureInvalid", ErrSignatureInvalid},
		{"ErrDIDNotFound", ErrDIDNotFound},
		{"ErrAgentInactive", ErrAgentInactive},
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrInvalidCredentials", ErrInvalidCredentials},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code == "" {
				t.Error("Code should not be empty")
			}
			if tt.err.Message == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}

func TestPredefinedErrors_Storage(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
	}{
		{"ErrNotFound", ErrNotFound},
		{"ErrStorageConnection", ErrStorageConnection},
		{"ErrStorageTimeout", ErrStorageTimeout},
		{"ErrAlreadyExists", ErrAlreadyExists},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code == "" {
				t.Error("Code should not be empty")
			}
			if tt.err.Message == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}

func TestPredefinedErrors_LLM(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
	}{
		{"ErrLLMConnection", ErrLLMConnection},
		{"ErrLLMRateLimit", ErrLLMRateLimit},
		{"ErrLLMInvalidResponse", ErrLLMInvalidResponse},
		{"ErrLLMTimeout", ErrLLMTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Category != CategoryLLM {
				t.Errorf("Category = %v, want %v", tt.err.Category, CategoryLLM)
			}
			if tt.err.Code == "" {
				t.Error("Code should not be empty")
			}
		})
	}
}

func TestPredefinedErrors_Network(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
	}{
		{"ErrNetworkTimeout", ErrNetworkTimeout},
		{"ErrNetworkUnavailable", ErrNetworkUnavailable},
		{"ErrConnectionRefused", ErrConnectionRefused},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Category != CategoryNetwork {
				t.Errorf("Category = %v, want %v", tt.err.Category, CategoryNetwork)
			}
		})
	}
}

func TestPredefinedErrors_Internal(t *testing.T) {
	tests := []struct {
		name string
		err  *Error
	}{
		{"ErrInternal", ErrInternal},
		{"ErrNotImplemented", ErrNotImplemented},
		{"ErrConfigurationError", ErrConfigurationError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Category != CategoryInternal {
				t.Errorf("Category = %v, want %v", tt.err.Category, CategoryInternal)
			}
		})
	}
}

func TestErrorUsage_WithDetails(t *testing.T) {
	// Test realistic usage scenario
	err := ErrInvalidInput.
		WithDetail("field", "messageId").
		WithDetail("reason", "empty value")

	if err.Details["field"] != "messageId" {
		t.Errorf("field detail = %v, want messageId", err.Details["field"])
	}

	if err.Details["reason"] != "empty value" {
		t.Errorf("reason detail = %v, want empty value", err.Details["reason"])
	}
}

func TestErrorUsage_ChainedOperations(t *testing.T) {
	// Test chaining operations
	err := ErrStorageConnection.
		WithMessage("failed to connect to Redis").
		WithDetails(map[string]interface{}{
			"host": "localhost:6379",
			"timeout": "5s",
		})

	if err.Details["host"] != "localhost:6379" {
		t.Errorf("host = %v, want localhost:6379", err.Details["host"])
	}
}
