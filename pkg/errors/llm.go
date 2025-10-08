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

// LLM provider errors
var (
	// ErrLLMConnection indicates failed to connect to LLM provider.
	ErrLLMConnection = &Error{
		Category: CategoryLLM,
		Code:     "LLM_CONNECTION_ERROR",
		Message:  "failed to connect to LLM provider",
	}

	// ErrLLMRateLimit indicates LLM rate limit was exceeded.
	ErrLLMRateLimit = &Error{
		Category: CategoryLLM,
		Code:     "RATE_LIMIT_EXCEEDED",
		Message:  "LLM rate limit exceeded",
	}

	// ErrLLMInvalidResponse indicates invalid response from LLM.
	ErrLLMInvalidResponse = &Error{
		Category: CategoryLLM,
		Code:     "INVALID_RESPONSE",
		Message:  "invalid response from LLM",
	}

	// ErrLLMTimeout indicates LLM request timed out.
	ErrLLMTimeout = &Error{
		Category: CategoryLLM,
		Code:     "LLM_TIMEOUT",
		Message:  "LLM request timed out",
	}
)
