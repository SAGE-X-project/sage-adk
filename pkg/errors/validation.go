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

// Validation errors
var (
	// ErrInvalidInput indicates invalid input was provided.
	ErrInvalidInput = &Error{
		Category: CategoryValidation,
		Code:     "INVALID_INPUT",
		Message:  "invalid input provided",
	}

	// ErrMissingField indicates a required field is missing.
	ErrMissingField = &Error{
		Category: CategoryValidation,
		Code:     "MISSING_FIELD",
		Message:  "required field is missing",
	}

	// ErrInvalidFormat indicates invalid format.
	ErrInvalidFormat = &Error{
		Category: CategoryValidation,
		Code:     "INVALID_FORMAT",
		Message:  "invalid format",
	}

	// ErrInvalidValue indicates an invalid value.
	ErrInvalidValue = &Error{
		Category: CategoryValidation,
		Code:     "INVALID_VALUE",
		Message:  "invalid value",
	}

	// ErrOutOfRange indicates a value is out of valid range.
	ErrOutOfRange = &Error{
		Category: CategoryValidation,
		Code:     "OUT_OF_RANGE",
		Message:  "value out of valid range",
	}
)
