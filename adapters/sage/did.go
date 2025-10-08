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
	"fmt"

	"github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/did/ethereum"
)

// ethereumResolverAdapter adapts EthereumClient to the did.Resolver interface.
// This is needed because EthereumClient returns crypto.PublicKey while
// did.Resolver expects interface{} for ResolvePublicKey.
type ethereumResolverAdapter struct {
	client *ethereum.EthereumClient
}

func (a *ethereumResolverAdapter) Resolve(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	return a.client.Resolve(ctx, agentDID)
}

func (a *ethereumResolverAdapter) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	key, err := a.client.ResolvePublicKey(ctx, agentDID)
	if err != nil {
		return nil, err
	}
	// Convert crypto.PublicKey to interface{}
	return interface{}(key), nil
}

func (a *ethereumResolverAdapter) VerifyMetadata(ctx context.Context, agentDID did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
	return a.client.VerifyMetadata(ctx, agentDID, metadata)
}

func (a *ethereumResolverAdapter) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
	return a.client.ListAgentsByOwner(ctx, ownerAddress)
}

func (a *ethereumResolverAdapter) Search(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	return a.client.Search(ctx, criteria)
}

// DIDResolver provides a wrapper around sage DID resolution for sage-adk.
// It simplifies DID operations while using the full functionality of the sage library.
type DIDResolver struct {
	resolver did.Resolver
	config   *Config
}

// NewDIDResolver creates a new DID resolver using the provided configuration.
// It initializes the appropriate chain-specific resolver based on the network.
func NewDIDResolver(cfg *Config) (*DIDResolver, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create registry config for Ethereum client
	registryConfig := &did.RegistryConfig{
		Chain:              did.ChainEthereum,
		Network:            did.Network(cfg.DID.Network),
		ContractAddress:    cfg.Blockchain.ContractAddr,
		RPCEndpoint:        cfg.Blockchain.NetworkRPC,
		PrivateKey:         "", // Optional: only needed for write operations
		GasPrice:           0,  // Use network's suggested gas price
		MaxRetries:         3,
		ConfirmationBlocks: 1,
	}

	// Create chain-specific resolver based on network
	var resolver did.Resolver

	switch cfg.DID.Network {
	case "ethereum", "mainnet", "sepolia", "goerli", "local", "localhost":
		// Create Ethereum resolver
		ethClient, err := ethereum.NewEthereumClient(registryConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Ethereum resolver: %w", err)
		}
		resolver = &ethereumResolverAdapter{client: ethClient}

	case "kaia", "cypress", "kairos", "kaia-testnet":
		// Kaia uses Ethereum-compatible resolver
		registryConfig.Chain = did.ChainEthereum // Kaia is Ethereum-compatible
		ethClient, err := ethereum.NewEthereumClient(registryConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kaia resolver: %w", err)
		}
		resolver = &ethereumResolverAdapter{client: ethClient}

	default:
		return nil, fmt.Errorf("unsupported network: %s", cfg.DID.Network)
	}

	return &DIDResolver{
		resolver: resolver,
		config:   cfg,
	}, nil
}

// NewMultiChainDIDResolver creates a multi-chain DID resolver.
// This is useful when you need to resolve DIDs across multiple blockchains.
func NewMultiChainDIDResolver(configs map[string]*Config) (*DIDResolver, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("at least one config is required")
	}

	multiResolver := did.NewMultiChainResolver()

	for network, cfg := range configs {
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid config for network %s: %w", network, err)
		}

		// Determine chain type
		var chain did.Chain
		switch cfg.DID.Network {
		case "ethereum", "mainnet", "sepolia", "goerli", "kaia", "cypress", "kairos", "kaia-testnet", "local", "localhost":
			chain = did.ChainEthereum
		default:
			return nil, fmt.Errorf("unsupported network: %s", cfg.DID.Network)
		}

		// Create registry config
		registryConfig := &did.RegistryConfig{
			Chain:              chain,
			Network:            did.Network(cfg.DID.Network),
			ContractAddress:    cfg.Blockchain.ContractAddr,
			RPCEndpoint:        cfg.Blockchain.NetworkRPC,
			PrivateKey:         "",
			GasPrice:           0,
			MaxRetries:         3,
			ConfirmationBlocks: 1,
		}

		// Create chain-specific resolver
		ethClient, err := ethereum.NewEthereumClient(registryConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create resolver for %s: %w", network, err)
		}

		multiResolver.AddResolver(chain, &ethereumResolverAdapter{client: ethClient})
	}

	// Use the first config as the default config
	var defaultConfig *Config
	for _, cfg := range configs {
		defaultConfig = cfg
		break
	}

	return &DIDResolver{
		resolver: multiResolver,
		config:   defaultConfig,
	}, nil
}

// Resolve resolves a DID to its metadata.
func (r *DIDResolver) Resolve(ctx context.Context, agentDID string) (*did.AgentMetadata, error) {
	if agentDID == "" {
		return nil, fmt.Errorf("DID cannot be empty")
	}

	metadata, err := r.resolver.Resolve(ctx, did.AgentDID(agentDID))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DID %s: %w", agentDID, err)
	}

	return metadata, nil
}

// ResolvePublicKey resolves a DID to its public key.
// This is a convenience method for getting just the public key.
func (r *DIDResolver) ResolvePublicKey(ctx context.Context, agentDID string) (interface{}, error) {
	if agentDID == "" {
		return nil, fmt.Errorf("DID cannot be empty")
	}

	publicKey, err := r.resolver.ResolvePublicKey(ctx, did.AgentDID(agentDID))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve public key for DID %s: %w", agentDID, err)
	}

	return publicKey, nil
}

// VerifyMetadata verifies that the provided metadata matches the on-chain data.
func (r *DIDResolver) VerifyMetadata(ctx context.Context, agentDID string, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
	if agentDID == "" {
		return nil, fmt.Errorf("DID cannot be empty")
	}

	if metadata == nil {
		return nil, fmt.Errorf("metadata cannot be nil")
	}

	result, err := r.resolver.VerifyMetadata(ctx, did.AgentDID(agentDID), metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to verify metadata for DID %s: %w", agentDID, err)
	}

	return result, nil
}

// ListAgentsByOwner lists all agents owned by a specific address.
func (r *DIDResolver) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
	if ownerAddress == "" {
		return nil, fmt.Errorf("owner address cannot be empty")
	}

	agents, err := r.resolver.ListAgentsByOwner(ctx, ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents for owner %s: %w", ownerAddress, err)
	}

	return agents, nil
}

// Search searches for agents matching the given criteria.
func (r *DIDResolver) Search(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	agents, err := r.resolver.Search(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	return agents, nil
}

// IsActive checks if an agent DID is active.
func (r *DIDResolver) IsActive(ctx context.Context, agentDID string) (bool, error) {
	metadata, err := r.Resolve(ctx, agentDID)
	if err != nil {
		return false, err
	}

	return metadata.IsActive, nil
}

// GetAgentCapabilities retrieves the capabilities of an agent.
func (r *DIDResolver) GetAgentCapabilities(ctx context.Context, agentDID string) (map[string]interface{}, error) {
	metadata, err := r.Resolve(ctx, agentDID)
	if err != nil {
		return nil, err
	}

	return metadata.Capabilities, nil
}

// Resolver returns the underlying sage did.Resolver for advanced operations.
func (r *DIDResolver) Resolver() did.Resolver {
	return r.resolver
}
