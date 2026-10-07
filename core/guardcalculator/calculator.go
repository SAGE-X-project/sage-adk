// SPDX-License-Identifier: LGPL-3.0-or-later
// Package guardcalculator binds ADK's statically compiled calculator to approved
// configuration and mandatory protected runtime measurement. It is not a dynamic
// code loader, process isolation service or deployment attestation.
package guardcalculator

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"sync/atomic"

	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	"github.com/sage-x-project/sage-adk/core/tools"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// ConfigurationPath is an independently approved component artifact, never a
// peer-supplied dispatch override. Its exact bytes must be in the manifest.
const ConfigurationPath = "calculator.json"

// Measurement must verify the actual protected running host image, exact policy
// evaluator, this factory, compiled calculator and their dependencies against
// the approved snapshot. Hashing source files or echoing supplied digests is not
// sufficient. The implementation is trusted, bounded, cancellation-aware,
// concurrent-safe and non-reentrant; it receives no signing or host authority.
type Measurement interface {
	Check(context.Context, *b.Snapshot) error
}

// Config selects protected measurement and bounds retained instances (1..1024).
// No model/plugin/peer may provision it. Measurement must outlive the factory.
type Config struct {
	Measurement  Measurement
	MaxInstances int
}

// Factory creates private calculator instances, never a mutable Tool Registry.
// Only use Load as guardbinding.Open's protected Factory. It does not provide
// signing or execution authorization. Retire hosts and Operations before Close.
type Factory struct {
	gate        chan struct{}
	retiring    atomic.Bool
	measurement Measurement
	limit       int
	instances   []*instance
}

type configuration struct {
	ImagePath  string   `json:"image_path"`
	Version    string   `json:"version"`
	Tool       string   `json:"tool"`
	Operations []string `json:"operations"`
	Limit      float64  `json:"absolute_operand_limit"`
}

type instance struct {
	owner            *Factory
	gate             chan struct{}
	retiring         atomic.Bool
	measurement      Measurement
	snapshot         *b.Snapshot
	policy, manifest string
	config           configuration
	tool             tools.Tool
}

var _ b.Factory = (*Factory)(nil)
var _ b.Instance = (*instance)(nil)

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
func gate() chan struct{} { c := make(chan struct{}, 1); c <- struct{}{}; return c }
func enter(ctx context.Context, c chan struct{}) error {
	if !active(ctx) || c == nil {
		return b.ErrDenied
	}
	select {
	case <-ctx.Done():
		return b.ErrDenied
	case <-c:
	}
	if !active(ctx) {
		c <- struct{}{}
		return b.ErrDenied
	}
	return nil
}

// NewFactory refuses absent measurement or unbounded instance ownership.
func NewFactory(c Config) (*Factory, error) {
	if absent(c.Measurement) || c.MaxInstances < 1 || c.MaxInstances > 1024 {
		return nil, b.ErrDenied
	}
	return &Factory{gate: gate(), measurement: c.Measurement, limit: c.MaxInstances}, nil
}

// Load checks the already-verified snapshot and runtime before constructing one
// fixed compiled tool. It never opens paths or loads code from snapshot files.
// Configuration fixes allowed operations and a finite operand bound; the schema
// is exactly operation (string), a (number), b (number), with no defaults.
func (f *Factory) Load(ctx context.Context, snapshot *b.Snapshot) (loaded b.Instance, err error) {
	if f == nil || f.retiring.Load() {
		return nil, b.ErrDenied
	}
	if enter(ctx, f.gate) != nil {
		return nil, b.ErrDenied
	}
	defer func() { f.gate <- struct{}{} }()
	defer func() {
		if recover() != nil {
			loaded = nil
			err = b.ErrDenied
		}
	}()
	if f.retiring.Load() || snapshot == nil || absent(f.measurement) || len(f.instances) >= f.limit {
		return nil, b.ErrDenied
	}
	pd, e := g.PolicyCommitment(snapshot.Policy())
	if e != nil {
		return nil, b.ErrDenied
	}
	md, e := g.ManifestCommitment(snapshot.Manifest())
	if e != nil {
		return nil, b.ErrDenied
	}
	var manifest struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if json.Unmarshal(snapshot.Manifest(), &manifest) != nil {
		return nil, b.ErrDenied
	}
	covered := false
	for _, file := range manifest.Files {
		covered = covered || file.Path == ConfigurationPath
	}
	if !covered {
		return nil, b.ErrDenied
	}
	artifacts := snapshot.Artifacts()
	if _, e = g.VerifyManifest(snapshot.Manifest(), componentArtifacts(artifacts, manifest.Files)); e != nil {
		return nil, b.ErrDenied
	}
	var raw []byte
	for _, a := range artifacts {
		if a.Path == ConfigurationPath {
			raw = a.Bytes
		}
	}
	config, e := parseConfiguration(raw)
	if e != nil {
		return nil, b.ErrDenied
	}
	var policy struct {
		Engine    string `json:"engine"`
		Artifacts struct {
			Files []struct {
				Path string `json:"path"`
			} `json:"files"`
		} `json:"artifacts"`
	}
	if json.Unmarshal(snapshot.Policy(), &policy) != nil || policy.Engine != b.Engine {
		return nil, b.ErrDenied
	}
	componentImage, evaluatorImage := false, false
	for _, file := range manifest.Files {
		componentImage = componentImage || file.Path == config.ImagePath
	}
	for _, file := range policy.Artifacts.Files {
		evaluatorImage = evaluatorImage || file.Path == config.ImagePath
	}
	if !componentImage || !evaluatorImage {
		return nil, b.ErrDenied
	}
	// No tool callbacks run until the trusted loaded-runtime provider succeeds.
	if f.measurement.Check(ctx, snapshot) != nil || !active(ctx) || f.retiring.Load() {
		return nil, b.ErrDenied
	}
	i := &instance{owner: f, gate: gate(), measurement: f.measurement, snapshot: snapshot, policy: pd, manifest: md, config: config, tool: tools.CalculatorTool()}
	f.instances = append(f.instances, i)
	return i, nil
}

func componentArtifacts(artifacts []g.Artifact, files []struct {
	Path string `json:"path"`
}) []g.Artifact {
	result := make([]g.Artifact, 0, len(files))
	for _, file := range files {
		for _, a := range artifacts {
			if a.Path == file.Path {
				result = append(result, a)
			}
		}
	}
	return result
}

func parseConfiguration(raw []byte) (configuration, error) {
	canonical, e := g.Canonicalize(raw)
	if e != nil {
		return configuration{}, b.ErrDenied
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(canonical, &fields) != nil || len(fields) != 5 {
		return configuration{}, b.ErrDenied
	}
	for _, key := range []string{"version", "tool", "operations", "absolute_operand_limit", "image_path"} {
		if fields[key] == nil {
			return configuration{}, b.ErrDenied
		}
	}
	var c configuration
	if json.Unmarshal(canonical, &c) != nil || c.Version != "0.10.0" || c.Tool != "calculator" || c.ImagePath == "" || c.ImagePath == ConfigurationPath || c.ImagePath == "rules.json" || len(c.Operations) < 1 || len(c.Operations) > 4 || math.IsNaN(c.Limit) || math.IsInf(c.Limit, 0) || c.Limit <= 0 || c.Limit > 1000000 {
		return configuration{}, b.ErrDenied
	}
	for n, operation := range c.Operations {
		if operation != "add" && operation != "subtract" && operation != "multiply" && operation != "divide" {
			return configuration{}, b.ErrDenied
		}
		if n > 0 && c.Operations[n-1] >= operation {
			return configuration{}, b.ErrDenied
		}
	}
	return c, nil
}

func (i *instance) checked(ctx context.Context) error {
	if i.owner == nil || i.owner.retiring.Load() || i.retiring.Load() || absent(i.tool) || i.measurement.Check(ctx, i.snapshot) != nil || !active(ctx) || i.owner.retiring.Load() {
		i.retiring.Store(true)
		return b.ErrDenied
	}
	return nil
}

// Check binds approved commitments to the same private compiled instance. A
// failed runtime observation permanently retires that instance, never reenrols it.
func (i *instance) Check(ctx context.Context, policy, manifest, tool string) (err error) {
	if i == nil || i.owner == nil || i.owner.retiring.Load() || i.retiring.Load() || enter(ctx, i.gate) != nil {
		return b.ErrDenied
	}
	defer func() { i.gate <- struct{}{} }()
	defer func() {
		if recover() != nil {
			i.retiring.Store(true)
			err = b.ErrDenied
		}
	}()
	if policy != i.policy || manifest != i.manifest || tool != "calculator" {
		return b.ErrDenied
	}
	return i.checked(ctx)
}

// Execute is a trusted loader capability, not an unsigned model-facing API.
// guardbinding's private wrapper and native admitted worker own its use. Runtime
// verification repeats immediately before the bounded arithmetic callback.
func (i *instance) Execute(ctx context.Context, raw []byte) (output []byte, err error) {
	if i == nil || i.owner == nil || i.owner.retiring.Load() || i.retiring.Load() || enter(ctx, i.gate) != nil {
		return nil, b.ErrDenied
	}
	defer func() { i.gate <- struct{}{} }()
	defer func() {
		if recover() != nil {
			i.retiring.Store(true)
			output = nil
			err = b.ErrDenied
		}
	}()
	canonical, e := g.Canonicalize(raw)
	if e != nil || !bytes.Equal(canonical, raw) {
		return nil, b.ErrDenied
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(canonical, &fields) != nil || len(fields) != 3 {
		return nil, b.ErrDenied
	}
	var operation string
	var a, c float64
	if json.Unmarshal(fields["operation"], &operation) != nil || json.Unmarshal(fields["a"], &a) != nil || json.Unmarshal(fields["b"], &c) != nil || bytes.Equal(fields["a"], []byte("null")) || bytes.Equal(fields["b"], []byte("null")) || fields["a"] == nil || fields["b"] == nil {
		return nil, b.ErrDenied
	}
	index := sort.SearchStrings(i.config.Operations, operation)
	if index == len(i.config.Operations) || i.config.Operations[index] != operation || math.IsNaN(a) || math.IsNaN(c) || math.IsInf(a, 0) || math.IsInf(c, 0) || math.Abs(a) > i.config.Limit || math.Abs(c) > i.config.Limit {
		return nil, b.ErrDenied
	}
	if i.checked(ctx) != nil {
		return nil, b.ErrDenied
	}
	result, e := i.tool.Execute(ctx, map[string]interface{}{"operation": operation, "a": a, "b": c})
	if e != nil || result == nil || !active(ctx) {
		i.retiring.Store(true)
		return nil, b.ErrDenied
	}
	encoded, e := json.Marshal(result)
	if e != nil {
		i.retiring.Store(true)
		return nil, b.ErrDenied
	}
	output, e = g.Canonicalize(encoded)
	if e != nil || !active(ctx) {
		i.retiring.Store(true)
		return nil, b.ErrDenied
	}
	return output, nil
}

// Close immediately refuses new loads/callbacks and waits for accepted bounded
// arithmetic to finish. Cancellation retains ownership; retry with a fresh context.
func (f *Factory) Close(ctx context.Context) error {
	if f == nil {
		return nil
	}
	f.retiring.Store(true)
	if enter(ctx, f.gate) != nil {
		return b.ErrDenied
	}
	defer func() { f.gate <- struct{}{} }()
	for _, i := range f.instances {
		i.retiring.Store(true)
	}
	for _, i := range f.instances {
		if enter(ctx, i.gate) != nil {
			return b.ErrDenied
		}
		i.tool = nil
		i.snapshot = nil
		i.gate <- struct{}{}
	}
	f.instances = nil
	return nil
}
