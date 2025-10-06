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
	"strings"
	"testing"
	"time"
)

func TestTaskState_IsTerminal(t *testing.T) {
	tests := []struct {
		name  string
		state TaskState
		want  bool
	}{
		{"completed is terminal", TaskStateCompleted, true},
		{"canceled is terminal", TaskStateCanceled, true},
		{"failed is terminal", TaskStateFailed, true},
		{"rejected is terminal", TaskStateRejected, true},
		{"submitted is not terminal", TaskStateSubmitted, false},
		{"working is not terminal", TaskStateWorking, false},
		{"input-required is not terminal", TaskStateInputRequired, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.want {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArtifact_Validate(t *testing.T) {
	tests := []struct {
		name     string
		artifact *Artifact
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid artifact",
			artifact: &Artifact{
				ArtifactID: "artifact-123",
				Name:       "result.txt",
				Parts: []Part{
					&TextPart{
						Kind: string(PartKindText),
						Text: "Result content",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty artifact ID",
			artifact: &Artifact{
				ArtifactID: "",
				Name:       "result.txt",
				Parts: []Part{
					&TextPart{
						Kind: string(PartKindText),
						Text: "Result content",
					},
				},
			},
			wantErr: true,
			errMsg:  "ArtifactID is required",
		},
		{
			name: "empty parts",
			artifact: &Artifact{
				ArtifactID: "artifact-123",
				Name:       "result.txt",
				Parts:      []Part{},
			},
			wantErr: true,
			errMsg:  "at least one part is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.artifact.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestTask_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		task    *Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid task",
			task: &Task{
				ID:        "task-123",
				ContextID: "ctx-456",
				Kind:      "task",
				Status: TaskStatus{
					State:     TaskStateSubmitted,
					Timestamp: now,
				},
			},
			wantErr: false,
		},
		{
			name: "empty task ID",
			task: &Task{
				ID:        "",
				ContextID: "ctx-456",
				Status: TaskStatus{
					State:     TaskStateSubmitted,
					Timestamp: now,
				},
			},
			wantErr: true,
			errMsg:  "task ID is required",
		},
		{
			name: "empty context ID",
			task: &Task{
				ID:        "task-123",
				ContextID: "",
				Status: TaskStatus{
					State:     TaskStateSubmitted,
					Timestamp: now,
				},
			},
			wantErr: true,
			errMsg:  "ContextID is required",
		},
		{
			name: "invalid state",
			task: &Task{
				ID:        "task-123",
				ContextID: "ctx-456",
				Status: TaskStatus{
					State:     "invalid-state",
					Timestamp: now,
				},
			},
			wantErr: true,
			errMsg:  "invalid task state",
		},
		{
			name: "missing timestamp",
			task: &Task{
				ID:        "task-123",
				ContextID: "ctx-456",
				Status: TaskStatus{
					State: TaskStateSubmitted,
				},
			},
			wantErr: true,
			errMsg:  "status timestamp is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestTask_UpdateState(t *testing.T) {
	task := &Task{
		ID:        "task-123",
		ContextID: "ctx-456",
		Kind:      "task",
		Status: TaskStatus{
			State:     TaskStateSubmitted,
			Timestamp: time.Now(),
		},
	}

	// Update state
	newState := TaskStateWorking
	task.UpdateState(newState, nil)

	if task.Status.State != newState {
		t.Errorf("State = %v, want %v", task.Status.State, newState)
	}

	if task.Status.Timestamp.IsZero() {
		t.Error("Timestamp should be updated")
	}

	if task.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be updated")
	}
}

func TestNewTask(t *testing.T) {
	id := "task-123"
	contextID := "ctx-456"

	task := NewTask(id, contextID)

	if task.ID != id {
		t.Errorf("ID = %v, want %v", task.ID, id)
	}

	if task.ContextID != contextID {
		t.Errorf("ContextID = %v, want %v", task.ContextID, contextID)
	}

	if task.Status.State != TaskStateSubmitted {
		t.Errorf("State = %v, want %v", task.Status.State, TaskStateSubmitted)
	}

	if task.Kind != "task" {
		t.Errorf("Kind = %v, want 'task'", task.Kind)
	}

	if task.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
}
