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

package resilience

import (
	"context"
	"testing"
	"time"
)

func TestWithTimeout_Success(t *testing.T) {
	config := &TimeoutConfig{
		Duration: 100 * time.Millisecond,
	}

	executed := false
	err := WithTimeout(context.Background(), config, func(ctx context.Context) error {
		executed = true
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("WithTimeout() error = %v, want nil", err)
	}
	if !executed {
		t.Error("function should be executed")
	}
}

func TestWithTimeout_Timeout(t *testing.T) {
	config := &TimeoutConfig{
		Duration: 50 * time.Millisecond,
	}

	err := WithTimeout(context.Background(), config, func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if err != ErrTimeout {
		t.Errorf("WithTimeout() error = %v, want ErrTimeout", err)
	}
}

func TestWithTimeout_DefaultConfig(t *testing.T) {
	executed := false
	err := WithTimeout(context.Background(), nil, func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("WithTimeout() error = %v, want nil", err)
	}
	if !executed {
		t.Error("function should be executed")
	}
}

func TestWithTimeout_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := &TimeoutConfig{
		Duration: 1 * time.Second,
	}

	// Cancel immediately
	cancel()

	err := WithTimeout(ctx, config, func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	if err != context.Canceled {
		t.Errorf("WithTimeout() error = %v, want context.Canceled", err)
	}
}
