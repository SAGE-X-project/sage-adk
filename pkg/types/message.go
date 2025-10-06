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
	"encoding/json"
	"fmt"
)

// MessageRole indicates the originator of a message (user or agent).
type MessageRole string

const (
	// MessageRoleUser is the role of a message originated from the user/client.
	MessageRoleUser MessageRole = "user"
	// MessageRoleAgent is the role of a message originated from the agent/server.
	MessageRoleAgent MessageRole = "agent"
)

// IsValid checks if the message role is valid.
func (r MessageRole) IsValid() bool {
	return r == MessageRoleUser || r == MessageRoleAgent
}

// Part is an interface representing a segment of a message (text, file, or data).
type Part interface {
	GetKind() string
	Validate() error
}

// PartKind represents the type of message part.
type PartKind string

const (
	// PartKindText is the kind of text part.
	PartKindText PartKind = "text"
	// PartKindFile is the kind of file part.
	PartKindFile PartKind = "file"
	// PartKindData is the kind of data part.
	PartKindData PartKind = "data"
)

// TextPart represents a text segment within a message.
type TextPart struct {
	// Kind is the type of the part (always "text").
	Kind string `json:"kind"`
	// Text is the text content.
	Text string `json:"text"`
	// Metadata is optional metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GetKind returns the kind of the part.
func (t *TextPart) GetKind() string {
	return string(PartKindText)
}

// Validate validates the text part.
func (t *TextPart) Validate() error {
	if t.Kind != string(PartKindText) {
		return fmt.Errorf("kind must be 'text', got '%s'", t.Kind)
	}
	if t.Text == "" {
		return fmt.Errorf("text cannot be empty")
	}
	return nil
}

// FileContent is an interface for file content (bytes or URI).
type FileContent interface {
	GetName() string
	GetMimeType() string
	Validate() error
}

// FileWithBytes represents file data with embedded content.
type FileWithBytes struct {
	// Name is the filename.
	Name string `json:"name,omitempty"`
	// MimeType is the MIME type.
	MimeType string `json:"mimeType,omitempty"`
	// Bytes is the file content.
	Bytes []byte `json:"bytes"`
}

// GetName returns the filename.
func (f *FileWithBytes) GetName() string {
	return f.Name
}

// GetMimeType returns the MIME type.
func (f *FileWithBytes) GetMimeType() string {
	return f.MimeType
}

// Validate validates the file with bytes.
func (f *FileWithBytes) Validate() error {
	if len(f.Bytes) == 0 {
		return fmt.Errorf("bytes cannot be empty")
	}
	return nil
}

// FileWithURI represents file data with URI reference.
type FileWithURI struct {
	// Name is the filename.
	Name string `json:"name,omitempty"`
	// MimeType is the MIME type.
	MimeType string `json:"mimeType,omitempty"`
	// URI is the URI pointing to the content.
	URI string `json:"uri"`
}

// GetName returns the filename.
func (f *FileWithURI) GetName() string {
	return f.Name
}

// GetMimeType returns the MIME type.
func (f *FileWithURI) GetMimeType() string {
	return f.MimeType
}

// Validate validates the file with URI.
func (f *FileWithURI) Validate() error {
	if f.URI == "" {
		return fmt.Errorf("URI cannot be empty")
	}
	return nil
}

// FilePart represents a file included in a message.
type FilePart struct {
	// Kind is the type of the part (always "file").
	Kind string `json:"kind"`
	// File is the file content (FileWithBytes or FileWithURI).
	File FileContent `json:"file"`
	// Metadata is optional metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GetKind returns the kind of the part.
func (f *FilePart) GetKind() string {
	return string(PartKindFile)
}

// Validate validates the file part.
func (f *FilePart) Validate() error {
	if f.Kind != string(PartKindFile) {
		return fmt.Errorf("kind must be 'file', got '%s'", f.Kind)
	}
	if f.File == nil {
		return fmt.Errorf("file content is required")
	}
	return f.File.Validate()
}

// MarshalJSON implements custom JSON marshaling for FilePart.
func (f *FilePart) MarshalJSON() ([]byte, error) {
	type Alias FilePart
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(f),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for FilePart.
func (f *FilePart) UnmarshalJSON(data []byte) error {
	type Alias FilePart
	temp := &struct {
		File json.RawMessage `json:"file"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal file part: %w", err)
	}

	// Determine file content type
	var fileContent map[string]interface{}
	if err := json.Unmarshal(temp.File, &fileContent); err != nil {
		return fmt.Errorf("failed to unmarshal file content: %w", err)
	}

	if _, hasBytes := fileContent["bytes"]; hasBytes {
		var fileWithBytes FileWithBytes
		if err := json.Unmarshal(temp.File, &fileWithBytes); err != nil {
			return fmt.Errorf("failed to unmarshal FileWithBytes: %w", err)
		}
		f.File = &fileWithBytes
	} else if _, hasURI := fileContent["uri"]; hasURI {
		var fileWithURI FileWithURI
		if err := json.Unmarshal(temp.File, &fileWithURI); err != nil {
			return fmt.Errorf("failed to unmarshal FileWithURI: %w", err)
		}
		f.File = &fileWithURI
	} else {
		return fmt.Errorf("unknown file type: must have either 'bytes' or 'uri' field")
	}

	return nil
}

// DataPart represents arbitrary structured data (JSON) within a message.
type DataPart struct {
	// Kind is the type of the part (always "data").
	Kind string `json:"kind"`
	// Data is the actual data payload.
	Data interface{} `json:"data"`
	// Metadata is optional metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GetKind returns the kind of the part.
func (d *DataPart) GetKind() string {
	return string(PartKindData)
}

// Validate validates the data part.
func (d *DataPart) Validate() error {
	if d.Kind != string(PartKindData) {
		return fmt.Errorf("kind must be 'data', got '%s'", d.Kind)
	}
	if d.Data == nil {
		return fmt.Errorf("data cannot be nil")
	}
	return nil
}

// Message represents a single exchange between a user and an agent.
// Supports both A2A and SAGE protocols.
type Message struct {
	// MessageID is the unique identifier for this message.
	MessageID string `json:"messageId"`
	// ContextID is the optional context identifier for the message.
	ContextID *string `json:"contextId,omitempty"`
	// Role is the sender of the message (user or agent).
	Role MessageRole `json:"role"`
	// Parts is the content parts.
	Parts []Part `json:"parts"`
	// Kind is the type discriminator (always "message").
	Kind string `json:"kind"`
	// Metadata is optional metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// A2A Optional Fields
	// TaskID is the optional task identifier this message belongs to.
	TaskID *string `json:"taskId,omitempty"`
	// ReferenceTaskIDs is the optional list of referenced task IDs.
	ReferenceTaskIDs []string `json:"referenceTaskIds,omitempty"`
	// Extensions is the optional list of extension URIs.
	Extensions []string `json:"extensions,omitempty"`

	// SAGE Security Fields (optional)
	// Security contains SAGE security metadata.
	Security *SecurityMetadata `json:"security,omitempty"`
}

// Validate validates the message.
func (m *Message) Validate() error {
	if m.MessageID == "" {
		return fmt.Errorf("MessageID is required")
	}
	if !m.Role.IsValid() {
		return fmt.Errorf("invalid role: %s", m.Role)
	}
	if len(m.Parts) == 0 {
		return fmt.Errorf("at least one part is required")
	}

	// Validate each part
	for i, part := range m.Parts {
		if err := part.Validate(); err != nil {
			return fmt.Errorf("part %d validation failed: %w", i, err)
		}
	}

	// Validate security metadata if present
	if m.Security != nil {
		if err := m.Security.Validate(); err != nil {
			return fmt.Errorf("security validation failed: %w", err)
		}
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Message.
func (m *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Message.
func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	temp := &struct {
		Parts []json.RawMessage `json:"parts"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Unmarshal parts
	m.Parts = make([]Part, 0, len(temp.Parts))
	for i, rawPart := range temp.Parts {
		part, err := unmarshalPart(rawPart)
		if err != nil {
			return fmt.Errorf("failed to unmarshal part %d: %w", i, err)
		}
		m.Parts = append(m.Parts, part)
	}

	return nil
}

// unmarshalPart determines the concrete type of a Part from raw JSON.
func unmarshalPart(rawPart json.RawMessage) (Part, error) {
	var typeDetect struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(rawPart, &typeDetect); err != nil {
		return nil, fmt.Errorf("failed to detect part type: %w", err)
	}

	switch typeDetect.Kind {
	case string(PartKindText):
		var p TextPart
		if err := json.Unmarshal(rawPart, &p); err != nil {
			return nil, fmt.Errorf("failed to unmarshal TextPart: %w", err)
		}
		return &p, nil
	case string(PartKindFile):
		var p FilePart
		if err := json.Unmarshal(rawPart, &p); err != nil {
			return nil, fmt.Errorf("failed to unmarshal FilePart: %w", err)
		}
		return &p, nil
	case string(PartKindData):
		var p DataPart
		if err := json.Unmarshal(rawPart, &p); err != nil {
			return nil, fmt.Errorf("failed to unmarshal DataPart: %w", err)
		}
		return &p, nil
	default:
		return nil, fmt.Errorf("unsupported part kind: %s", typeDetect.Kind)
	}
}
