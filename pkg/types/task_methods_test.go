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

func TestTask_AddArtifact(t *testing.T) {
	task := NewTask("task-123", "ctx-456")
	initialLength := len(task.Artifacts)

	artifact := *NewArtifact("result.txt", "Test result", []Part{
		NewTextPart("Result content"),
	})

	task.AddArtifact(artifact)

	if len(task.Artifacts) != initialLength+1 {
		t.Errorf("Artifacts length = %v, want %v", len(task.Artifacts), initialLength+1)
	}

	if task.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestTask_AddHistoryMessage(t *testing.T) {
	task := NewTask("task-123", "ctx-456")
	initialLength := len(task.History)

	msg := *NewMessage(MessageRoleUser, []Part{
		NewTextPart("Hello"),
	})

	task.AddHistoryMessage(msg)

	if len(task.History) != initialLength+1 {
		t.Errorf("History length = %v, want %v", len(task.History), initialLength+1)
	}

	if task.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestTaskStatusUpdateEvent_IsFinal(t *testing.T) {
	tests := []struct {
		name  string
		event *TaskStatusUpdateEvent
		want  bool
	}{
		{
			name: "final event",
			event: &TaskStatusUpdateEvent{
				Final: true,
			},
			want: true,
		},
		{
			name: "non-final event",
			event: &TaskStatusUpdateEvent{
				Final: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsFinal(); got != tt.want {
				t.Errorf("IsFinal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskArtifactUpdateEvent_IsFinal(t *testing.T) {
	tests := []struct {
		name  string
		event *TaskArtifactUpdateEvent
		want  bool
	}{
		{
			name: "final chunk",
			event: &TaskArtifactUpdateEvent{
				LastChunk: boolPtr(true),
			},
			want: true,
		},
		{
			name: "non-final chunk",
			event: &TaskArtifactUpdateEvent{
				LastChunk: boolPtr(false),
			},
			want: false,
		},
		{
			name: "nil lastChunk",
			event: &TaskArtifactUpdateEvent{
				LastChunk: nil,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsFinal(); got != tt.want {
				t.Errorf("IsFinal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestFileWithBytes_Validate(t *testing.T) {
	tests := []struct {
		name    string
		file    *FileWithBytes
		wantErr bool
	}{
		{
			name: "valid",
			file: &FileWithBytes{
				Name:     "test.txt",
				MimeType: "text/plain",
				Bytes:    []byte("content"),
			},
			wantErr: false,
		},
		{
			name: "empty bytes",
			file: &FileWithBytes{
				Name:     "test.txt",
				MimeType: "text/plain",
				Bytes:    []byte{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.file.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileWithURI_Validate(t *testing.T) {
	tests := []struct {
		name    string
		file    *FileWithURI
		wantErr bool
	}{
		{
			name: "valid",
			file: &FileWithURI{
				Name:     "test.txt",
				MimeType: "text/plain",
				URI:      "https://example.com/file.txt",
			},
			wantErr: false,
		},
		{
			name: "empty URI",
			file: &FileWithURI{
				Name:     "test.txt",
				MimeType: "text/plain",
				URI:      "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.file.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileContent_GetMethods(t *testing.T) {
	t.Run("FileWithBytes", func(t *testing.T) {
		file := &FileWithBytes{
			Name:     "test.txt",
			MimeType: "text/plain",
			Bytes:    []byte("content"),
		}

		if file.GetName() != "test.txt" {
			t.Errorf("GetName() = %v, want test.txt", file.GetName())
		}
		if file.GetMimeType() != "text/plain" {
			t.Errorf("GetMimeType() = %v, want text/plain", file.GetMimeType())
		}
	})

	t.Run("FileWithURI", func(t *testing.T) {
		file := &FileWithURI{
			Name:     "test.txt",
			MimeType: "text/plain",
			URI:      "https://example.com/file.txt",
		}

		if file.GetName() != "test.txt" {
			t.Errorf("GetName() = %v, want test.txt", file.GetName())
		}
		if file.GetMimeType() != "text/plain" {
			t.Errorf("GetMimeType() = %v, want text/plain", file.GetMimeType())
		}
	})
}
