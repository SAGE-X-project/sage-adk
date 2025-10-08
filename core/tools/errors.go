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

package tools

import "errors"

var (
	// ErrNilTool is returned when trying to register a nil tool.
	ErrNilTool = errors.New("tool cannot be nil")

	// ErrEmptyToolName is returned when a tool has an empty name.
	ErrEmptyToolName = errors.New("tool name cannot be empty")

	// ErrToolAlreadyExists is returned when trying to register a duplicate tool.
	ErrToolAlreadyExists = errors.New("tool already exists")

	// ErrToolNotFound is returned when a tool is not found in the registry.
	ErrToolNotFound = errors.New("tool not found")

	// ErrInvalidParameters is returned when tool parameters are invalid.
	ErrInvalidParameters = errors.New("invalid tool parameters")

	// ErrExecutionFailed is returned when tool execution fails.
	ErrExecutionFailed = errors.New("tool execution failed")
)
