// SPDX-License-Identifier: LGPL-3.0-or-later
package guardbinding

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

func openReceiver(t *testing.T, c Config) *Operation {
	t.Helper()
	o, err := OpenReceiver(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := o.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return o
}

// A receiver operation answers only for its exact issuer and policy
// commitment, evaluates the same rules and binds the same loaded instance.
func TestReceiverServesProvisionedMapping(t *testing.T) {
	e := environment(t)
	o := openReceiver(t, e.config)
	ctx := context.Background()
	policy, manifest, err := o.Approved(ctx, alice, o.policyDigest)
	if err != nil {
		t.Fatal(err)
	}
	pd, _ := g.PolicyCommitment(policy)
	md, _ := g.ManifestCommitment(manifest)
	if pd != o.policyDigest || md != o.manifestDigest {
		t.Fatal("descriptors do not match the provisioned commitments")
	}
	if o.Authorize(ctx, alice, "sum", []byte(`{"a":2,"b":3}`)) != nil {
		t.Fatal("approved arguments refused")
	}
	if o.Authorize(ctx, alice, "sum", []byte(`{"a":2,"b":4}`)) == nil {
		t.Fatal("other arguments authorized")
	}
	b, err := o.Binding(ctx)
	if err != nil || b.Name != "sum" || b.ManifestDigest != o.manifestDigest {
		t.Fatal("binding", err)
	}
	if _, err = g.NewReceiverPolicy(o); err != nil {
		t.Fatal(err)
	}
}

func TestReceiverRefusesIssuerRolesAndOtherCommitments(t *testing.T) {
	e := environment(t)
	o := openReceiver(t, e.config)
	ctx := context.Background()
	if _, _, err := o.Approved(ctx, bob, o.policyDigest); err == nil {
		t.Fatal("other issuer answered")
	}
	if _, _, err := o.Approved(ctx, alice, o.manifestDigest); err == nil {
		t.Fatal("other commitment answered")
	}
	if d, p, m, err := o.Bindings(ctx, alice, e.request.ID()); err == nil || d != "" || p != nil || m != nil {
		t.Fatal("receiver produced original bindings")
	}
	if o.ApproveIntent(ctx, unsigned(o, e.request)) == nil {
		t.Fatal("receiver approved an intent")
	}
	// A capture-bound operation does not act as a receiver mapping.
	issuer := e.open(t)
	if _, _, err := issuer.Approved(ctx, alice, issuer.policyDigest); err == nil {
		t.Fatal("capture-bound operation answered as a receiver")
	}
}

func TestReceiverRetiresOnDriftAndClose(t *testing.T) {
	e := environment(t)
	o := openReceiver(t, e.config)
	ctx := context.Background()
	if err := os.WriteFile(filepath.Join(e.config.Directory, "rules.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := o.Approved(ctx, alice, o.policyDigest); err == nil {
		t.Fatal("changed rules accepted")
	}
	if !o.retiring.Load() {
		t.Fatal("drift did not retire the receiver")
	}
	e2 := environment(t)
	closed, err := OpenReceiver(ctx, e2.config)
	if err != nil {
		t.Fatal(err)
	}
	if err = closed.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err = closed.Approved(ctx, alice, closed.policyDigest); err == nil {
		t.Fatal("closed receiver answered")
	}
}

func TestOpenRequiresExplicitReceiverMode(t *testing.T) {
	e := environment(t)
	ctx := context.Background()
	if o, err := Open(ctx, nil, e.config); err == nil || o != nil {
		t.Fatal("Open without a capture became a receiver")
	}
	if o, err := OpenHop(ctx, nil, e.config); err == nil || o != nil {
		t.Fatal("OpenHop without a capture accepted")
	}
	if o, err := open(ctx, e.request, e.config, "", "", true); err == nil || o != nil {
		t.Fatal("receiver mode accepted a capture")
	}
	if o, err := open(ctx, nil, e.config, testParent, alice, true); err == nil || o != nil {
		t.Fatal("receiver mode accepted a parent")
	}
	bad := e.config
	bad.Factory = nil
	if o, err := OpenReceiver(ctx, bad); err == nil || o != nil {
		t.Fatal("receiver without a factory accepted")
	}
}
