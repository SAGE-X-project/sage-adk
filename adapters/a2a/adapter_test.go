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

package a2a

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestNewAdapter_Success(t *testing.T) {
	cfg := &config.A2AConfig{
		ServerURL: "http://localhost:8080/",
		Timeout:   30,
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	if adapter == nil {
		t.Fatal("NewAdapter() should not return nil")
	}

	if adapter.Name() != "a2a" {
		t.Errorf("Name() = %v, want a2a", adapter.Name())
	}
}

func TestNewAdapter_InvalidURL(t *testing.T) {
	cfg := &config.A2AConfig{
		ServerURL: "://invalid",
		Timeout:   30,
	}

	_, err := NewAdapter(cfg)
	if err == nil {
		t.Error("NewAdapter() should return error for invalid URL")
	}
}

func TestAdapter_Name(t *testing.T) {
	cfg := &config.A2AConfig{
		ServerURL: "http://localhost:8080/",
	}

	adapter, _ := NewAdapter(cfg)
	if adapter.Name() != "a2a" {
		t.Errorf("Name() = %v, want a2a", adapter.Name())
	}
}

func TestAdapter_SupportsStreaming(t *testing.T) {
	cfg := &config.A2AConfig{
		ServerURL: "http://localhost:8080/",
	}

	adapter, _ := NewAdapter(cfg)
	if !adapter.SupportsStreaming() {
		t.Error("SupportsStreaming() = false, want true")
	}
}

func TestAdapter_Verify(t *testing.T) {
	cfg := &config.A2AConfig{
		ServerURL: "http://localhost:8080/",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// Verify should always return nil for A2A (handled by client)
	err := adapter.Verify(context.Background(), msg)
	if err != nil {
		t.Errorf("Verify() error = %v, want nil", err)
	}
}

func TestAdapter_ReceiveMessage(t *testing.T) {
	cfg := &config.A2AConfig{
		ServerURL: "http://localhost:8080/",
	}

	adapter, _ := NewAdapter(cfg)

	// ReceiveMessage should return ErrNotImplemented for A2A
	_, err := adapter.ReceiveMessage(context.Background())
	if err == nil {
		t.Error("ReceiveMessage() should return error for A2A")
	}
}
