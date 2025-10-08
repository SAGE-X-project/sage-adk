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
	"testing"
)

func TestError_ErrorWithoutWrappedError(t *testing.T) {
	err := &Error{
		Category: CategoryValidation,
		Code:     "TEST",
		Message:  "test message",
		Err:      nil,
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}
}

func TestError_UnwrapNil(t *testing.T) {
	err := &Error{
		Category: CategoryValidation,
		Code:     "TEST",
		Message:  "test message",
	}

	if err.Unwrap() != nil {
		t.Error("Unwrap() should return nil when no wrapped error")
	}
}

func TestError_WithDetailNilDetails(t *testing.T) {
	err := &Error{
		Category: CategoryValidation,
		Code:     "TEST",
		Message:  "test message",
		Details:  nil,
	}

	newErr := err.WithDetail("key", "value")

	if newErr.Details == nil {
		t.Fatal("Details should be initialized")
	}

	if newErr.Details["key"] != "value" {
		t.Errorf("Details[key] = %v, want value", newErr.Details["key"])
	}
}

func TestError_WithDetailsNilDetails(t *testing.T) {
	err := &Error{
		Category: CategoryValidation,
		Code:     "TEST",
		Message:  "test message",
		Details:  nil,
	}

	newErr := err.WithDetails(map[string]interface{}{"key": "value"})

	if newErr.Details == nil {
		t.Fatal("Details should be initialized")
	}

	if newErr.Details["key"] != "value" {
		t.Errorf("Details[key] = %v, want value", newErr.Details["key"])
	}
}

func TestError_AsNotMatching(t *testing.T) {
	err := &Error{
		Category: CategoryValidation,
		Code:     "TEST",
		Message:  "test",
	}

	var target *int
	if err.As(&target) {
		t.Error("As() should return false for non-matching type")
	}
}

func TestWrap_NilError(t *testing.T) {
	wrapped := Wrap(nil, "context")
	if wrapped != nil {
		t.Error("Wrap(nil) should return nil")
	}
}

func TestWrap_ADKError(t *testing.T) {
	originalErr := ErrInvalidInput
	wrapped := Wrap(originalErr, "additional context")

	var adkErr *Error
	if !errors.As(wrapped, &adkErr) {
		t.Fatal("Wrapped error should be an ADK error")
	}
}

func TestWrap_StandardError(t *testing.T) {
	originalErr := errors.New("standard error")
	wrapped := Wrap(originalErr, "wrapped")

	var adkErr *Error
	if !errors.As(wrapped, &adkErr) {
		t.Fatal("Wrapped error should be an ADK error")
	}

	if adkErr.Code != "WRAPPED_ERROR" {
		t.Errorf("Code = %v, want WRAPPED_ERROR", adkErr.Code)
	}

	if adkErr.Category != CategoryInternal {
		t.Errorf("Category = %v, want %v", adkErr.Category, CategoryInternal)
	}
}

func TestCopyDetails_Nil(t *testing.T) {
	copied := copyDetails(nil)
	if copied != nil {
		t.Error("copyDetails(nil) should return nil")
	}
}

func TestCopyDetails_NotNil(t *testing.T) {
	original := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	copied := copyDetails(original)

	if len(copied) != len(original) {
		t.Errorf("copied length = %v, want %v", len(copied), len(original))
	}

	// Modify original, copied should not be affected
	original["key3"] = "value3"

	if _, exists := copied["key3"]; exists {
		t.Error("Copied map should not be affected by changes to original")
	}
}

func TestError_WithDetailsPreservesExisting(t *testing.T) {
	err := &Error{
		Category: CategoryValidation,
		Code:     "TEST",
		Message:  "test",
		Details: map[string]interface{}{
			"existing": "value",
		},
	}

	newErr := err.WithDetails(map[string]interface{}{
		"new": "value",
	})

	if newErr.Details["existing"] != "value" {
		t.Error("Existing details should be preserved")
	}

	if newErr.Details["new"] != "value" {
		t.Error("New details should be added")
	}
}

func TestIs_WithStandardError(t *testing.T) {
	stdErr := errors.New("standard error")
	adkErr := ErrInternal

	if Is(adkErr, stdErr) {
		t.Error("Is() should return false for different error types")
	}
}

func TestAs_WithStandardError(t *testing.T) {
	stdErr := errors.New("standard error")
	var target *Error

	if As(stdErr, &target) {
		t.Error("As() should return false for standard error")
	}
}
