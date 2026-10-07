// SPDX-License-Identifier: LGPL-3.0-or-later
package guardcalculator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/capture"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
)

// Unit-only protected-image assertions; they are not loaded-image attestation.
type measurement struct {
	fail       atomic.Bool
	panicCheck atomic.Bool
	calls      atomic.Int64
	callback   func()
}

func (m *measurement) Check(_ context.Context, s *b.Snapshot) error {
	m.calls.Add(1)
	if m.callback != nil {
		m.callback()
	}
	if m.panicCheck.Load() {
		panic("controlled provider failure")
	}
	if m.fail.Load() || s == nil {
		return errors.New("unavailable runtime observation")
	}
	return nil
}
func encoded(v any) []byte {
	data, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return data
}
func digest(data []byte) string { d := sha256.Sum256(data); return hex.EncodeToString(d[:]) }

const alice = "did:sage:web:agent.example:alice"
const bob = "did:sage:web:agent.example:bob"

type environment struct {
	factory *Factory
	measure *measurement
	request *capture.Request
	config  b.Config
}

func fixture(t *testing.T, arguments string, config map[string]any) *environment {
	t.Helper()
	root := t.TempDir()
	store, e := capture.OpenFileStore(filepath.Join(root, "originals"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	host, e := capture.NewHost(store)
	if e != nil {
		t.Fatal(e)
	}
	request, e := host.Capture(context.Background(), [][]byte{[]byte("approved bounded arithmetic")})
	if e != nil {
		t.Fatal(e)
	}
	artifacts := filepath.Join(root, "artifacts")
	if e = os.Mkdir(artifacts, 0700); e != nil {
		t.Fatal(e)
	}
	if config == nil {
		config = map[string]any{"version": "0.10.0", "tool": "calculator", "image_path": "image.fixture", "operations": []string{"add", "divide", "multiply", "subtract"}, "absolute_operand_limit": 1000000}
	}
	rules := encoded(map[string]any{"version": "0.10.0", "recipient": bob, "keyid": alice + "#signing-1", "tool": "calculator", "arguments": json.RawMessage(arguments), "max_lifetime": 300})
	// image.fixture is explicitly a synthetic unit artifact, never a runtime image.
	data := map[string][]byte{ConfigurationPath: encoded(config), "image.fixture": []byte("unit-only image declaration"), "rules.json": rules}
	for path, content := range data {
		if e = os.WriteFile(filepath.Join(artifacts, path), content, 0600); e != nil {
			t.Fatal(e)
		}
	}
	manifest := func(names ...string) []byte {
		rows := []any{}
		for _, name := range names {
			rows = append(rows, map[string]any{"path": name, "sha256": digest(data[name])})
		}
		return encoded(map[string]any{"version": "0.10.0", "files": rows})
	}
	policy := encoded(map[string]any{"version": "0.10.0", "issuer": alice, "epoch": "00000000-0000-4000-8000-000000000001", "engine": b.Engine, "artifacts": json.RawMessage(manifest("image.fixture", "rules.json"))})
	measure := &measurement{}
	factory, e := NewFactory(Config{Measurement: measure, MaxInstances: 4})
	if e != nil {
		t.Fatal(e)
	}
	// Registered first, hence factory cleanup runs after later Operation cleanups.
	t.Cleanup(func() {
		if err := factory.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return &environment{factory, measure, request, b.Config{Directory: artifacts, Policy: policy, Manifest: manifest(ConfigurationPath, "image.fixture"), Limits: b.Limits{FileBytes: 4096, TotalBytes: 16384}, Factory: factory}}
}
func (e *environment) open(t *testing.T) *b.Operation {
	t.Helper()
	op, err := b.Open(context.Background(), e.request, e.config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := op.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return op
}

func TestActualCompiledCalculator(t *testing.T) {
	for _, row := range []struct{ arguments, expected string }{
		{`{"a":2,"b":3,"operation":"add"}`, `{"output":5,"success":true}`},
		{`{"a":2,"b":3,"operation":"subtract"}`, `{"output":-1,"success":true}`},
		{`{"a":2,"b":3,"operation":"multiply"}`, `{"output":6,"success":true}`},
		{`{"a":2,"b":4,"operation":"divide"}`, `{"output":0.5,"success":true}`},
		{`{"a":2,"b":0,"operation":"divide"}`, `{"error":"division by zero","success":false}`},
	} {
		t.Run(row.arguments, func(t *testing.T) {
			e := fixture(t, row.arguments, nil)
			op := e.open(t)
			binding, err := op.Binding(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if binding.Name != "calculator" || binding.Tool.Check(context.Background(), binding.ManifestDigest, "calculator") != nil {
				t.Fatal("binding identity")
			}
			out, err := binding.Tool.Execute(context.Background(), []byte(row.arguments))
			if err != nil || string(out) != row.expected {
				t.Fatalf("%s: %v", out, err)
			}
		})
	}
}
func TestClosedArgumentsRefuseBeforeTool(t *testing.T) {
	for _, args := range []string{`{"a":2,"b":3}`, `{"a":2,"b":3,"operation":"add","extra":true}`, `{"a":null,"b":3,"operation":"add"}`, `{"a":2,"b":null,"operation":"add"}`, `{"a":"2","b":3,"operation":"add"}`, `{"a":2,"b":3,"operation":null}`, `{"a":1000001,"b":3,"operation":"add"}`, `{"a":2,"b":-1000001,"operation":"add"}`, `{"a":2,"b":3,"operation":"power"}`} {
		t.Run(args, func(t *testing.T) {
			e := fixture(t, args, nil)
			op := e.open(t)
			binding, err := op.Binding(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if out, err := binding.Tool.Execute(context.Background(), []byte(args)); err == nil || out != nil {
				t.Fatal("accepted unsupported arguments")
			}
		})
	}
	e := fixture(t, `{"a":2,"b":3,"operation":"add"}`, nil)
	op := e.open(t)
	binding, err := op.Binding(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range []string{`{"a":2, "b":3,"operation":"add"}`, `[]`, `{"a":2,"a":2,"b":3,"operation":"add"}`} {
		if out, err := binding.Tool.Execute(context.Background(), []byte(args)); err == nil || out != nil {
			t.Fatal("accepted noncanonical input")
		}
	}
}
func TestConfigurationAndMeasurementRequired(t *testing.T) {
	for _, change := range []struct {
		field string
		value any
	}{
		{"image_path", ""}, {"image_path", ConfigurationPath}, {"image_path", "rules.json"}, {"image_path", "missing.image"}, {"extra", true}, {"version", "other"}, {"tool", "echo"}, {"operations", []string{}}, {"operations", []string{"add", "add"}}, {"operations", []string{"subtract", "add"}}, {"operations", []string{"power"}}, {"operations", []string{"add", "divide", "multiply", "subtract", "extra"}}, {"absolute_operand_limit", 0}, {"absolute_operand_limit", 1000001}, {"absolute_operand_limit", nil},
	} {
		t.Run(change.field+string(encoded(change.value)), func(t *testing.T) {
			c := map[string]any{"version": "0.10.0", "tool": "calculator", "image_path": "image.fixture", "operations": []string{"add"}, "absolute_operand_limit": 100}
			c[change.field] = change.value
			e := fixture(t, `{"a":2,"b":3,"operation":"add"}`, c)
			if op, err := b.Open(context.Background(), e.request, e.config); err == nil || op != nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
	var nilMeasurement *measurement
	for _, c := range []Config{{}, {Measurement: nilMeasurement, MaxInstances: 1}, {Measurement: &measurement{}, MaxInstances: 0}, {Measurement: &measurement{}, MaxInstances: 1025}} {
		if f, err := NewFactory(c); err == nil || f != nil {
			t.Fatal("accepted unavailable measurement or bounds")
		}
	}
	for _, mode := range []string{"uncovered", "unavailable", "panic", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			e := fixture(t, `{"a":2,"b":3,"operation":"add"}`, nil)
			ctx := context.Background()
			switch mode {
			case "uncovered":
				e.config.Manifest = encoded(map[string]any{"version": "0.10.0", "files": []any{map[string]any{"path": "image.fixture", "sha256": digest([]byte("unit-only image declaration"))}}})
			case "unavailable":
				e.measure.fail.Store(true)
			case "panic":
				e.measure.panicCheck.Store(true)
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if op, err := b.Open(ctx, e.request, e.config); err == nil || op != nil {
				t.Fatal("accepted unmeasured instance")
			}
		})
	}
}
func TestDriftAndRetirementRefuse(t *testing.T) {
	for _, mode := range []string{"measurement", "panic", "file", "close"} {
		t.Run(mode, func(t *testing.T) {
			e := fixture(t, `{"a":2,"b":3,"operation":"add"}`, nil)
			op := e.open(t)
			binding, err := op.Binding(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "measurement":
				e.measure.fail.Store(true)
			case "panic":
				e.measure.panicCheck.Store(true)
			case "file":
				if err = os.WriteFile(filepath.Join(e.config.Directory, ConfigurationPath), []byte("changed approved configuration"), 0600); err != nil {
					t.Fatal(err)
				}
			case "close":
				if err = e.factory.Close(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if out, err := binding.Tool.Execute(context.Background(), []byte(`{"a":2,"b":3,"operation":"add"}`)); err == nil || out != nil {
				t.Fatal("accepted retired instance")
			}
			e.measure.fail.Store(false)
			e.measure.panicCheck.Store(false)
			if _, err := op.Binding(context.Background()); err == nil {
				t.Fatal("silently restored retired binding")
			}
		})
	}
}
func TestFactoryBoundsAndCloseWait(t *testing.T) {
	e := fixture(t, `{"a":2,"b":3,"operation":"add"}`, nil)
	e.factory.limit = 1
	op := e.open(t)
	if other, err := b.Open(context.Background(), e.request, e.config); err == nil || other != nil {
		t.Fatal("instance ownership unbounded")
	}
	binding, err := op.Binding(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	e.measure.callback = func() { close(started); <-release }
	done := make(chan error, 1)
	go func() {
		_, err := binding.Tool.Execute(context.Background(), []byte(`{"a":2,"b":3,"operation":"add"}`))
		done <- err
	}()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if e.factory.Close(ctx) == nil {
		t.Fatal("close returned with active trusted callback")
	}
	if loaded, err := e.factory.Load(context.Background(), &b.Snapshot{}); err == nil || loaded != nil {
		t.Fatal("retiring factory loaded again")
	}
	close(release)
	if err = <-done; err == nil {
		t.Fatal("retired instance executed after waiting measurement")
	}
	e.measure.callback = nil
	if err = e.factory.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestInvalidHandlesAndCancellation(t *testing.T) {
	var factory *Factory
	var i *instance
	if loaded, err := factory.Load(context.Background(), nil); err == nil || loaded != nil {
		t.Fatal("nil factory")
	}
	if factory.Close(context.Background()) != nil {
		t.Fatal("nil close")
	}
	if i.Check(context.Background(), "", "", "") == nil {
		t.Fatal("nil instance")
	}
	if out, err := i.Execute(context.Background(), nil); err == nil || out != nil {
		t.Fatal("nil execution")
	}
	e := fixture(t, `{"a":2,"b":3,"operation":"add"}`, nil)
	if loaded, err := e.factory.Load(context.Background(), &b.Snapshot{}); err == nil || loaded != nil {
		t.Fatal("zero snapshot")
	}
	op := e.open(t)
	i = e.factory.instances[0]
	for _, row := range []struct{ policy, manifest, tool string }{{"other", i.manifest, "calculator"}, {i.policy, "other", "calculator"}, {i.policy, i.manifest, "echo"}} {
		if i.Check(context.Background(), row.policy, row.manifest, row.tool) == nil {
			t.Fatal("commitment mismatch")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := i.Execute(ctx, []byte(`{"a":2,"b":3,"operation":"add"}`)); err == nil {
		t.Fatal("canceled execution")
	}
	if i.Check(ctx, i.policy, i.manifest, "calculator") == nil {
		t.Fatal("canceled measurement")
	}
	out, err := i.Execute(context.Background(), []byte(`{"a":2,"b":3,"operation":"add"}`))
	if err != nil || !bytes.Equal(out, []byte(`{"output":5,"success":true}`)) {
		t.Fatal(err)
	}
	if _, err = op.Binding(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestImageMustBeCoveredByBothDescriptors(t *testing.T) {
	for _, scope := range []string{"policy", "component"} {
		t.Run(scope, func(t *testing.T) {
			e := fixture(t, `{"a":2,"b":3,"operation":"add"}`, nil)
			var descriptor map[string]json.RawMessage
			raw := e.config.Manifest
			if scope == "policy" {
				if err := json.Unmarshal(e.config.Policy, &descriptor); err != nil {
					t.Fatal(err)
				}
				raw = descriptor["artifacts"]
			}
			var artifacts map[string]json.RawMessage
			if err := json.Unmarshal(raw, &artifacts); err != nil {
				t.Fatal(err)
			}
			var rows []map[string]string
			if err := json.Unmarshal(artifacts["files"], &rows); err != nil {
				t.Fatal(err)
			}
			retained := rows[:0]
			for _, row := range rows {
				if row["path"] != "image.fixture" {
					retained = append(retained, row)
				}
			}
			artifacts["files"] = encoded(retained)
			if scope == "policy" {
				descriptor["artifacts"] = encoded(artifacts)
				e.config.Policy = encoded(descriptor)
			} else {
				e.config.Manifest = encoded(artifacts)
			}
			if operation, err := b.Open(context.Background(), e.request, e.config); err == nil || operation != nil {
				t.Fatal("accepted image covered by only one descriptor")
			}
		})
	}
}

func TestNonfiniteArithmeticOutputRefusesAndRetires(t *testing.T) {
	args := []byte(`{"a":1000000,"b":1e-320,"operation":"divide"}`)
	e := fixture(t, string(args), nil)
	operation := e.open(t)
	binding, err := operation.Binding(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if output, err := binding.Tool.Execute(context.Background(), args); err == nil || output != nil {
		t.Fatal("nonfinite arithmetic escaped canonical output checks")
	}
	if _, err = operation.Binding(context.Background()); err == nil {
		t.Fatal("uncertain output silently restored the instance")
	}
}
