// SPDX-License-Identifier: LGPL-3.0-or-later
package guardbinding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/sage-x-project/sage-adk/core/capture"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

var name = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)
var key = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

func active(ctx context.Context) bool { return ctx != nil && ctx.Err() == nil }
func absent(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

// Open verifies independent approved descriptors and exact file snapshots before
// publishing an operation. The selected root must be protected local storage;
// ordinary directory permissions do not establish attacker-resistant isolation.
func Open(ctx context.Context, request *capture.Request, c Config) (*Operation, error) {
	return open(ctx, request, c, "", "")
}

// OpenHop binds independent approved policy/artifacts and one immutable loader
// instance to a retained original from an actually running native parent. The
// policy issuer must equal that authenticated inbound recipient. No permission
// or key is inherited. Keep this capability in the protected native coordinator;
// loaders receive only Snapshot, never Invocation, admission or signing custody.
func OpenHop(ctx context.Context, request *capture.HopRequest, c Config) (*Operation, error) {
	if !active(ctx) || request == nil {
		return nil, ErrDenied
	}
	inputs, err := request.Inputs(ctx)
	if err != nil || len(inputs) != 1 {
		return nil, ErrDenied
	}
	var envelope struct {
		Intent struct {
			Recipient string `json:"recipient"`
			CallID    string `json:"call_id"`
		} `json:"intent"`
	}
	if json.Unmarshal(inputs[0], &envelope) != nil || envelope.Intent.Recipient == "" ||
		!uuid.MatchString(request.ParentCallID()) || envelope.Intent.CallID != request.ParentCallID() {
		return nil, ErrDenied
	}
	return open(ctx, request, c, request.ParentCallID(), envelope.Intent.Recipient)
}

func open(ctx context.Context, request retainedInput, c Config, parent, issuer string) (operation *Operation, err error) {
	var root *os.Root
	defer func() {
		if recover() != nil {
			operation = nil
			err = ErrDenied
		}
		if err != nil && root != nil {
			_ = root.Close()
		}
	}()
	if !active(ctx) || absent(request) || absent(c.Factory) || c.Limits.FileBytes < 1 || c.Limits.FileBytes > 64<<20 || c.Limits.TotalBytes < 1 || c.Limits.TotalBytes > 256<<20 {
		return nil, ErrDenied
	}
	if _, e := request.Inputs(ctx); e != nil {
		return nil, ErrDenied
	}
	policy, e := g.Canonicalize(c.Policy)
	if e != nil {
		return nil, ErrDenied
	}
	pd, e := g.PolicyCommitment(policy)
	if e != nil {
		return nil, ErrDenied
	}
	md, e := g.ManifestCommitment(c.Manifest)
	if e != nil {
		return nil, ErrDenied
	}
	manifest, e := jcs.Canonicalize(c.Manifest)
	if e != nil {
		return nil, ErrDenied
	}
	var p struct {
		Issuer    string     `json:"issuer"`
		Engine    string     `json:"engine"`
		Artifacts descriptor `json:"artifacts"`
	}
	var m descriptor
	if json.Unmarshal(policy, &p) != nil || json.Unmarshal(manifest, &m) != nil || p.Engine != Engine || len(m.Files) == 0 || (issuer != "" && p.Issuer != issuer) {
		return nil, ErrDenied
	}
	entries := map[string]string{}
	hasRules := false
	for _, f := range append(p.Artifacts.Files, m.Files...) {
		if digest, exists := entries[f.Path]; exists && digest != f.SHA256 {
			return nil, ErrDenied
		}
		entries[f.Path] = f.SHA256
	}
	for _, f := range p.Artifacts.Files {
		hasRules = hasRules || f.Path == "rules.json"
	}
	if !hasRules {
		return nil, ErrDenied
	}
	root, e = os.OpenRoot(c.Directory)
	if e != nil {
		return nil, ErrDenied
	}
	o := &Operation{root: root, request: request, parentID: parent, limits: c.Limits, policy: policy, manifest: manifest, policyDigest: pd, manifestDigest: md, issuer: p.Issuer, gate: make(chan struct{}, 1)}
	o.gate <- struct{}{}
	for path, digest := range entries {
		o.entries = append(o.entries, file{path, digest})
	}
	sort.Slice(o.entries, func(i, j int) bool { return o.entries[i].Path < o.entries[j].Path })
	artifacts, e := o.read(ctx)
	if e != nil {
		return nil, ErrDenied
	}
	for _, f := range artifacts {
		if f.Path == "rules.json" {
			o.rules, e = parseRules(f.Bytes, p.Issuer, policy)
			if e != nil {
				return nil, ErrDenied
			}
		}
	}
	snapshot := &Snapshot{artifacts: artifacts, policy: policy, manifest: manifest}
	if _, e = request.Inputs(ctx); e != nil {
		return nil, ErrDenied
	}
	o.instance, e = c.Factory.Load(ctx, snapshot)
	if e != nil || absent(o.instance) || o.check(ctx) != nil {
		return nil, ErrDenied
	}
	return o, nil
}
func parseRules(raw []byte, issuer string, policy []byte) (rules, error) {
	canonical, e := g.Canonicalize(raw)
	if e != nil {
		return rules{}, ErrDenied
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(canonical, &m) != nil || len(m) != 6 {
		return rules{}, ErrDenied
	}
	for _, k := range []string{"version", "recipient", "keyid", "tool", "arguments", "max_lifetime"} {
		if m[k] == nil {
			return rules{}, ErrDenied
		}
	}
	var r rules
	if json.Unmarshal(m["version"], &r.Version) != nil || json.Unmarshal(m["recipient"], &r.Recipient) != nil || json.Unmarshal(m["keyid"], &r.KeyID) != nil || json.Unmarshal(m["tool"], &r.Tool) != nil || json.Unmarshal(m["max_lifetime"], &r.Lifetime) != nil {
		return rules{}, ErrDenied
	}
	r.Arguments = append([]byte(nil), m["arguments"]...)
	if r.Version != "0.10.0" || !name.MatchString(r.Tool) || r.Tool == "sage_secure_call" || !strings.HasPrefix(r.KeyID, issuer+"#") || !key.MatchString(strings.TrimPrefix(r.KeyID, issuer+"#")) || len(r.Arguments) == 0 || r.Arguments[0] != '{' || r.Lifetime < 1 || r.Lifetime > 300 {
		return rules{}, ErrDenied
	}
	// Reuse the core DID grammar rather than accepting a separate local dialect.
	var p map[string]any
	if json.Unmarshal(policy, &p) != nil {
		return rules{}, ErrDenied
	}
	p["issuer"] = r.Recipient
	b, _ := json.Marshal(p)
	if _, e = g.PolicyCommitment(b); e != nil {
		return rules{}, ErrDenied
	}
	return r, nil
}
func (o *Operation) read(ctx context.Context) ([]g.Artifact, error) {
	result := make([]g.Artifact, 0, len(o.entries))
	total := int64(0)
	for _, entry := range o.entries {
		if !active(ctx) {
			return nil, ErrDenied
		}
		path := ""
		parts := strings.Split(entry.Path, "/")
		for i, part := range parts {
			path = filepath.Join(path, part)
			info, e := o.root.Lstat(path)
			if e != nil || info.Mode()&os.ModeSymlink != 0 || (i < len(parts)-1 && !info.IsDir()) || (i == len(parts)-1 && (!info.Mode().IsRegular() || info.Size() > o.limits.FileBytes || info.Size() > o.limits.TotalBytes-total)) {
				return nil, ErrDenied
			}
		}
		b, e := readFile(o.root, entry.Path, min(o.limits.FileBytes, o.limits.TotalBytes-total))
		if e != nil || !active(ctx) {
			return nil, ErrDenied
		}
		total += int64(len(b))
		// The single-file manifest validates exact bytes using the existing core hash.
		one, _ := json.Marshal(descriptor{Version: "0.10.0", Files: []file{entry}})
		if _, e = g.VerifyManifest(one, []g.Artifact{{Path: entry.Path, Bytes: b}}); e != nil {
			return nil, ErrDenied
		}
		result = append(result, g.Artifact{Path: entry.Path, Bytes: b})
	}
	return result, nil
}
func readFile(root *os.Root, path string, limit int64) (b []byte, err error) {
	f, e := openArtifact(root, path)
	if e != nil {
		return nil, ErrDenied
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, ErrDenied
	}
	b, err = io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, ErrDenied
	}
	return b, nil
}
func (o *Operation) check(ctx context.Context) error {
	if !active(ctx) {
		return ErrDenied
	}
	if _, e := o.request.Inputs(ctx); e != nil {
		if active(ctx) {
			o.retiring.Store(true)
		}
		return ErrDenied
	}
	if _, e := o.read(ctx); e != nil {
		if active(ctx) {
			o.retiring.Store(true)
		}
		return ErrDenied
	}
	if o.instance.Check(ctx, o.policyDigest, o.manifestDigest, o.rules.Tool) != nil || !active(ctx) {
		if active(ctx) {
			o.retiring.Store(true)
		}
		return ErrDenied
	}
	// Attestation is an external trusted callback: recheck files and capture
	// after it, too, before returning a current policy/measurement snapshot.
	if _, e := o.read(ctx); e != nil {
		if active(ctx) {
			o.retiring.Store(true)
		}
		return ErrDenied
	}
	if _, e := o.request.Inputs(ctx); e != nil {
		if active(ctx) {
			o.retiring.Store(true)
		}
		return ErrDenied
	}
	return nil
}
func canonicalObject(raw []byte) ([]byte, error) {
	b, e := g.Canonicalize(raw)
	if e != nil || len(b) == 0 || b[0] != '{' || !bytes.Equal(raw, b) {
		return nil, ErrDenied
	}
	return b, nil
}
