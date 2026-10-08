// SPDX-License-Identifier: LGPL-3.0-or-later
package toolhost_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/sage-x-project/sage-adk/core/capture"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	calc "github.com/sage-x-project/sage-adk/core/guardcalculator"
	"github.com/sage-x-project/sage-adk/core/toolhost"
)

// This fixture keeps both downstream issuer and receiver in one test process.
// Independent protected policy/custody/measurement deployments are not supplied.
// The native coordinator opens a concrete operation after receiving actual
// Invocation. The fixed receiver shim sees only its private loaded-tool binding.
type approvedHopFixture struct {
	config      b.Config
	factory     *calc.Factory
	measurement *approvedHopMeasurement
	operation   atomic.Pointer[b.Operation]
	binding     atomic.Pointer[toolhost.Binding]
	effects     atomic.Int64
	mode        string
}

type approvedHopMeasurement struct {
	enabled atomic.Bool
	checks  atomic.Int64
}

// Deliberately synthetic loaded-runtime appraisal, never a production provider.
func (m *approvedHopMeasurement) Check(ctx context.Context, snapshot *b.Snapshot) error {
	m.checks.Add(1)
	if ctx.Err() != nil || !m.enabled.Load() || snapshot == nil {
		return errors.New("fixture measurement unavailable")
	}
	return nil
}

func newApprovedHopFixture(t *testing.T, env *fixtureEnvironment, mode string) *approvedHopFixture {
	t.Helper()
	directory := filepath.Join(env.root, "approved-hop-artifacts")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"image.fixture":        []byte("synthetic protected image appraisal"),
		"rules.json":           fixtureJSON(map[string]any{"version": "0.10.0", "recipient": fixtureAlice, "keyid": fixtureBob + "#signing-1", "tool": "calculator", "arguments": map[string]any{"a": 2, "b": 3, "operation": "add"}, "max_lifetime": 300}),
		calc.ConfigurationPath: fixtureJSON(map[string]any{"version": "0.10.0", "tool": "calculator", "image_path": "image.fixture", "operations": []string{"add"}, "absolute_operand_limit": 100}),
	}
	for path, data := range files {
		if err := os.WriteFile(filepath.Join(directory, path), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	descriptor := func(names ...string) []byte {
		rows := []any{}
		for _, name := range names {
			rows = append(rows, map[string]any{"path": name, "sha256": fixtureHash(files[name])})
		}
		return fixtureJSON(map[string]any{"version": "0.10.0", "files": rows})
	}
	policy := fixtureJSON(map[string]any{"version": "0.10.0", "issuer": fixtureBob, "epoch": "00000000-0000-4000-8000-000000000012", "engine": b.Engine, "artifacts": json.RawMessage(descriptor("image.fixture", "rules.json"))})
	measurement := &approvedHopMeasurement{}
	measurement.enabled.Store(mode != "approved-measurement-denied")
	factory, err := calc.NewFactory(calc.Config{Measurement: measurement, MaxInstances: 1})
	if err != nil {
		t.Fatal(err)
	}
	return &approvedHopFixture{mode: mode, factory: factory, measurement: measurement, config: b.Config{Directory: directory, Policy: policy, Manifest: descriptor(calc.ConfigurationPath, "image.fixture"), Limits: b.Limits{FileBytes: 4096, TotalBytes: 16384}, Factory: factory}}
}

func (f *approvedHopFixture) open(ctx context.Context, request *capture.HopRequest) (*b.Operation, error) {
	// A valid foreign policy issuer must fail before invoking the loader.
	foreign := f.config
	var fields map[string]any
	if json.Unmarshal(foreign.Policy, &fields) != nil {
		return nil, errors.New("fixture policy")
	}
	fields["issuer"] = fixtureAlice
	foreign.Policy = fixtureJSON(fields)
	before := f.measurement.checks.Load()
	wrong, err := b.OpenHop(ctx, request, foreign)
	if err == nil || wrong != nil || f.measurement.checks.Load() != before {
		return nil, errors.New("foreign policy reached loader")
	}
	operation, err := b.OpenHop(ctx, request, f.config)
	if err != nil {
		return nil, err
	}
	binding, err := operation.Binding(ctx)
	if err != nil {
		_ = operation.Close(context.Background())
		return nil, err
	}
	if !f.operation.CompareAndSwap(nil, operation) {
		_ = operation.Close(context.Background())
		return nil, errors.New("fixture operation replaced")
	}
	f.binding.Store(&binding)
	if f.mode == "approved-retired" {
		if err := operation.Close(ctx); err != nil {
			return nil, err
		}
	}
	return operation, nil
}

func (f *approvedHopFixture) Check(ctx context.Context, manifest, tool string) error {
	binding := f.binding.Load()
	if binding == nil {
		return errors.New("fixed fixture binding unavailable")
	}
	return binding.Tool.Check(ctx, manifest, tool)
}
func (f *approvedHopFixture) Execute(ctx context.Context, args []byte) ([]byte, error) {
	binding := f.binding.Load()
	if binding == nil {
		return nil, errors.New("fixed fixture binding unavailable")
	}
	f.effects.Add(1) // Count every attempted protected receiver callback, even refusal.
	return binding.Tool.Execute(ctx, args)
}

func TestApprovedHopOperationNativeRuntime(t *testing.T) {
	for _, mode := range []string{"approved-allowed", "approved-policy-denied", "approved-measurement-denied", "approved-retired", "approved-upstream-policy-loss"} {
		t.Run(mode, func(t *testing.T) { runAdmittedHop(t, mode) })
	}
}
