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

// Internal errors
var (
	// ErrInternal indicates an internal server error.
	ErrInternal = &Error{
		Category: CategoryInternal,
		Code:     "INTERNAL_ERROR",
		Message:  "internal server error",
	}

	// ErrNotImplemented indicates a feature is not implemented.
	ErrNotImplemented = &Error{
		Category: CategoryInternal,
		Code:     "NOT_IMPLEMENTED",
		Message:  "feature not implemented",
	}

	// ErrConfigurationError indicates a configuration error.
	ErrConfigurationError = &Error{
		Category: CategoryInternal,
		Code:     "CONFIGURATION_ERROR",
		Message:  "configuration error",
	}
)
