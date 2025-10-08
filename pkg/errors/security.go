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

// Security errors
var (
	// ErrSignatureInvalid indicates signature verification failed.
	ErrSignatureInvalid = &Error{
		Category: CategorySecurity,
		Code:     "INVALID_SIGNATURE",
		Message:  "signature verification failed",
	}

	// ErrDIDNotFound indicates DID was not found in registry.
	ErrDIDNotFound = &Error{
		Category: CategorySecurity,
		Code:     "DID_NOT_FOUND",
		Message:  "DID not found in registry",
	}

	// ErrAgentInactive indicates the agent is deactivated.
	ErrAgentInactive = &Error{
		Category: CategorySecurity,
		Code:     "AGENT_INACTIVE",
		Message:  "agent is deactivated",
	}

	// ErrUnauthorized indicates unauthorized access.
	ErrUnauthorized = &Error{
		Category: CategoryUnauthorized,
		Code:     "UNAUTHORIZED",
		Message:  "unauthorized access",
	}

	// ErrInvalidCredentials indicates invalid credentials.
	ErrInvalidCredentials = &Error{
		Category: CategoryUnauthorized,
		Code:     "INVALID_CREDENTIALS",
		Message:  "invalid credentials provided",
	}
)
