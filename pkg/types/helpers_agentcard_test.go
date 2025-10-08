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

package types

import (
	"testing"
)

func TestNewAgentCard(t *testing.T) {
	name := "test-agent"
	description := "Test agent description"
	version := "1.0.0"

	card := NewAgentCard(name, description, version)

	if card == nil {
		t.Fatal("NewAgentCard() should not return nil")
	}

	if card.ID == "" {
		t.Error("AgentCard ID should not be empty")
	}

	if card.Name != name {
		t.Errorf("AgentCard Name = %v, want %v", card.Name, name)
	}

	if card.Description != description {
		t.Errorf("AgentCard Description = %v, want %v", card.Description, description)
	}

	if card.Version != version {
		t.Errorf("AgentCard Version = %v, want %v", card.Version, version)
	}

	if card.Capabilities == nil {
		t.Error("AgentCard Capabilities should not be nil")
	}

	if len(card.Capabilities) != 0 {
		t.Errorf("AgentCard Capabilities length = %v, want 0", len(card.Capabilities))
	}

	if card.Metadata == nil {
		t.Error("AgentCard Metadata should not be nil")
	}
}

func TestAgentCard_IDGeneration(t *testing.T) {
	card1 := NewAgentCard("agent1", "desc1", "1.0.0")
	card2 := NewAgentCard("agent2", "desc2", "2.0.0")

	if card1.ID == card2.ID {
		t.Error("NewAgentCard() should generate unique IDs")
	}

	// Check ID format
	if len(card1.ID) < 10 {
		t.Error("AgentCard ID should be a valid generated ID")
	}
}
