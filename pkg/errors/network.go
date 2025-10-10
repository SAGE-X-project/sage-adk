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

// Network errors
var (
	// ErrNetworkTimeout indicates network request timed out.
	ErrNetworkTimeout = &Error{
		Category: CategoryNetwork,
		Code:     "NETWORK_TIMEOUT",
		Message:  "network request timed out",
	}

	// ErrTimeout is an alias for ErrNetworkTimeout for convenience.
	ErrTimeout = ErrNetworkTimeout

	// ErrNetworkUnavailable indicates network is unavailable.
	ErrNetworkUnavailable = &Error{
		Category: CategoryNetwork,
		Code:     "NETWORK_UNAVAILABLE",
		Message:  "network unavailable",
	}

	// ErrConnectionRefused indicates connection was refused.
	ErrConnectionRefused = &Error{
		Category: CategoryNetwork,
		Code:     "CONNECTION_REFUSED",
		Message:  "connection refused",
	}

	// ErrRateLimitExceeded indicates rate limit has been exceeded.
	ErrRateLimitExceeded = &Error{
		Category: CategoryNetwork,
		Code:     "RATE_LIMIT_EXCEEDED",
		Message:  "rate limit exceeded",
	}
)
