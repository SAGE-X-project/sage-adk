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

package sage

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage-adk/config"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestNewAdapter_Success(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, err := NewAdapter(cfg)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	if adapter == nil {
		t.Fatal("NewAdapter() should not return nil")
	}

	if adapter.Name() != "sage" {
		t.Errorf("Name() = %v, want sage", adapter.Name())
	}
}

func TestNewAdapter_MissingDID(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "",
	}

	_, err := NewAdapter(cfg)
	if err == nil {
		t.Error("NewAdapter() should return error when DID is missing")
	}
}

func TestAdapter_Name(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	if adapter.Name() != "sage" {
		t.Errorf("Name() = %v, want sage", adapter.Name())
	}
}

func TestAdapter_SupportsStreaming(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)

	// Streaming not supported in Phase 1
	if adapter.SupportsStreaming() {
		t.Error("SupportsStreaming() should return false in Phase 1")
	}
}

func TestAdapter_SendMessage_NotImplemented(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// SendMessage not implemented in Phase 1 (no transport layer)
	err := adapter.SendMessage(context.Background(), msg)
	if err == nil {
		t.Error("SendMessage() should return error (not implemented)")
	}
}

func TestAdapter_ReceiveMessage_NotImplemented(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)

	// ReceiveMessage not implemented in Phase 1
	_, err := adapter.ReceiveMessage(context.Background())
	if err == nil {
		t.Error("ReceiveMessage() should return error (not implemented)")
	}
}

func TestAdapter_Verify_MissingSecurityMetadata(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	// Message without security metadata should fail
	err := adapter.Verify(context.Background(), msg)
	if err == nil {
		t.Error("Verify() should return error when security metadata is missing")
	}
}

func TestAdapter_Stream_NotImplemented(t *testing.T) {
	cfg := &config.SAGEConfig{
		Enabled: true,
		Network: "ethereum",
		DID:     "did:sage:eth:0x1234567890abcdef",
	}

	adapter, _ := NewAdapter(cfg)

	fn := func(chunk string) error {
		return nil
	}

	// Streaming not implemented in Phase 1
	err := adapter.Stream(context.Background(), fn)
	if err == nil {
		t.Error("Stream() should return error (not implemented)")
	}
}
