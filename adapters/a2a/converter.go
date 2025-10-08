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
	"encoding/base64"
	"fmt"

	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
	a2a "trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// toA2AMessage converts a sage-adk Message to an A2A Message.
func toA2AMessage(msg *types.Message) (a2a.Message, error) {
	if msg == nil {
		return a2a.Message{}, errors.ErrInvalidInput.WithMessage("message is nil")
	}

	parts, err := toA2AParts(msg.Parts)
	if err != nil {
		return a2a.Message{}, fmt.Errorf("failed to convert parts: %w", err)
	}

	a2aMsg := a2a.Message{
		MessageID: msg.MessageID,
		Role:      a2a.MessageRole(msg.Role),
		Parts:     parts,
		Kind:      a2a.KindMessage,
		ContextID: msg.ContextID,
	}

	return a2aMsg, nil
}

// fromA2AMessage converts an A2A Message to a sage-adk Message.
func fromA2AMessage(msg *a2a.Message) (*types.Message, error) {
	if msg == nil {
		return nil, errors.ErrInvalidInput.WithMessage("message is nil")
	}

	parts, err := fromA2AParts(msg.Parts)
	if err != nil {
		return nil, fmt.Errorf("failed to convert parts: %w", err)
	}

	result := &types.Message{
		MessageID: msg.MessageID,
		Role:      types.MessageRole(msg.Role),
		Parts:     parts,
		Kind:      msg.Kind,
		ContextID: msg.ContextID,
	}

	return result, nil
}

// toA2AParts converts sage-adk Parts to A2A Parts.
func toA2AParts(parts []types.Part) ([]a2a.Part, error) {
	if parts == nil {
		return nil, nil
	}

	result := make([]a2a.Part, 0, len(parts))

	for i, part := range parts {
		a2aPart, err := toA2APart(part)
		if err != nil {
			return nil, fmt.Errorf("failed to convert part %d: %w", i, err)
		}
		result = append(result, a2aPart)
	}

	return result, nil
}

// fromA2AParts converts A2A Parts to sage-adk Parts.
func fromA2AParts(parts []a2a.Part) ([]types.Part, error) {
	if parts == nil {
		return nil, nil
	}

	result := make([]types.Part, 0, len(parts))

	for i, part := range parts {
		sdkPart, err := fromA2APart(part)
		if err != nil {
			return nil, fmt.Errorf("failed to convert part %d: %w", i, err)
		}
		result = append(result, sdkPart)
	}

	return result, nil
}

// toA2APart converts a single sage-adk Part to an A2A Part.
func toA2APart(part types.Part) (a2a.Part, error) {
	switch p := part.(type) {
	case *types.TextPart:
		return a2a.TextPart{
			Kind: a2a.KindText,
			Text: p.Text,
		}, nil

	case *types.FilePart:
		return toA2AFilePart(p)

	case *types.DataPart:
		return a2a.DataPart{
			Kind: a2a.KindData,
			Data: p.Data,
		}, nil

	default:
		return nil, errors.ErrInvalidInput.WithDetail("type", fmt.Sprintf("%T", part))
	}
}

// fromA2APart converts a single A2A Part to a sage-adk Part.
func fromA2APart(part a2a.Part) (types.Part, error) {
	switch p := part.(type) {
	case a2a.TextPart:
		return &types.TextPart{
			Kind: string(types.PartKindText),
			Text: p.Text,
		}, nil

	case *a2a.TextPart:
		return &types.TextPart{
			Kind: string(types.PartKindText),
			Text: p.Text,
		}, nil

	case a2a.FilePart:
		return fromA2AFilePart(&p)

	case *a2a.FilePart:
		return fromA2AFilePart(p)

	case a2a.DataPart:
		return &types.DataPart{
			Kind: string(types.PartKindData),
			Data: p.Data,
		}, nil

	case *a2a.DataPart:
		return &types.DataPart{
			Kind: string(types.PartKindData),
			Data: p.Data,
		}, nil

	default:
		return nil, errors.ErrInvalidInput.WithDetail("type", fmt.Sprintf("%T", part))
	}
}

// toA2AFilePart converts a sage-adk FilePart to an A2A FilePart.
func toA2AFilePart(p *types.FilePart) (a2a.FilePart, error) {
	if p.File == nil {
		return a2a.FilePart{}, errors.ErrInvalidInput.WithMessage("file content is nil")
	}

	var fileUnion a2a.FileUnion

	switch f := p.File.(type) {
	case *types.FileWithBytes:
		// Encode bytes to base64 for A2A
		encoded := base64.StdEncoding.EncodeToString(f.Bytes)
		fileUnion = &a2a.FileWithBytes{
			Name:     &f.Name,
			MimeType: &f.MimeType,
			Bytes:    encoded,
		}

	case *types.FileWithURI:
		fileUnion = &a2a.FileWithURI{
			Name:     &f.Name,
			MimeType: &f.MimeType,
			URI:      f.URI,
		}

	default:
		return a2a.FilePart{}, errors.ErrInvalidInput.WithDetail("file_type", fmt.Sprintf("%T", f))
	}

	return a2a.FilePart{
		Kind: a2a.KindFile,
		File: fileUnion,
	}, nil
}

// fromA2AFilePart converts an A2A FilePart to a sage-adk FilePart.
func fromA2AFilePart(p *a2a.FilePart) (*types.FilePart, error) {
	if p.File == nil {
		return nil, errors.ErrInvalidInput.WithMessage("file content is nil")
	}

	var file types.FileContent

	switch f := p.File.(type) {
	case *a2a.FileWithBytes:
		// Decode base64 from A2A
		decoded, err := base64.StdEncoding.DecodeString(f.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to decode file bytes: %w", err)
		}

		name := ""
		if f.Name != nil {
			name = *f.Name
		}

		mimeType := ""
		if f.MimeType != nil {
			mimeType = *f.MimeType
		}

		file = &types.FileWithBytes{
			Name:     name,
			MimeType: mimeType,
			Bytes:    decoded,
		}

	case *a2a.FileWithURI:
		name := ""
		if f.Name != nil {
			name = *f.Name
		}

		mimeType := ""
		if f.MimeType != nil {
			mimeType = *f.MimeType
		}

		file = &types.FileWithURI{
			Name:     name,
			MimeType: mimeType,
			URI:      f.URI,
		}

	default:
		return nil, errors.ErrInvalidInput.WithDetail("file_type", fmt.Sprintf("%T", f))
	}

	return &types.FilePart{
		Kind: string(types.PartKindFile),
		File: file,
	}, nil
}
