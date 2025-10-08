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

package state

import "errors"

var (
	// ErrStateNotFound is returned when a state is not found.
	ErrStateNotFound = errors.New("state not found")

	// ErrStateExists is returned when trying to create a state that already exists.
	ErrStateExists = errors.New("state already exists")

	// ErrInvalidSessionID is returned when the session ID is invalid.
	ErrInvalidSessionID = errors.New("invalid session ID")

	// ErrInvalidAgentID is returned when the agent ID is invalid.
	ErrInvalidAgentID = errors.New("invalid agent ID")

	// ErrStateExpired is returned when a state has expired.
	ErrStateExpired = errors.New("state expired")

	// ErrVariableNotFound is returned when a variable is not found.
	ErrVariableNotFound = errors.New("variable not found")

	// ErrMaxMessagesExceeded is returned when the maximum number of messages is exceeded.
	ErrMaxMessagesExceeded = errors.New("maximum number of messages exceeded")
)
