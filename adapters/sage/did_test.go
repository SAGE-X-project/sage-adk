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

package sage

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	sageconfig "github.com/sage-x-project/sage/config"
	"github.com/sage-x-project/sage/did"
	adkconfig "github.com/sage-x-project/sage-adk/config"
)

// mockResolver implements did.Resolver interface for testing
type mockResolver struct {
	resolveFunc           func(ctx context.Context, did did.AgentDID) (*did.AgentMetadata, error)
	resolvePublicKeyFunc  func(ctx context.Context, did did.AgentDID) (interface{}, error)
	resolveKEMKeyFunc     func(ctx context.Context, did did.AgentDID) (interface{}, error)
	verifyMetadataFunc    func(ctx context.Context, did did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error)
	listAgentsByOwnerFunc func(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error)
	searchFunc            func(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error)
}

func (m *mockResolver) Resolve(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(ctx, agentDID)
	}
	return nil, did.ErrDIDNotFound
}

func (m *mockResolver) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	if m.resolvePublicKeyFunc != nil {
		return m.resolvePublicKeyFunc(ctx, agentDID)
	}
	return nil, did.ErrDIDNotFound
}

func (m *mockResolver) ResolveKEMKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	if m.resolveKEMKeyFunc != nil {
		return m.resolveKEMKeyFunc(ctx, agentDID)
	}
	return nil, did.ErrDIDNotFound
}

func (m *mockResolver) VerifyMetadata(ctx context.Context, agentDID did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
	if m.verifyMetadataFunc != nil {
		return m.verifyMetadataFunc(ctx, agentDID, metadata)
	}
	return &did.VerificationResult{Valid: false}, nil
}

func (m *mockResolver) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
	if m.listAgentsByOwnerFunc != nil {
		return m.listAgentsByOwnerFunc(ctx, ownerAddress)
	}
	return nil, nil
}

func (m *mockResolver) Search(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, criteria)
	}
	return nil, nil
}

func createTestConfig(network string) *Config {
	adkCfg := &adkconfig.SAGEConfig{
		Enabled:         true,
		Network:         network,
		DID:             "did:sage:" + network + ":0x123",
		RPCEndpoint:     "https://test.example.com",
		ContractAddress: "0xABC",
		PrivateKeyPath:  "/test/key.pem",
		CacheTTL:        1 * time.Hour,
	}

	cfg, _ := FromADKConfig(adkCfg)
	return cfg
}

func TestNewDIDResolver(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "invalid config - empty DID",
			config: &Config{
				Config: &sageconfig.Config{
					DID: &sageconfig.DIDConfig{
						Network: "ethereum",
					},
					Blockchain: &sageconfig.BlockchainConfig{
						NetworkRPC: "https://test.example.com",
					},
					KeyStore: &sageconfig.KeyStoreConfig{
						Directory: "/test",
					},
				},
				LocalDID:       "",
				PrivateKeyPath: "/test/key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewDIDResolver(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewDIDResolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resolver == nil {
				t.Error("NewDIDResolver() returned nil resolver")
			}
		})
	}
}

// Note: Testing actual NewDIDResolver with real networks requires live RPC endpoints.
// Such tests should be integration tests and run separately with the integration build tag.

func TestDIDResolver_Resolve(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)

	tests := []struct {
		name       string
		did        string
		mockResult *did.AgentMetadata
		mockError  error
		wantErr    bool
	}{
		{
			name: "successful resolution",
			did:  "did:sage:ethereum:0x123",
			mockResult: &did.AgentMetadata{
				DID:       "did:sage:ethereum:0x123",
				Name:      "Test Agent",
				PublicKey: pub,
				IsActive:  true,
			},
			mockError: nil,
			wantErr:   false,
		},
		{
			name:       "DID not found",
			did:        "did:sage:ethereum:0x999",
			mockResult: nil,
			mockError:  did.ErrDIDNotFound,
			wantErr:    true,
		},
		{
			name:       "empty DID",
			did:        "",
			mockResult: nil,
			mockError:  nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockResolver{
				resolveFunc: func(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResult, nil
				},
			}

			resolver := &DIDResolver{
				resolver: mock,
				config:   createTestConfig("ethereum"),
			}

			ctx := context.Background()
			metadata, err := resolver.Resolve(ctx, tt.did)

			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && metadata == nil {
				t.Error("Resolve() returned nil metadata")
			}

			if !tt.wantErr && metadata != nil {
				if metadata.DID != tt.mockResult.DID {
					t.Errorf("Resolve() DID = %s, want %s", metadata.DID, tt.mockResult.DID)
				}
			}
		})
	}
}

func TestDIDResolver_ResolvePublicKey(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)

	tests := []struct {
		name      string
		did       string
		mockKey   interface{}
		mockError error
		wantErr   bool
	}{
		{
			name:      "successful resolution",
			did:       "did:sage:ethereum:0x123",
			mockKey:   pub,
			mockError: nil,
			wantErr:   false,
		},
		{
			name:      "DID not found",
			did:       "did:sage:ethereum:0x999",
			mockKey:   nil,
			mockError: did.ErrDIDNotFound,
			wantErr:   true,
		},
		{
			name:      "empty DID",
			did:       "",
			mockKey:   nil,
			mockError: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockResolver{
				resolvePublicKeyFunc: func(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockKey, nil
				},
			}

			resolver := &DIDResolver{
				resolver: mock,
				config:   createTestConfig("ethereum"),
			}

			ctx := context.Background()
			key, err := resolver.ResolvePublicKey(ctx, tt.did)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePublicKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && key == nil {
				t.Error("ResolvePublicKey() returned nil key")
			}
		})
	}
}

func TestDIDResolver_VerifyMetadata(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)

	validMetadata := &did.AgentMetadata{
		DID:       "did:sage:ethereum:0x123",
		Name:      "Test Agent",
		PublicKey: pub,
		IsActive:  true,
	}

	tests := []struct {
		name       string
		did        string
		metadata   *did.AgentMetadata
		mockResult *did.VerificationResult
		mockError  error
		wantErr    bool
		wantValid  bool
	}{
		{
			name:     "valid metadata",
			did:      "did:sage:ethereum:0x123",
			metadata: validMetadata,
			mockResult: &did.VerificationResult{
				Valid:      true,
				Agent:      validMetadata,
				VerifiedAt: time.Now(),
			},
			mockError: nil,
			wantErr:   false,
			wantValid: true,
		},
		{
			name:     "invalid metadata",
			did:      "did:sage:ethereum:0x123",
			metadata: validMetadata,
			mockResult: &did.VerificationResult{
				Valid: false,
				Error: "metadata mismatch",
			},
			mockError: nil,
			wantErr:   false,
			wantValid: false,
		},
		{
			name:       "empty DID",
			did:        "",
			metadata:   validMetadata,
			mockResult: nil,
			mockError:  nil,
			wantErr:    true,
		},
		{
			name:       "nil metadata",
			did:        "did:sage:ethereum:0x123",
			metadata:   nil,
			mockResult: nil,
			mockError:  nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockResolver{
				verifyMetadataFunc: func(ctx context.Context, agentDID did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResult, nil
				},
			}

			resolver := &DIDResolver{
				resolver: mock,
				config:   createTestConfig("ethereum"),
			}

			ctx := context.Background()
			result, err := resolver.VerifyMetadata(ctx, tt.did, tt.metadata)

			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result.Valid != tt.wantValid {
				t.Errorf("VerifyMetadata() valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

func TestDIDResolver_ListAgentsByOwner(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)

	agents := []*did.AgentMetadata{
		{
			DID:       "did:sage:ethereum:0x123",
			Name:      "Agent 1",
			PublicKey: pub,
			IsActive:  true,
		},
		{
			DID:       "did:sage:ethereum:0x456",
			Name:      "Agent 2",
			PublicKey: pub,
			IsActive:  true,
		},
	}

	tests := []struct {
		name         string
		ownerAddress string
		mockAgents   []*did.AgentMetadata
		mockError    error
		wantErr      bool
		wantCount    int
	}{
		{
			name:         "successful list",
			ownerAddress: "0xowner123",
			mockAgents:   agents,
			mockError:    nil,
			wantErr:      false,
			wantCount:    2,
		},
		{
			name:         "no agents found",
			ownerAddress: "0xowner999",
			mockAgents:   []*did.AgentMetadata{},
			mockError:    nil,
			wantErr:      false,
			wantCount:    0,
		},
		{
			name:         "empty owner address",
			ownerAddress: "",
			mockAgents:   nil,
			mockError:    nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockResolver{
				listAgentsByOwnerFunc: func(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockAgents, nil
				},
			}

			resolver := &DIDResolver{
				resolver: mock,
				config:   createTestConfig("ethereum"),
			}

			ctx := context.Background()
			result, err := resolver.ListAgentsByOwner(ctx, tt.ownerAddress)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListAgentsByOwner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(result) != tt.wantCount {
				t.Errorf("ListAgentsByOwner() count = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestDIDResolver_Search(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)

	agents := []*did.AgentMetadata{
		{
			DID:       "did:sage:ethereum:0x123",
			Name:      "Test Agent",
			PublicKey: pub,
			IsActive:  true,
		},
	}

	tests := []struct {
		name       string
		criteria   did.SearchCriteria
		mockAgents []*did.AgentMetadata
		mockError  error
		wantErr    bool
		wantCount  int
	}{
		{
			name: "successful search",
			criteria: did.SearchCriteria{
				Name:       "Test",
				ActiveOnly: true,
				Limit:      10,
			},
			mockAgents: agents,
			mockError:  nil,
			wantErr:    false,
			wantCount:  1,
		},
		{
			name: "no results",
			criteria: did.SearchCriteria{
				Name:  "Nonexistent",
				Limit: 10,
			},
			mockAgents: []*did.AgentMetadata{},
			mockError:  nil,
			wantErr:    false,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockResolver{
				searchFunc: func(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockAgents, nil
				},
			}

			resolver := &DIDResolver{
				resolver: mock,
				config:   createTestConfig("ethereum"),
			}

			ctx := context.Background()
			result, err := resolver.Search(ctx, tt.criteria)

			if (err != nil) != tt.wantErr {
				t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(result) != tt.wantCount {
				t.Errorf("Search() count = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestDIDResolver_IsActive(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)

	tests := []struct {
		name       string
		did        string
		mockActive bool
		mockError  error
		wantErr    bool
		wantActive bool
	}{
		{
			name: "active agent",
			did:  "did:sage:ethereum:0x123",
			mockActive: true,
			mockError:  nil,
			wantErr:    false,
			wantActive: true,
		},
		{
			name:       "inactive agent",
			did:        "did:sage:ethereum:0x456",
			mockActive: false,
			mockError:  nil,
			wantErr:    false,
			wantActive: false,
		},
		{
			name:       "DID not found",
			did:        "did:sage:ethereum:0x999",
			mockActive: false,
			mockError:  did.ErrDIDNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockResolver{
				resolveFunc: func(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return &did.AgentMetadata{
						DID:       agentDID,
						PublicKey: pub,
						IsActive:  tt.mockActive,
					}, nil
				},
			}

			resolver := &DIDResolver{
				resolver: mock,
				config:   createTestConfig("ethereum"),
			}

			ctx := context.Background()
			isActive, err := resolver.IsActive(ctx, tt.did)

			if (err != nil) != tt.wantErr {
				t.Errorf("IsActive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && isActive != tt.wantActive {
				t.Errorf("IsActive() = %v, want %v", isActive, tt.wantActive)
			}
		})
	}
}

func TestDIDResolver_GetAgentCapabilities(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)

	capabilities := map[string]interface{}{
		"messaging": true,
		"payments":  false,
	}

	tests := []struct {
		name             string
		did              string
		mockCapabilities map[string]interface{}
		mockError        error
		wantErr          bool
	}{
		{
			name:             "successful retrieval",
			did:              "did:sage:ethereum:0x123",
			mockCapabilities: capabilities,
			mockError:        nil,
			wantErr:          false,
		},
		{
			name:             "DID not found",
			did:              "did:sage:ethereum:0x999",
			mockCapabilities: nil,
			mockError:        did.ErrDIDNotFound,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockResolver{
				resolveFunc: func(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return &did.AgentMetadata{
						DID:          agentDID,
						PublicKey:    pub,
						IsActive:     true,
						Capabilities: tt.mockCapabilities,
					}, nil
				},
			}

			resolver := &DIDResolver{
				resolver: mock,
				config:   createTestConfig("ethereum"),
			}

			ctx := context.Background()
			caps, err := resolver.GetAgentCapabilities(ctx, tt.did)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAgentCapabilities() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(caps) != len(tt.mockCapabilities) {
				t.Errorf("GetAgentCapabilities() count = %d, want %d", len(caps), len(tt.mockCapabilities))
			}
		})
	}
}

func TestNewMultiChainDIDResolver(t *testing.T) {
	tests := []struct {
		name    string
		configs map[string]*Config
		wantErr bool
	}{
		{
			name:    "empty configs",
			configs: map[string]*Config{},
			wantErr: true,
		},
		{
			name:    "nil configs",
			configs: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewMultiChainDIDResolver(tt.configs)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewMultiChainDIDResolver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resolver == nil {
				t.Error("NewMultiChainDIDResolver() returned nil resolver")
			}
		})
	}
}

// Note: Testing actual NewMultiChainDIDResolver with real networks requires live RPC endpoints.
// Such tests should be integration tests and run separately with the integration build tag.
