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

/*
Package grpc provides gRPC server implementation for SAGE ADK agents.

This package implements the AgentService gRPC service defined in proto/agent.proto,
enabling high-performance bidirectional streaming communication between agents and clients.

Features:
  - Unary RPC for single message exchange
  - Bidirectional streaming for real-time communication
  - Agent metadata and health check endpoints
  - Automatic protocol conversion (gRPC ↔ internal types)
  - Connection management and graceful shutdown

Example:

	import (
	    "github.com/sage-x-project/sage-adk/server/grpc"
	    "github.com/sage-x-project/sage-adk/core/agent"
	)

	// Create agent
	agent, _ := builder.NewAgent("my-agent").Build()

	// Create gRPC server
	server := grpc.NewServer(agent, grpc.ServerConfig{
	    Port: 9090,
	    MaxConcurrentStreams: 100,
	})

	// Start server
	if err := server.Start(); err != nil {
	    log.Fatal(err)
	}

	// Graceful shutdown
	defer server.Stop(context.Background())
*/
package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/sage-x-project/sage-adk/core/agent"
	pb "github.com/sage-x-project/sage-adk/proto/pb"
)

// Server represents a gRPC server for SAGE ADK agents
type Server struct {
	pb.UnimplementedAgentServiceServer

	agent      agent.Agent
	grpcServer *grpc.Server
	config     ServerConfig
	listener   net.Listener

	// Health checker
	healthServer *health.Server

	// Active streams tracking
	mu      sync.RWMutex
	streams map[string]context.CancelFunc
}

// ServerConfig holds configuration for the gRPC server
type ServerConfig struct {
	// Port to listen on
	Port int

	// Maximum concurrent streams
	MaxConcurrentStreams uint32

	// Connection keep-alive settings
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration

	// Maximum message size
	MaxMessageSize int
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:                 9090,
		MaxConcurrentStreams: 100,
		KeepAliveTime:        2 * time.Hour,
		KeepAliveTimeout:     20 * time.Second,
		MaxMessageSize:       4 * 1024 * 1024, // 4MB
	}
}

// NewServer creates a new gRPC server
func NewServer(ag agent.Agent, config ServerConfig) *Server {
	if config.Port == 0 {
		config = DefaultServerConfig()
	}

	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(config.MaxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    config.KeepAliveTime,
			Timeout: config.KeepAliveTimeout,
		}),
		grpc.MaxRecvMsgSize(config.MaxMessageSize),
		grpc.MaxSendMsgSize(config.MaxMessageSize),
		grpc.Creds(insecure.NewCredentials()),
	}

	grpcServer := grpc.NewServer(opts...)

	// Create health server
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("sage.adk.v1.AgentService", healthpb.HealthCheckResponse_SERVING)

	s := &Server{
		agent:        ag,
		grpcServer:   grpcServer,
		config:       config,
		healthServer: healthServer,
		streams:      make(map[string]context.CancelFunc),
	}

	// Register services
	pb.RegisterAgentServiceServer(grpcServer, s)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	return s
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener

	log.Printf("🚀 gRPC server listening on %s", addr)

	// Start serving
	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("🛑 Stopping gRPC server...")

	// Cancel all active streams
	s.mu.Lock()
	for streamID, cancel := range s.streams {
		log.Printf("Cancelling stream: %s", streamID)
		cancel()
	}
	s.streams = make(map[string]context.CancelFunc)
	s.mu.Unlock()

	// Update health status
	s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	// Graceful stop with timeout
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("✅ gRPC server stopped gracefully")
	case <-ctx.Done():
		log.Println("⏱️ Timeout waiting for graceful stop, forcing shutdown")
		s.grpcServer.Stop()
	}

	return nil
}

// SendMessage implements unary message sending
func (s *Server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	if req.Message == nil {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}

	// Convert protobuf message to internal type
	msg, err := ProtoToMessage(req.Message)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid message: %v", err)
	}

	// Process message through agent
	response, err := s.agent.Process(ctx, msg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "agent processing failed: %v", err)
	}

	// Convert response back to protobuf
	pbResponse, err := MessageToProto(response)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "response conversion failed: %v", err)
	}

	return &pb.SendMessageResponse{
		Message:     pbResponse,
		StatusCode:  200,
		ProcessedAt: timestamppb.Now(),
	}, nil
}

// StreamMessages implements bidirectional streaming
func (s *Server) StreamMessages(stream pb.AgentService_StreamMessagesServer) error {
	streamID := fmt.Sprintf("stream-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// Register stream
	s.mu.Lock()
	s.streams[streamID] = cancel
	s.mu.Unlock()

	// Deregister on exit
	defer func() {
		s.mu.Lock()
		delete(s.streams, streamID)
		s.mu.Unlock()
		log.Printf("Stream closed: %s", streamID)
	}()

	log.Printf("New stream: %s", streamID)

	// Handle incoming messages
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "stream receive error: %v", err)
		}

		// Handle different request types
		switch r := req.Request.(type) {
		case *pb.StreamMessageRequest_Message:
			// Process message
			msg, err := ProtoToMessage(r.Message)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid message: %v", err)
			}

			response, err := s.agent.Process(ctx, msg)
			if err != nil {
				// Send error status
				stream.Send(&pb.StreamMessageResponse{
					Response: &pb.StreamMessageResponse_Status{
						Status: &pb.StreamStatus{
							Type:      pb.StreamStatus_STATUS_TYPE_ERROR,
							Message:   err.Error(),
							Timestamp: timestamppb.Now(),
						},
					},
				})
				continue
			}

			// Send response
			pbResponse, _ := MessageToProto(response)
			stream.Send(&pb.StreamMessageResponse{
				Response: &pb.StreamMessageResponse_Message{
					Message: pbResponse,
				},
			})

		case *pb.StreamMessageRequest_Control:
			// Handle control messages
			switch r.Control.Type {
			case pb.StreamControl_CONTROL_TYPE_PING:
				stream.Send(&pb.StreamMessageResponse{
					Response: &pb.StreamMessageResponse_Status{
						Status: &pb.StreamStatus{
							Type:      pb.StreamStatus_STATUS_TYPE_PONG,
							Timestamp: timestamppb.Now(),
						},
					},
				})

			case pb.StreamControl_CONTROL_TYPE_CLOSE:
				stream.Send(&pb.StreamMessageResponse{
					Response: &pb.StreamMessageResponse_Status{
						Status: &pb.StreamStatus{
							Type:      pb.StreamStatus_STATUS_TYPE_CLOSED,
							Message:   "Stream closed by client",
							Timestamp: timestamppb.Now(),
						},
					},
				})
				return nil
			}
		}
	}
}

// GetAgentInfo returns agent metadata
func (s *Server) GetAgentInfo(ctx context.Context, req *pb.GetAgentInfoRequest) (*pb.GetAgentInfoResponse, error) {
	card := s.agent.Card()

	return &pb.GetAgentInfoResponse{
		AgentCard: &pb.AgentCard{
			Name:        card.Name,
			Description: card.Description,
			Version:     card.Version,
			// Capabilities and protocols would come from card
		},
	}, nil
}

// HealthCheck implements health checking
func (s *Server) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	// Check agent health (this is simplified)
	return &pb.HealthCheckResponse{
		Status:    pb.HealthCheckResponse_SERVING_STATUS_SERVING,
		Message:   "Agent is healthy",
		CheckedAt: timestamppb.Now(),
	}, nil
}
