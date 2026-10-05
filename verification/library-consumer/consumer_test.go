// SPDX-License-Identifier: LGPL-3.0-or-later

package consumer_test

import (
	"context"
	"crypto/ed25519"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/adapters/a2a"
	sage "github.com/sage-x-project/sage-adk/adapters/sage"
	"github.com/sage-x-project/sage-adk/adapters/sage/configmodel"
	"github.com/sage-x-project/sage-adk/builder"
	"github.com/sage-x-project/sage-adk/core/agent"
	"github.com/sage-x-project/sage-adk/pkg/types"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
)

func TestPublicKeyAndAgentAPIs(t *testing.T) {
	km := sage.NewKeyManager()
	key, err := km.Generate()
	if err != nil {
		t.Fatal(err)
	}
	for _, format := range []crypto.KeyFormat{crypto.KeyFormatPEM, crypto.KeyFormatJWK} {
		t.Run(string(format), func(t *testing.T) {
			data, err := km.ExportKeyPair(key, format)
			if err != nil {
				t.Fatal(err)
			}
			imported, err := km.ImportKeyPair(data, format)
			if err != nil {
				t.Fatal(err)
			}
			private, err := km.ExtractEd25519PrivateKey(imported)
			if err != nil {
				t.Fatal(err)
			}
			public, err := km.ExtractEd25519PublicKey(key)
			if err != nil {
				t.Fatal(err)
			}
			body := []byte("inert compatibility fixture")
			if !ed25519.Verify(public, body, ed25519.Sign(private, body)) {
				t.Fatal("key roundtrip changed the key")
			}
		})
	}
	cfg := &sage.Config{Config: &configmodel.Config{Environment: "development"}}
	if cfg.Environment != "development" {
		t.Fatal("embedded options unavailable")
	}
	ag, err := builder.NewAgent("consumer").WithDescription("inert consumer").OnMessage(
		func(_ context.Context, msg agent.MessageContext) error { return msg.Reply("fixed response") },
	).Build()
	if err != nil {
		t.Fatal(err)
	}
	response, err := ag.Process(context.Background(), types.NewMessage(types.MessageRoleUser, []types.Part{types.NewTextPart("fixed input")}))
	if err != nil {
		t.Fatal(err)
	}
	if response == nil || len(response.Parts) != 1 || response.Parts[0].(*types.TextPart).Text != "fixed response" {
		t.Fatal("agent processing failed")
	}
}

func TestLegacyA2ALoopback(t *testing.T) {
	var calls atomic.Int32
	srv, err := a2a.NewServer(&a2a.ServerConfig{
		AgentName: "loopback-fixture", AgentURL: "http://127.0.0.1/",
		MessageHandler: func(_ context.Context, msg agent.MessageContext) error {
			if msg.Text() != "inert input" {
				t.Errorf("unexpected request text: %q", msg.Text())
			}
			calls.Add(1)
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	loopback := httptest.NewServer(srv.Handler())
	defer loopback.Close()
	cl, err := a2a.NewClient(loopback.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := cl.SendMessage(ctx, types.NewMessage(types.MessageRoleUser, []types.Part{types.NewTextPart("inert input")}))
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || response == nil {
		t.Fatalf("calls=%d response=%v", calls.Load(), response)
	}
	// The existing A2A reply path is a placeholder: this is packaging/transport
	// evidence only, not a SAGE 0.10.0 protection or reply-content verdict.
}
