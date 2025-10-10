// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpc

import (
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sage-x-project/sage-adk/pkg/types"
	pb "github.com/sage-x-project/sage-adk/proto/pb"
)

// ProtoToMessage converts protobuf Message to internal Message type
func ProtoToMessage(pbMsg *pb.Message) (*types.Message, error) {
	if pbMsg == nil {
		return nil, fmt.Errorf("nil protobuf message")
	}

	// Convert role
	role := MessageRoleFromProto(pbMsg.Role)

	// Convert parts
	parts := make([]types.Part, 0, len(pbMsg.Parts))
	for _, pbPart := range pbMsg.Parts {
		part, err := PartFromProto(pbPart)
		if err != nil {
			return nil, fmt.Errorf("failed to convert part: %w", err)
		}
		parts = append(parts, part)
	}

	// Create message
	msg := &types.Message{
		MessageID: pbMsg.MessageId,
		Role:      role,
		Parts:     parts,
		Kind:      pbMsg.Kind,
	}

	// Set optional fields
	if pbMsg.ContextId != "" {
		msg.ContextID = &pbMsg.ContextId
	}

	if pbMsg.TaskId != "" {
		msg.TaskID = &pbMsg.TaskId
	}

	if len(pbMsg.ReferenceTaskIds) > 0 {
		msg.ReferenceTaskIDs = pbMsg.ReferenceTaskIds
	}

	if len(pbMsg.Extensions) > 0 {
		msg.Extensions = pbMsg.Extensions
	}

	// Convert metadata
	if pbMsg.Metadata != nil {
		msg.Metadata = pbMsg.Metadata.AsMap()
	}

	return msg, nil
}

// MessageToProto converts internal Message to protobuf Message
func MessageToProto(msg *types.Message) (*pb.Message, error) {
	if msg == nil {
		return nil, fmt.Errorf("nil message")
	}

	// Convert role
	role := MessageRoleToProto(msg.Role)

	// Convert parts
	pbParts := make([]*pb.Part, 0, len(msg.Parts))
	for _, part := range msg.Parts {
		pbPart, err := PartToProto(part)
		if err != nil {
			return nil, fmt.Errorf("failed to convert part: %w", err)
		}
		pbParts = append(pbParts, pbPart)
	}

	pbMsg := &pb.Message{
		MessageId: msg.MessageID,
		Role:      role,
		Parts:     pbParts,
		Kind:      msg.Kind,
		CreatedAt: timestamppb.Now(),
	}

	// Set optional fields
	if msg.ContextID != nil {
		pbMsg.ContextId = *msg.ContextID
	}

	if msg.TaskID != nil {
		pbMsg.TaskId = *msg.TaskID
	}

	if len(msg.ReferenceTaskIDs) > 0 {
		pbMsg.ReferenceTaskIds = msg.ReferenceTaskIDs
	}

	if len(msg.Extensions) > 0 {
		pbMsg.Extensions = msg.Extensions
	}

	return pbMsg, nil
}

// PartFromProto converts protobuf Part to internal Part
func PartFromProto(pbPart *pb.Part) (types.Part, error) {
	if pbPart == nil {
		return nil, fmt.Errorf("nil protobuf part")
	}

	switch p := pbPart.Part.(type) {
	case *pb.Part_Text:
		return &types.TextPart{
			Kind: p.Text.Kind,
			Text: p.Text.Text,
		}, nil

	case *pb.Part_File:
		// Simplified file conversion
		return &types.FilePart{
			Kind: p.File.Kind,
			// File content conversion would go here
		}, nil

	case *pb.Part_Data:
		return &types.DataPart{
			Kind: p.Data.Kind,
			Data: p.Data.Data.AsMap(),
		}, nil

	default:
		return nil, fmt.Errorf("unknown part type")
	}
}

// PartToProto converts internal Part to protobuf Part
func PartToProto(part types.Part) (*pb.Part, error) {
	if part == nil {
		return nil, fmt.Errorf("nil part")
	}

	switch p := part.(type) {
	case *types.TextPart:
		return &pb.Part{
			Part: &pb.Part_Text{
				Text: &pb.TextPart{
					Kind: p.Kind,
					Text: p.Text,
				},
			},
		}, nil

	case *types.FilePart:
		return &pb.Part{
			Part: &pb.Part_File{
				File: &pb.FilePart{
					Kind: p.Kind,
					// File content conversion would go here
				},
			},
		}, nil

	case *types.DataPart:
		return &pb.Part{
			Part: &pb.Part_Data{
				Data: &pb.DataPart{
					Kind: p.Kind,
					// Data conversion would go here
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown part type: %T", part)
	}
}

// MessageRoleFromProto converts protobuf MessageRole to internal MessageRole
func MessageRoleFromProto(role pb.MessageRole) types.MessageRole {
	switch role {
	case pb.MessageRole_MESSAGE_ROLE_USER:
		return types.MessageRoleUser
	case pb.MessageRole_MESSAGE_ROLE_AGENT:
		return types.MessageRoleAgent
	case pb.MessageRole_MESSAGE_ROLE_SYSTEM:
		return types.MessageRoleUser // Map system to user for now
	default:
		return types.MessageRoleUser
	}
}

// MessageRoleToProto converts internal MessageRole to protobuf MessageRole
func MessageRoleToProto(role types.MessageRole) pb.MessageRole {
	switch role {
	case types.MessageRoleUser:
		return pb.MessageRole_MESSAGE_ROLE_USER
	case types.MessageRoleAgent:
		return pb.MessageRole_MESSAGE_ROLE_AGENT
	default:
		return pb.MessageRole_MESSAGE_ROLE_UNSPECIFIED
	}
}
