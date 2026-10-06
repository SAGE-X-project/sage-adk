// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/toolhost"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

type inertTool struct{}

func (*inertTool) Check(context.Context, string, string) error     { return nil }
func (*inertTool) Execute(context.Context, []byte) ([]byte, error) { return []byte(`{}`), nil }
func TestInvalidConfigurationDoesNotCreateStorage(t *testing.T) {
	for _, name := range []string{"empty", "missing-tool", "typed-nil-tool", "name", "reserved-name", "digest", "uppercase-digest", "duplicate", "too-many", "services"} {
		t.Run(name, func(t *testing.T) {
			bindings := []toolhost.Binding{{Name: "sum", ManifestDigest: strings.Repeat("a", 64), Tool: &inertTool{}}}
			switch name {
			case "empty":
				bindings = nil
			case "missing-tool":
				bindings[0].Tool = nil
			case "typed-nil-tool":
				var x *inertTool
				bindings[0].Tool = x
			case "name":
				bindings[0].Name = "../sum"
			case "reserved-name":
				bindings[0].Name = "sage_secure_call"
			case "digest":
				bindings[0].ManifestDigest = "bad"
			case "uppercase-digest":
				bindings[0].ManifestDigest = strings.Repeat("A", 64)
			case "duplicate":
				bindings = append(bindings, bindings[0])
			case "too-many":
				bindings = make([]toolhost.Binding, 1025)
			}
			path := filepath.Join(t.TempDir(), "ledger")
			h, e := toolhost.Open(path, true, "did:sage:web:agent.example:bob", toolhost.Services{}, bindings, g.MCPHostBounds{})
			if e == nil || h != nil {
				t.Fatal("invalid configuration accepted")
			}
			if _, e = os.Stat(path); !os.IsNotExist(e) {
				t.Fatal("invalid configuration touched ledger")
			}
		})
	}
	var h *toolhost.Host
	if e := h.Close(context.Background()); e == nil {
		t.Fatal("nil host")
	}
	var zero toolhost.Host
	if e := zero.Close(context.Background()); e == nil {
		t.Fatal("zero host")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := zero.Connect(ctx, nil, g.MCPConnectionConfig{}, nil); e == nil {
		t.Fatal("zero connection")
	}
	if e := zero.Serve(ctx, nil, 1, g.MCPConnectionConfig{}, nil); e == nil {
		t.Fatal("zero serve")
	}
}
