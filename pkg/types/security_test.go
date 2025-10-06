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

func TestProtocolMode_IsValid(t *testing.T) {
	tests := []struct {
		name string
		mode ProtocolMode
		want bool
	}{
		{"a2a is valid", ProtocolModeA2A, true},
		{"sage is valid", ProtocolModeSAGE, true},
		{"auto is valid", ProtocolModeAuto, true},
		{"invalid mode", ProtocolMode("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignatureAlgorithm_IsValid(t *testing.T) {
	tests := []struct {
		name string
		alg  SignatureAlgorithm
		want bool
	}{
		{"EdDSA is valid", AlgorithmEdDSA, true},
		{"ES256K is valid", AlgorithmES256K, true},
		{"ECDSA is valid", AlgorithmECDSA, true},
		{"ECDSA-secp256k1 is valid", AlgorithmECDSASecp256k1, true},
		{"invalid algorithm", SignatureAlgorithm("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.alg.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSignatureData_Validate(t *testing.T) {
	tests := []struct {
		name    string
		sig     *SignatureData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid signature",
			sig: &SignatureData{
				Algorithm:    AlgorithmEdDSA,
				KeyID:        "key-123",
				Signature:    []byte("signature"),
				SignedFields: []string{"messageId", "timestamp"},
			},
			wantErr: false,
		},
		{
			name: "invalid algorithm",
			sig: &SignatureData{
				Algorithm:    "invalid",
				KeyID:        "key-123",
				Signature:    []byte("signature"),
				SignedFields: []string{"messageId"},
			},
			wantErr: true,
			errMsg:  "invalid signature algorithm",
		},
		{
			name: "missing KeyID",
			sig: &SignatureData{
				Algorithm:    AlgorithmEdDSA,
				KeyID:        "",
				Signature:    []byte("signature"),
				SignedFields: []string{"messageId"},
			},
			wantErr: true,
			errMsg:  "KeyID is required",
		},
		{
			name: "missing signature",
			sig: &SignatureData{
				Algorithm:    AlgorithmEdDSA,
				KeyID:        "key-123",
				Signature:    []byte{},
				SignedFields: []string{"messageId"},
			},
			wantErr: true,
			errMsg:  "signature is required",
		},
		{
			name: "empty signedFields",
			sig: &SignatureData{
				Algorithm:    AlgorithmEdDSA,
				KeyID:        "key-123",
				Signature:    []byte("signature"),
				SignedFields: []string{},
			},
			wantErr: true,
			errMsg:  "signedFields cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sig.Validate()
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

func TestSecurityMetadata_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		sec     *SecurityMetadata
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid A2A mode",
			sec: &SecurityMetadata{
				Mode: ProtocolModeA2A,
			},
			wantErr: false,
		},
		{
			name: "valid SAGE mode",
			sec: &SecurityMetadata{
				Mode:      ProtocolModeSAGE,
				AgentDID:  "did:sage:eth:0x123",
				Nonce:     "nonce-789",
				Timestamp: now,
				Sequence:  1,
			},
			wantErr: false,
		},
		{
			name: "valid SAGE with signature",
			sec: &SecurityMetadata{
				Mode:      ProtocolModeSAGE,
				AgentDID:  "did:sage:eth:0x123",
				Nonce:     "nonce-789",
				Timestamp: now,
				Signature: &SignatureData{
					Algorithm:    AlgorithmEdDSA,
					KeyID:        "key-123",
					Signature:    []byte("signature"),
					SignedFields: []string{"messageId"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			sec: &SecurityMetadata{
				Mode: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid protocol mode",
		},
		{
			name: "SAGE without AgentDID",
			sec: &SecurityMetadata{
				Mode:      ProtocolModeSAGE,
				Nonce:     "nonce-789",
				Timestamp: now,
			},
			wantErr: true,
			errMsg:  "AgentDID required for SAGE mode",
		},
		{
			name: "SAGE without Nonce",
			sec: &SecurityMetadata{
				Mode:      ProtocolModeSAGE,
				AgentDID:  "did:sage:eth:0x123",
				Timestamp: now,
			},
			wantErr: true,
			errMsg:  "Nonce required for SAGE mode",
		},
		{
			name: "SAGE without Timestamp",
			sec: &SecurityMetadata{
				Mode:     ProtocolModeSAGE,
				AgentDID: "did:sage:eth:0x123",
				Nonce:    "nonce-789",
			},
			wantErr: true,
			errMsg:  "Timestamp required for SAGE mode",
		},
		{
			name: "invalid signature",
			sec: &SecurityMetadata{
				Mode:      ProtocolModeSAGE,
				AgentDID:  "did:sage:eth:0x123",
				Nonce:     "nonce-789",
				Timestamp: now,
				Signature: &SignatureData{
					Algorithm: "invalid",
					KeyID:     "key-123",
					Signature: []byte("signature"),
				},
			},
			wantErr: true,
			errMsg:  "signature validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sec.Validate()
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
