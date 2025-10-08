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
	"errors"
	"strings"
	"testing"
)

func TestError_Error(t *testing.T) {
	err := &Error{
		Category: CategoryValidation,
		Code:     "TEST_ERROR",
		Message:  "test error message",
	}

	got := err.Error()
	if !strings.Contains(got, "TEST_ERROR") {
		t.Errorf("Error() = %v, should contain TEST_ERROR", got)
	}
	if !strings.Contains(got, "test error message") {
		t.Errorf("Error() = %v, should contain message", got)
	}
}

func TestError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := &Error{
		Category: CategoryInternal,
		Code:     "WRAPPED",
		Message:  "wrapped error",
		Err:      innerErr,
	}

	if err.Unwrap() != innerErr {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), innerErr)
	}
}

func TestError_Is(t *testing.T) {
	baseErr := &Error{
		Category: CategoryValidation,
		Code:     "INVALID_INPUT",
		Message:  "invalid input",
	}

	tests := []struct {
		name   string
		err    *Error
		target error
		want   bool
	}{
		{
			name:   "same error",
			err:    baseErr,
			target: baseErr,
			want:   true,
		},
		{
			name: "same code",
			err: &Error{
				Category: CategoryValidation,
				Code:     "INVALID_INPUT",
				Message:  "different message",
			},
			target: baseErr,
			want:   true,
		},
		{
			name: "different code",
			err: &Error{
				Category: CategoryValidation,
				Code:     "DIFFERENT_CODE",
				Message:  "invalid input",
			},
			target: baseErr,
			want:   false,
		},
		{
			name:   "not ADK error",
			err:    baseErr,
			target: errors.New("standard error"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Is(tt.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestError_WithMessage(t *testing.T) {
	baseErr := &Error{
		Category: CategoryValidation,
		Code:     "TEST_ERROR",
		Message:  "original message",
	}

	newErr := baseErr.WithMessage("additional context")

	if newErr == baseErr {
		t.Error("WithMessage() should return a new error instance")
	}

	if !strings.Contains(newErr.Message, "additional context") {
		t.Errorf("Message = %v, should contain additional context", newErr.Message)
	}
}

func TestError_WithDetail(t *testing.T) {
	err := &Error{
		Category: CategoryValidation,
		Code:     "TEST_ERROR",
		Message:  "test error",
	}

	newErr := err.WithDetail("field", "messageId")

	if newErr.Details == nil {
		t.Fatal("Details should be initialized")
	}

	if newErr.Details["field"] != "messageId" {
		t.Errorf("Details[field] = %v, want messageId", newErr.Details["field"])
	}
}

func TestError_WithDetails(t *testing.T) {
	err := &Error{
		Category: CategoryValidation,
		Code:     "TEST_ERROR",
		Message:  "test error",
	}

	details := map[string]interface{}{
		"field1": "value1",
		"field2": 123,
	}

	newErr := err.WithDetails(details)

	if len(newErr.Details) != 2 {
		t.Errorf("Details length = %v, want 2", len(newErr.Details))
	}

	if newErr.Details["field1"] != "value1" {
		t.Errorf("Details[field1] = %v, want value1", newErr.Details["field1"])
	}
}

func TestError_Wrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := &Error{
		Category: CategoryInternal,
		Code:     "WRAPPER",
		Message:  "wrapper error",
	}

	wrapped := err.Wrap(innerErr)

	if wrapped.Err != innerErr {
		t.Errorf("Err = %v, want %v", wrapped.Err, innerErr)
	}

	// Should be able to unwrap
	if !errors.Is(wrapped, innerErr) {
		t.Error("Wrapped error should match inner error with errors.Is")
	}
}

func TestNew(t *testing.T) {
	err := New(CategoryValidation, "TEST_CODE", "test message")

	if err.Category != CategoryValidation {
		t.Errorf("Category = %v, want %v", err.Category, CategoryValidation)
	}

	if err.Code != "TEST_CODE" {
		t.Errorf("Code = %v, want TEST_CODE", err.Code)
	}

	if err.Message != "test message" {
		t.Errorf("Message = %v, want test message", err.Message)
	}
}

func TestWrap(t *testing.T) {
	innerErr := errors.New("inner error")
	wrapped := Wrap(innerErr, "additional context")

	if !errors.Is(wrapped, innerErr) {
		t.Error("Wrapped error should match inner error")
	}

	errStr := wrapped.Error()
	if !strings.Contains(errStr, "additional context") {
		t.Errorf("Error string = %v, should contain additional context", errStr)
	}
}

func TestIs(t *testing.T) {
	baseErr := &Error{
		Category: CategoryValidation,
		Code:     "INVALID_INPUT",
		Message:  "invalid input",
	}

	wrappedErr := baseErr.Wrap(errors.New("inner"))

	if !Is(wrappedErr, baseErr) {
		t.Error("Is() should return true for matching error codes")
	}

	differentErr := &Error{
		Category: CategoryValidation,
		Code:     "DIFFERENT_CODE",
		Message:  "different error",
	}

	if Is(wrappedErr, differentErr) {
		t.Error("Is() should return false for different error codes")
	}
}

func TestAs(t *testing.T) {
	adkErr := &Error{
		Category: CategoryValidation,
		Code:     "TEST_ERROR",
		Message:  "test error",
	}

	var target *Error
	if !As(adkErr, &target) {
		t.Error("As() should return true for matching type")
	}

	if target.Code != "TEST_ERROR" {
		t.Errorf("Extracted error code = %v, want TEST_ERROR", target.Code)
	}
}

func TestErrorCategory_String(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		want     string
	}{
		{CategoryValidation, "validation"},
		{CategoryProtocol, "protocol"},
		{CategorySecurity, "security"},
		{CategoryStorage, "storage"},
		{CategoryLLM, "llm"},
		{CategoryNetwork, "network"},
		{CategoryInternal, "internal"},
		{CategoryNotFound, "not_found"},
		{CategoryUnauthorized, "unauthorized"},
	}

	for _, tt := range tests {
		t.Run(string(tt.category), func(t *testing.T) {
			if string(tt.category) != tt.want {
				t.Errorf("Category = %v, want %v", tt.category, tt.want)
			}
		})
	}
}
