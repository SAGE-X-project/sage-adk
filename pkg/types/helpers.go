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
	"github.com/google/uuid"
)

// GenerateMessageID generates a new unique message ID.
func GenerateMessageID() string {
	return "msg-" + uuid.New().String()
}

// GenerateContextID generates a new unique context ID.
func GenerateContextID() string {
	return "ctx-" + uuid.New().String()
}

// GenerateTaskID generates a new unique task ID.
func GenerateTaskID() string {
	return "task-" + uuid.New().String()
}

// GenerateArtifactID generates a new unique artifact ID.
func GenerateArtifactID() string {
	return "artifact-" + uuid.New().String()
}

// GenerateAgentID generates a new unique agent ID.
func GenerateAgentID() string {
	return "agent-" + uuid.New().String()
}

// GenerateNonce generates a new unique nonce for security.
func GenerateNonce() string {
	return "nonce-" + uuid.New().String()
}

// NewTextPart creates a new TextPart containing the given text.
func NewTextPart(text string) *TextPart {
	return &TextPart{
		Kind:     string(PartKindText),
		Text:     text,
		Metadata: make(map[string]interface{}),
	}
}

// NewFilePartWithBytes creates a new FilePart with embedded bytes content.
func NewFilePartWithBytes(name, mimeType string, bytes []byte) *FilePart {
	return &FilePart{
		Kind: string(PartKindFile),
		File: &FileWithBytes{
			Name:     name,
			MimeType: mimeType,
			Bytes:    bytes,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewFilePartWithURI creates a new FilePart with URI reference.
func NewFilePartWithURI(name, mimeType string, uri string) *FilePart {
	return &FilePart{
		Kind: string(PartKindFile),
		File: &FileWithURI{
			Name:     name,
			MimeType: mimeType,
			URI:      uri,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewDataPart creates a new DataPart with the given data.
func NewDataPart(data interface{}) *DataPart {
	return &DataPart{
		Kind:     string(PartKindData),
		Data:     data,
		Metadata: make(map[string]interface{}),
	}
}

// NewMessage creates a new Message with the specified role and parts.
func NewMessage(role MessageRole, parts []Part) *Message {
	return &Message{
		MessageID: GenerateMessageID(),
		Role:      role,
		Parts:     parts,
		Kind:      "message",
		Metadata:  make(map[string]interface{}),
	}
}

// NewMessageWithContext creates a new Message with context information.
func NewMessageWithContext(role MessageRole, parts []Part, taskID, contextID *string) *Message {
	return &Message{
		MessageID: GenerateMessageID(),
		Role:      role,
		Parts:     parts,
		Kind:      "message",
		TaskID:    taskID,
		ContextID: contextID,
		Metadata:  make(map[string]interface{}),
	}
}

// NewArtifact creates a new Artifact with a generated ID.
func NewArtifact(name, description string, parts []Part) *Artifact {
	return &Artifact{
		ArtifactID:  GenerateArtifactID(),
		Name:        name,
		Description: description,
		Parts:       parts,
		Metadata:    make(map[string]interface{}),
	}
}
