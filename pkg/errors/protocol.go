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

// Protocol errors
var (
	// ErrProtocolMismatch indicates a protocol mismatch.
	ErrProtocolMismatch = &Error{
		Category: CategoryProtocol,
		Code:     "PROTOCOL_MISMATCH",
		Message:  "protocol mismatch",
	}

	// ErrUnsupportedProtocol indicates an unsupported protocol.
	ErrUnsupportedProtocol = &Error{
		Category: CategoryProtocol,
		Code:     "UNSUPPORTED_PROTOCOL",
		Message:  "protocol not supported",
	}

	// ErrMessageParsing indicates message parsing failed.
	ErrMessageParsing = &Error{
		Category: CategoryProtocol,
		Code:     "MESSAGE_PARSING_ERROR",
		Message:  "failed to parse message",
	}

	// ErrInvalidProtocolVersion indicates an invalid protocol version.
	ErrInvalidProtocolVersion = &Error{
		Category: CategoryProtocol,
		Code:     "INVALID_PROTOCOL_VERSION",
		Message:  "invalid protocol version",
	}
)
