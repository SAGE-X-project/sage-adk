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
	"fmt"
)

// ErrorCategory represents the category of an error.
type ErrorCategory string

const (
	// CategoryValidation indicates a validation error.
	CategoryValidation ErrorCategory = "validation"
	// CategoryProtocol indicates a protocol-related error.
	CategoryProtocol ErrorCategory = "protocol"
	// CategorySecurity indicates a security-related error.
	CategorySecurity ErrorCategory = "security"
	// CategoryStorage indicates a storage-related error.
	CategoryStorage ErrorCategory = "storage"
	// CategoryLLM indicates an LLM provider error.
	CategoryLLM ErrorCategory = "llm"
	// CategoryNetwork indicates a network error.
	CategoryNetwork ErrorCategory = "network"
	// CategoryInternal indicates an internal error.
	CategoryInternal ErrorCategory = "internal"
	// CategoryNotFound indicates a resource not found error.
	CategoryNotFound ErrorCategory = "not_found"
	// CategoryUnauthorized indicates an authorization error.
	CategoryUnauthorized ErrorCategory = "unauthorized"
)

// Error represents a structured error in SAGE ADK.
type Error struct {
	// Category is the error category.
	Category ErrorCategory
	// Code is the machine-readable error code.
	Code string
	// Message is the human-readable error message.
	Message string
	// Details contains additional error context.
	Details map[string]interface{}
	// Err is the wrapped error.
	Err error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s: %v", e.Category, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Category, e.Code, e.Message)
}

// Unwrap returns the wrapped error for Go 1.13+ error unwrapping.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target error.
// Two errors match if they have the same Code.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// As checks if the error can be assigned to target.
func (e *Error) As(target interface{}) bool {
	t, ok := target.(**Error)
	if !ok {
		return false
	}
	*t = e
	return true
}

// WithMessage returns a new error with additional message context.
func (e *Error) WithMessage(msg string) *Error {
	return &Error{
		Category: e.Category,
		Code:     e.Code,
		Message:  fmt.Sprintf("%s: %s", e.Message, msg),
		Details:  copyDetails(e.Details),
		Err:      e.Err,
	}
}

// WithDetail returns a new error with an additional detail field.
func (e *Error) WithDetail(key string, value interface{}) *Error {
	details := copyDetails(e.Details)
	if details == nil {
		details = make(map[string]interface{})
	}
	details[key] = value

	return &Error{
		Category: e.Category,
		Code:     e.Code,
		Message:  e.Message,
		Details:  details,
		Err:      e.Err,
	}
}

// WithDetails returns a new error with additional details.
func (e *Error) WithDetails(details map[string]interface{}) *Error {
	newDetails := copyDetails(e.Details)
	if newDetails == nil {
		newDetails = make(map[string]interface{})
	}

	for k, v := range details {
		newDetails[k] = v
	}

	return &Error{
		Category: e.Category,
		Code:     e.Code,
		Message:  e.Message,
		Details:  newDetails,
		Err:      e.Err,
	}
}

// Wrap wraps an existing error with this error as context.
func (e *Error) Wrap(err error) *Error {
	return &Error{
		Category: e.Category,
		Code:     e.Code,
		Message:  e.Message,
		Details:  copyDetails(e.Details),
		Err:      err,
	}
}

// copyDetails creates a shallow copy of the details map.
func copyDetails(details map[string]interface{}) map[string]interface{} {
	if details == nil {
		return nil
	}

	copied := make(map[string]interface{}, len(details))
	for k, v := range details {
		copied[k] = v
	}
	return copied
}

// New creates a new Error with the specified category, code, and message.
func New(category ErrorCategory, code, message string) *Error {
	return &Error{
		Category: category,
		Code:     code,
		Message:  message,
	}
}

// Wrap wraps an existing error with a message.
// If err is already an ADK Error, it returns a new Error wrapping it.
// Otherwise, it creates a new internal error wrapping the original error.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	var adkErr *Error
	if errors.As(err, &adkErr) {
		return adkErr.WithMessage(message)
	}

	return &Error{
		Category: CategoryInternal,
		Code:     "WRAPPED_ERROR",
		Message:  message,
		Err:      err,
	}
}

// Is reports whether any error in err's chain matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// IsCategory checks if an error belongs to a specific category.
func IsCategory(err error, category ErrorCategory) bool {
	var adkErr *Error
	if errors.As(err, &adkErr) {
		return adkErr.Category == category
	}
	return false
}

// IsInvalidInput checks if an error is an invalid input error.
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput) || IsCategory(err, CategoryValidation)
}

// IsUnauthorized checks if an error is an unauthorized error.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized) || IsCategory(err, CategoryUnauthorized)
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || IsCategory(err, CategoryNotFound)
}

// IsRateLimitExceeded checks if an error is a rate limit exceeded error.
func IsRateLimitExceeded(err error) bool {
	return errors.Is(err, ErrRateLimitExceeded)
}

// IsTimeout checks if an error is a timeout error.
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout) || errors.Is(err, ErrNetworkTimeout)
}
