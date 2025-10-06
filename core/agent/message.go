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

package agent

import (
	"context"

	"github.com/sage-x-project/sage-adk/pkg/errors"
	"github.com/sage-x-project/sage-adk/pkg/types"
)

// messageContext implements MessageContext.
type messageContext struct {
	agent    *agentImpl
	message  *types.Message
	ctx      context.Context
	response *types.Message
}

// Text returns the message text content.
func (m *messageContext) Text() string {
	for _, part := range m.message.Parts {
		if textPart, ok := part.(*types.TextPart); ok {
			return textPart.Text
		}
	}
	return ""
}

// Parts returns all message parts.
func (m *messageContext) Parts() []types.Part {
	return m.message.Parts
}

// ContextID returns the conversation context ID.
func (m *messageContext) ContextID() string {
	if m.message.ContextID == nil {
		return ""
	}
	return *m.message.ContextID
}

// MessageID returns the message ID.
func (m *messageContext) MessageID() string {
	return m.message.MessageID
}

// Reply sends a text response.
func (m *messageContext) Reply(text string) error {
	parts := []types.Part{types.NewTextPart(text)}
	return m.ReplyWithParts(parts)
}

// ReplyWithParts sends a response with multiple parts.
func (m *messageContext) ReplyWithParts(parts []types.Part) error {
	if m.response != nil {
		return errors.ErrInternal.WithMessage("response already sent")
	}

	// Create response message
	response := types.NewMessage(types.MessageRoleAgent, parts)
	response.ContextID = m.message.ContextID

	m.response = response
	return nil
}
