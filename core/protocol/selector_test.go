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

package protocol

import (
	"testing"

	"github.com/sage-x-project/sage-adk/pkg/types"
)

func TestNewSelector(t *testing.T) {
	selector := NewSelector()

	if selector == nil {
		t.Fatal("NewSelector() should not return nil")
	}

	if selector.GetMode() != ProtocolAuto {
		t.Errorf("Default mode = %v, want %v", selector.GetMode(), ProtocolAuto)
	}
}

func TestSelector_Register(t *testing.T) {
	selector := NewSelector()
	adapter := NewMockAdapter("test")

	selector.Register(ProtocolA2A, adapter)

	// Select should use registered adapter for A2A
	selector.SetMode(ProtocolA2A)
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	selected := selector.Select(msg)
	if selected.Name() != adapter.Name() {
		t.Errorf("Select() adapter = %v, want %v", selected.Name(), adapter.Name())
	}
}

func TestSelector_SetMode(t *testing.T) {
	selector := NewSelector()

	selector.SetMode(ProtocolA2A)
	if selector.GetMode() != ProtocolA2A {
		t.Errorf("GetMode() = %v, want %v", selector.GetMode(), ProtocolA2A)
	}

	selector.SetMode(ProtocolSAGE)
	if selector.GetMode() != ProtocolSAGE {
		t.Errorf("GetMode() = %v, want %v", selector.GetMode(), ProtocolSAGE)
	}
}

func TestSelector_Select_Auto_SAGE(t *testing.T) {
	selector := NewSelector()
	a2aAdapter := NewMockAdapter("a2a")
	sageAdapter := NewMockAdapter("sage")

	selector.Register(ProtocolA2A, a2aAdapter)
	selector.Register(ProtocolSAGE, sageAdapter)
	selector.SetMode(ProtocolAuto)

	// Message with SAGE security
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)
	msg.Security = &types.SecurityMetadata{
		Mode: types.ProtocolModeSAGE,
	}

	selected := selector.Select(msg)
	if selected.Name() != sageAdapter.Name() {
		t.Errorf("Select() adapter = %v, want %v", selected.Name(), sageAdapter.Name())
	}
}

func TestSelector_Select_Auto_A2A(t *testing.T) {
	selector := NewSelector()
	a2aAdapter := NewMockAdapter("a2a")
	sageAdapter := NewMockAdapter("sage")

	selector.Register(ProtocolA2A, a2aAdapter)
	selector.Register(ProtocolSAGE, sageAdapter)
	selector.SetMode(ProtocolAuto)

	// Message without SAGE security
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	selected := selector.Select(msg)
	if selected.Name() != a2aAdapter.Name() {
		t.Errorf("Select() adapter = %v, want %v", selected.Name(), a2aAdapter.Name())
	}
}

func TestSelector_Select_Fixed_A2A(t *testing.T) {
	selector := NewSelector()
	a2aAdapter := NewMockAdapter("a2a")
	sageAdapter := NewMockAdapter("sage")

	selector.Register(ProtocolA2A, a2aAdapter)
	selector.Register(ProtocolSAGE, sageAdapter)
	selector.SetMode(ProtocolA2A)

	// Message with SAGE security but mode is fixed to A2A
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)
	msg.Security = &types.SecurityMetadata{
		Mode: types.ProtocolModeSAGE,
	}

	selected := selector.Select(msg)
	if selected.Name() != a2aAdapter.Name() {
		t.Errorf("Select() adapter = %v, want %v (fixed mode should override)", selected.Name(), a2aAdapter.Name())
	}
}

func TestSelector_Select_Fixed_SAGE(t *testing.T) {
	selector := NewSelector()
	a2aAdapter := NewMockAdapter("a2a")
	sageAdapter := NewMockAdapter("sage")

	selector.Register(ProtocolA2A, a2aAdapter)
	selector.Register(ProtocolSAGE, sageAdapter)
	selector.SetMode(ProtocolSAGE)

	// Message without SAGE security but mode is fixed to SAGE
	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	selected := selector.Select(msg)
	if selected.Name() != sageAdapter.Name() {
		t.Errorf("Select() adapter = %v, want %v (fixed mode should override)", selected.Name(), sageAdapter.Name())
	}
}

func TestSelector_Select_NotRegistered(t *testing.T) {
	selector := NewSelector()
	selector.SetMode(ProtocolA2A)

	msg := types.NewMessage(
		types.MessageRoleUser,
		[]types.Part{types.NewTextPart("test")},
	)

	selected := selector.Select(msg)
	if selected != nil {
		t.Error("Select() should return nil when adapter not registered")
	}
}
