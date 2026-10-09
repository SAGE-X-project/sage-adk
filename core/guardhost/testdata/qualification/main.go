//go:build linux || darwin

// SPDX-License-Identifier: LGPL-3.0-or-later

// Command qualification runs the guardhost caller and receiver as separate
// processes for one benign 2+3 calculator call. It is a test-only program:
// its Registry Source reads a local JSON file and its calculator measurement
// accepts any snapshot. Neither is authoritative, observed or isolated, so a
// successful run is not Registry, measurement or deployment evidence.
//
//	qualification setup    -dir DIR -alice HEX -bob HEX -kem HEX
//	qualification receiver -config FILE
//	qualification caller   -config FILE
//	qualification clockwatch -for DURATION
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	calc "github.com/sage-x-project/sage-adk/core/guardcalculator"
	"github.com/sage-x-project/sage-adk/core/guardhost"
	"github.com/sage-x-project/sage-adk/core/guardservices"
	"github.com/sage-x-project/sage-adk/core/guardsigner"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
	r "github.com/sage-x-project/sage/pkg/agent/registry010"
)

const (
	alice    = "did:sage:web:agent.example:alice"
	bob      = "did:sage:web:agent.example:bob"
	source   = "synthetic-qualification-fixture"
	registry = "web:agent.example"
)

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("usage: qualification setup|receiver|caller|clockwatch [flags]"))
	}
	var err error
	switch os.Args[1] {
	case "setup":
		err = setup(os.Args[2:])
	case "receiver":
		err = receiver(os.Args[2:])
	case "caller":
		err = caller(os.Args[2:])
	case "clockwatch":
		err = clockwatch(os.Args[2:])
	default:
		err = errors.New("unknown command")
	}
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "qualification:", err)
	os.Exit(1)
}

// clockwatch samples the wall clock continuously and reports every backward
// step. The host clock refuses any regression, so a stepping environment
// cannot keep a protected host running.
func clockwatch(args []string) error {
	fs := flag.NewFlagSet("clockwatch", flag.ContinueOnError)
	d := fs.Duration("for", time.Minute, "observation duration")
	if err := fs.Parse(args); err != nil {
		return err
	}
	start := time.Now()
	last := start.UnixNano()
	steps := 0
	for time.Since(start) < *d {
		now := time.Now().UnixNano()
		if now < last {
			steps++
			fmt.Printf("clock backward %.3f ms at +%s\n", float64(last-now)/1e6, time.Since(start).Round(time.Millisecond))
		}
		last = now
	}
	fmt.Printf("clock observed %s backward-steps=%d\n", *d, steps)
	return nil
}

func hash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func mustJSON(v any) []byte {
	out, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return out
}

// setup writes the calculator artifacts, the policy and manifest descriptors
// and the synthetic Registry records for both identities.
func setup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	dir := fs.String("dir", "", "output directory")
	aliceKey := fs.String("alice", "", "Alice signing public key hex")
	bobKey := fs.String("bob", "", "Bob signing public key hex")
	kem := fs.String("kem", "", "Bob X25519 public key hex")
	if err := fs.Parse(args); err != nil || *dir == "" || *aliceKey == "" || *bobKey == "" || *kem == "" {
		return errors.New("setup requires -dir, -alice, -bob and -kem")
	}
	args2 := json.RawMessage(`{"a":2,"b":3,"operation":"add"}`)
	artifacts := map[string][]byte{
		"rules.json":           mustJSON(map[string]any{"version": "0.10.0", "recipient": bob, "keyid": alice + "#signing-1", "tool": "calculator", "arguments": args2, "max_lifetime": 300}),
		calc.ConfigurationPath: mustJSON(map[string]any{"version": "0.10.0", "tool": "calculator", "image_path": "image.fixture", "operations": []string{"add"}, "absolute_operand_limit": 100}),
		"image.fixture":        []byte("synthetic runtime image observation"),
	}
	files := func(names ...string) []byte {
		rows := []any{}
		for _, n := range names {
			rows = append(rows, map[string]any{"path": n, "sha256": hash(artifacts[n])})
		}
		return mustJSON(map[string]any{"version": "0.10.0", "files": rows})
	}
	out := map[string][]byte{
		"policy.json":   mustJSON(map[string]any{"version": "0.10.0", "issuer": alice, "epoch": "00000000-0000-4000-8000-000000000001", "engine": b.Engine, "artifacts": json.RawMessage(files("image.fixture", "rules.json"))}),
		"manifest.json": files(calc.ConfigurationPath, "image.fixture"),
		"registry.json": mustJSON(map[string]string{alice: *aliceKey, bob: *bobKey, "kem": *kem}),
	}
	if err := os.MkdirAll(filepath.Join(*dir, "artifacts"), 0755); err != nil {
		return err
	}
	for name, data := range artifacts {
		if err := os.WriteFile(filepath.Join(*dir, "artifacts", name), data, 0644); err != nil {
			return err
		}
	}
	for name, data := range out {
		if err := os.WriteFile(filepath.Join(*dir, name), data, 0644); err != nil {
			return err
		}
	}
	return nil
}

// fileSource reads a local JSON file on every Read. Its readiness flags are
// asserted by this fixture, not observed from any chain.
type fileSource struct {
	path  string
	clock *guardservices.SystemClock
}

func (s fileSource) Read(ctx context.Context, did string) (r.Snapshot, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil || ctx.Err() != nil {
		return r.Snapshot{}, errors.New("unavailable")
	}
	var keys map[string]string
	if json.Unmarshal(raw, &keys) != nil || keys[did] == "" {
		return r.Snapshot{}, errors.New("unknown")
	}
	list := []r.Key{{Name: "signing-1", Alg: "ed25519", Material: keys[did], State: "accepted"}}
	if did == bob {
		list = append([]r.Key{{Name: "kem-1", Alg: "x25519", Material: keys["kem"], State: "accepted"}}, list...)
	}
	stamp, err := s.clock.Now()
	if err != nil {
		return r.Snapshot{}, err
	}
	return r.Snapshot{Source: source, Registry: registry, Network: "local", DID: did, Version: "1", State: "active", Digest: hash(raw), Ready: true, Validated: true, Finalized: true, AcquiredMS: stamp.MonoMS, Keys: list}, nil
}

// syntheticMeasurement accepts any snapshot; it measures nothing.
type syntheticMeasurement struct{}

func (syntheticMeasurement) Check(ctx context.Context, s *b.Snapshot) error {
	if ctx.Err() != nil || s == nil {
		return errors.New("unavailable")
	}
	return nil
}

type config struct {
	State, Artifacts, Shared string
	Signer                   string
	SignerUID                uint32
	Approver                 string
	Address                  string
	Create                   bool
	Wait                     string
}

func load(args []string) (config, error) {
	fs := flag.NewFlagSet("host", flag.ContinueOnError)
	path := fs.String("config", "", "JSON configuration")
	if err := fs.Parse(args); err != nil || *path == "" {
		return config{}, errors.New("requires -config")
	}
	raw, err := os.ReadFile(*path)
	if err != nil {
		return config{}, err
	}
	var c config
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	return c, d.Decode(&c)
}

type parts struct {
	clock    *guardservices.SystemClock
	signer   *guardsigner.Client
	env      guardhost.Environment
	approved guardhost.Approved
	factory  *calc.Factory
}

func assemble(c config) (parts, error) {
	clock := guardservices.NewSystemClock()
	signer, err := guardsigner.NewClient(c.Signer, c.SignerUID, 2*time.Second)
	if err != nil {
		return parts{}, err
	}
	read := func(name string) ([]byte, error) { return os.ReadFile(filepath.Join(c.Shared, name)) }
	policy, err := read("policy.json")
	if err != nil {
		return parts{}, err
	}
	manifest, err := read("manifest.json")
	if err != nil {
		return parts{}, err
	}
	approval, err := read("approval.json")
	if err != nil {
		return parts{}, err
	}
	approver, err := hex.DecodeString(c.Approver)
	if err != nil || len(approver) != ed25519.PublicKeySize {
		return parts{}, errors.New("invalid approver")
	}
	factory, err := calc.NewFactory(calc.Config{Measurement: syntheticMeasurement{}, MaxInstances: 4})
	if err != nil {
		return parts{}, err
	}
	return parts{
		clock:  clock,
		signer: signer,
		env:    guardhost.Environment{Dir: c.State, Create: c.Create, Registry: r.Config{Source: source, Registry: registry, Network: "local"}, Source: fileSource{filepath.Join(c.Shared, "registry.json"), clock}, Clock: clock},
		approved: guardhost.Approved{Operation: b.Config{Directory: c.Artifacts, Policy: policy, Manifest: manifest, Limits: b.Limits{FileBytes: 4096, TotalBytes: 16384}, Factory: factory},
			Approval: approval, Approvers: []ed25519.PublicKey{approver}},
		factory: factory,
	}, nil
}

func bounds() g.MCPHostBounds {
	return g.MCPHostBounds{Capacity: 2, Preparations: 2, Clients: 2, Owners: 4, Workers: 1, Request: 20 * time.Second, Claim: 10 * time.Second, Worker: 5 * time.Second, Client: 20 * time.Second, Tick: time.Millisecond}
}

func receiver(args []string) error {
	c, err := load(args)
	if err != nil {
		return err
	}
	p, err := assemble(c)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	host, err := guardhost.OpenReceiver(ctx, guardhost.ReceiverConfig{Environment: p.env, Identity: guardhost.Identity{DID: bob, KeyID: bob + "#signing-1", Transport: p.signer, Result: p.signer}, KEM: p.signer.KEM(), Issuer: alice, IssuerKey: alice + "#signing-1", Approved: p.approved, Bounds: bounds(), Connection: g.MCPConnectionConfig{Role: g.MCPResponder, Name: "adk qualification receiver", Version: "1", TTLSeconds: 300, Timeout: 3 * time.Second}})
	if err != nil {
		return fmt.Errorf("open receiver: %w", err)
	}
	l, err := net.Listen("tcp", c.Address)
	if err != nil {
		return err
	}
	fmt.Println("receiver ready", c.Address, time.Now().UTC().Format(time.RFC3339Nano))
	served := host.Serve(ctx, l, 1)
	fmt.Println("receiver serve returned", time.Now().UTC().Format(time.RFC3339Nano), served, "signal", ctx.Err())
	closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = errors.Join(host.Close(closeCtx), p.factory.Close(closeCtx)); err != nil {
		return err
	}
	if served != nil && !errors.Is(served, context.Canceled) {
		return served
	}
	fmt.Println("receiver stopped")
	return nil
}

func caller(args []string) error {
	c, err := load(args)
	if err != nil {
		return err
	}
	p, err := assemble(c)
	if err != nil {
		return err
	}
	wait, err := time.ParseDuration(c.Wait)
	if err != nil {
		return err
	}
	ctx := context.Background()
	host, err := guardhost.OpenCaller(ctx, guardhost.CallerConfig{Environment: p.env, Identity: guardhost.Identity{DID: alice, KeyID: alice + "#signing-1", Transport: p.signer, Result: p.signer}, Intent: p.signer, Recipient: bob, RecipientKey: bob + "#signing-1", Approved: p.approved, Bounds: bounds(), Connection: g.MCPConnectionConfig{Role: g.MCPInitiator, Recipient: bob, RecipientKey: bob + "#signing-1", Name: "adk qualification caller", Version: "1", TTLSeconds: 300, Timeout: 3 * time.Second}})
	if err != nil {
		return fmt.Errorf("open caller: %w", err)
	}
	defer func() { _ = host.Close() }()
	fmt.Println("caller opened; waiting", wait, "for the replay quarantine", time.Now().UTC().Format(time.RFC3339Nano))
	time.Sleep(wait)
	run := func(arguments string) ([]byte, error) {
		conn, err := net.DialTimeout("tcp", c.Address, 3*time.Second)
		if err != nil {
			return nil, err
		}
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return host.Call(callCtx, [][]byte{[]byte("please add two and three")}, g.IntentProposal{Tool: "calculator", Arguments: []byte(arguments), LifetimeSeconds: 300}, conn)
	}
	if out, err := run(`{"a":2,"b":4,"operation":"add"}`); err == nil || out != nil {
		return errors.New("unapproved arguments were called")
	}
	fmt.Println("unapproved call refused")
	out, err := run(`{"a":2,"b":3,"operation":"add"}`)
	if err != nil {
		return fmt.Errorf("approved call: %w", err)
	}
	fmt.Println("verified output", string(out))
	if string(out) != `{"output":5,"success":true}` {
		return errors.New("unexpected output")
	}
	return p.factory.Close(ctx)
}
