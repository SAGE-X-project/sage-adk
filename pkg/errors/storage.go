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

// Storage errors
var (
	// ErrNotFound indicates resource was not found in storage.
	ErrNotFound = &Error{
		Category: CategoryNotFound,
		Code:     "NOT_FOUND",
		Message:  "resource not found in storage",
	}

	// ErrStorageConnection indicates storage connection failed.
	ErrStorageConnection = &Error{
		Category: CategoryStorage,
		Code:     "CONNECTION_ERROR",
		Message:  "storage connection failed",
	}

	// ErrStorageTimeout indicates storage operation timed out.
	ErrStorageTimeout = &Error{
		Category: CategoryStorage,
		Code:     "TIMEOUT",
		Message:  "storage operation timed out",
	}

	// ErrAlreadyExists indicates the resource already exists.
	ErrAlreadyExists = &Error{
		Category: CategoryStorage,
		Code:     "ALREADY_EXISTS",
		Message:  "resource already exists",
	}
)
