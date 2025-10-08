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

// Package errors provides structured error handling for SAGE ADK.
//
// The package defines a comprehensive error system with:
//
//   - Categorized errors for different domains
//   - Rich error context with details
//   - Standard Go error wrapping support
//   - Type-safe error checking
//
// # Error Categories
//
// Errors are organized into categories:
//
//   - Validation: Input validation errors
//   - Protocol: A2A/SAGE protocol errors
//   - Security: Authentication and authorization errors
//   - Storage: Database and storage errors
//   - LLM: LLM provider errors
//   - Network: Network communication errors
//   - Internal: Internal server errors
//
// # Creating Errors
//
// Use predefined errors:
//
//	err := errors.ErrInvalidInput.WithDetail("field", "messageId")
//
// Or create custom errors:
//
//	err := errors.New(
//	    errors.CategoryValidation,
//	    "CUSTOM_ERROR",
//	    "custom error message",
//	)
//
// # Wrapping Errors
//
// Wrap errors to add context:
//
//	if err := validateMessage(msg); err != nil {
//	    return errors.ErrInvalidInput.
//	        WithMessage("message validation failed").
//	        Wrap(err)
//	}
//
// # Error Checking
//
// Check error types using standard Go patterns:
//
//	// Check if error matches a specific type
//	if errors.Is(err, errors.ErrNotFound) {
//	    // handle not found
//	}
//
//	// Extract error details
//	var adkErr *errors.Error
//	if errors.As(err, &adkErr) {
//	    log.Printf("Code: %s, Details: %v", adkErr.Code, adkErr.Details)
//	}
package errors
