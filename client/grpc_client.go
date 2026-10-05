// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

package client

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/sage-x-project/sage-adk/pkg/types"
	pb "github.com/sage-x-project/sage-adk/proto/pb"
	grpcserver "github.com/sage-x-project/sage-adk/server/grpc"
)

// GRPCClient implements Client interface using gRPC protocol
type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.AgentServiceClient
	config *Client
}

// NewGRPCClient creates a legacy plaintext gRPC client.
// WithTimeout and WithRetry apply; HTTP-specific options have no gRPC effect.
// This transport is not the protected SAGE 0.10.0 binding.
func NewGRPCClient(target string, opts ...Option) (*GRPCClient, error) {
	config := &Client{timeout: 30 * time.Second, maxRetries: 3, initialDelay: 100 * time.Millisecond, maxDelay: 5 * time.Second}
	for _, opt := range opts {
		opt(config)
	}

	// Set up gRPC connection options
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(4*1024*1024),
			grpc.MaxCallSendMsgSize(4*1024*1024),
		),
	}

	// Connect to gRPC server
	conn, err := grpc.Dial(target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &GRPCClient{
		conn:   conn,
		client: pb.NewAgentServiceClient(conn),
		config: config,
	}, nil
}

// SendMessage sends a message via gRPC
func (c *GRPCClient) SendMessage(ctx context.Context, message *types.Message) (*types.Message, error) {
	// Apply timeout
	if c.config.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.timeout)
		defer cancel()
	}

	// Convert to protobuf
	pbMsg, err := grpcserver.MessageToProto(message)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message: %w", err)
	}

	// Send request
	req := &pb.SendMessageRequest{
		Message:        pbMsg,
		TimeoutSeconds: int32(c.config.timeout.Seconds()),
	}

	// Execute with retry
	var response *pb.SendMessageResponse
	err = c.executeWithRetry(ctx, func() error {
		var err error
		response, err = c.client.SendMessage(ctx, req)
		return err
	})

	if err != nil {
		return nil, err
	}

	// Convert response
	return grpcserver.ProtoToMessage(response.Message)
}

// SendMessageStream creates a bidirectional stream
func (c *GRPCClient) SendMessageStream(ctx context.Context) (MessageStream, error) {
	stream, err := c.client.StreamMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	return &grpcMessageStream{
		stream: stream,
	}, nil
}

// Close closes the gRPC connection
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// executeWithRetry executes a function with retry logic
func (c *GRPCClient) executeWithRetry(ctx context.Context, fn func() error) error {
	if c.config.maxRetries == 0 {
		return fn()
	}

	var lastErr error
	backoff := c.config.initialDelay

	for attempt := 0; attempt <= c.config.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}

			// Exponential backoff with jitter
			backoff = time.Duration(float64(backoff) * 2.0)
			if backoff > c.config.maxDelay {
				backoff = c.config.maxDelay
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// grpcMessageStream implements MessageStream for gRPC
type grpcMessageStream struct {
	stream pb.AgentService_StreamMessagesClient
}

func (s *grpcMessageStream) Send(msg *types.Message) error {
	pbMsg, err := grpcserver.MessageToProto(msg)
	if err != nil {
		return fmt.Errorf("failed to convert message: %w", err)
	}

	return s.stream.Send(&pb.StreamMessageRequest{
		Request: &pb.StreamMessageRequest_Message{
			Message: pbMsg,
		},
	})
}

func (s *grpcMessageStream) Recv() (*types.Message, error) {
	resp, err := s.stream.Recv()
	if err == io.EOF {
		return nil, io.EOF
	}
	if err != nil {
		return nil, err
	}

	switch r := resp.Response.(type) {
	case *pb.StreamMessageResponse_Message:
		return grpcserver.ProtoToMessage(r.Message)
	case *pb.StreamMessageResponse_Status:
		if r.Status.Type == pb.StreamStatus_STATUS_TYPE_ERROR {
			return nil, fmt.Errorf("stream error: %s", r.Status.Message)
		}
		// For other status types, continue receiving
		return s.Recv()
	default:
		return nil, fmt.Errorf("unknown response type")
	}
}

func (s *grpcMessageStream) Close() error {
	// Send close control message
	err := s.stream.Send(&pb.StreamMessageRequest{
		Request: &pb.StreamMessageRequest_Control{
			Control: &pb.StreamControl{
				Type: pb.StreamControl_CONTROL_TYPE_CLOSE,
			},
		},
	})
	if err != nil {
		return err
	}

	return s.stream.CloseSend()
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	// gRPC-specific error checking would go here
	// For now, retry on any error
	return true
}
