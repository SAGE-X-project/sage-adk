// SPDX-License-Identifier: LGPL-3.0-or-later
// Package guardimage verifies and observes a narrowly supported sealed Linux
// host executable. It supplies no sandbox, signing authority or remote attestation.
package guardimage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"debug/elf"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"runtime"
	"sync/atomic"

	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

var (
	ErrDenied      = errors.New("approved runtime image unavailable or inconsistent")
	ErrUnsupported = errors.New("sealed host image unsupported on this platform")
)

const MaxImageBytes = 64 << 20

// Config is protected local administration, never model/plugin/peer input.
// SHA256 is independently approved; supplying a digest cannot approve itself.
// Only a static Linux Go executable of the native architecture is supported.
// No arguments or environment overrides are accepted. IO accepts only owned
// file handles; arbitrary Reader/Writer callbacks are not passed to exec.Cmd.
// The approved program must await native admission before any protected effect.
type Config struct {
	Image                 []byte
	SHA256                string
	Limit                 int64
	Stdin, Stdout, Stderr *os.File
}

// Observation describes a local executable backing object, not instruction-page
// attestation, complete host isolation, authorization or transferable proof.
type Observation struct {
	SHA256       string
	Architecture string
	PID          int
}

type backend interface {
	observe(context.Context) (Observation, error)
	close(context.Context) error
}

// Process retains the same verified sealed image and pidfd across observations.
// Keep it with protected supervision, outside plugin/model capabilities. This
// observes a child host; it is deliberately not guardcalculator.Measurement for
// a calculator running in the parent. A protected child-to-supervisor binding
// is still needed before using it as the child calculator's measurement service.
type Process struct {
	gate    chan struct{}
	retired atomic.Bool
	driver  backend
	digest  string
}

func active(ctx context.Context) bool { return ctx != nil && ctx.Err() == nil }
func acquire(ctx context.Context, gate chan struct{}) error {
	if !active(ctx) || gate == nil {
		return ErrDenied
	}
	select {
	case <-ctx.Done():
		return ErrDenied
	case <-gate:
	}
	if !active(ctx) {
		gate <- struct{}{}
		return ErrDenied
	}
	return nil
}

// Start verifies owned bytes before sealing and executing that same object.
// Context bounds startup; successful processes require explicit Close. No host
// is selected as a deployment subject merely by constructing this capability.
func Start(ctx context.Context, c Config) (*Process, error) {
	if !active(ctx) {
		return nil, ErrDenied
	}
	if runtime.GOOS != "linux" || (runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64") {
		return nil, ErrUnsupported
	}
	raw, err := approved(c)
	if err != nil {
		return nil, err
	}
	driver, err := start(ctx, c, raw)
	if err != nil {
		return nil, err
	}
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return &Process{gate: gate, driver: driver, digest: c.SHA256}, nil
}

func approved(c Config) ([]byte, error) {
	if c.Limit < 1 || c.Limit > MaxImageBytes || len(c.Image) == 0 || int64(len(c.Image)) > c.Limit || len(c.SHA256) != 64 {
		return nil, ErrDenied
	}
	raw := append([]byte(nil), c.Image...)
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != c.SHA256 {
		return nil, ErrDenied
	}
	if staticGo(raw, runtime.GOARCH) != nil {
		return nil, ErrDenied
	}
	return raw, nil
}

func staticGo(raw []byte, arch string) error {
	f, err := elf.NewFile(bytes.NewReader(raw))
	if err != nil {
		return ErrDenied
	}
	defer func() { _ = f.Close() }()
	machine := map[string]elf.Machine{"amd64": elf.EM_X86_64, "arm64": elf.EM_AARCH64}[arch]
	if machine == 0 || f.Machine != machine || f.Class != elf.ELFCLASS64 || f.Data != elf.ELFDATA2LSB || f.Type != elf.ET_EXEC {
		return ErrDenied
	}
	entry := false
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP || p.Type == elf.PT_DYNAMIC {
			return ErrDenied
		}
		if p.Type == elf.PT_GNU_STACK && p.Flags&elf.PF_X != 0 {
			return ErrDenied
		}
		if p.Filesz > uint64(len(raw)) || p.Off > uint64(len(raw))-p.Filesz {
			return ErrDenied
		}
		if p.Type != elf.PT_LOAD {
			continue
		}
		if p.Memsz < p.Filesz || p.Vaddr+p.Memsz < p.Vaddr || p.Flags&(elf.PF_W|elf.PF_X) == elf.PF_W|elf.PF_X {
			return ErrDenied
		}
		entry = entry || (p.Flags&elf.PF_X != 0 && f.Entry >= p.Vaddr && f.Entry-p.Vaddr < p.Filesz)
	}
	if !entry {
		return ErrDenied
	}
	info, err := buildinfo.Read(bytes.NewReader(raw))
	if err != nil {
		return ErrDenied
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		if _, exists := settings[s.Key]; exists {
			return ErrDenied
		}
		settings[s.Key] = s.Value
	}
	if settings["GOOS"] != "linux" || settings["GOARCH"] != arch || settings["CGO_ENABLED"] != "0" || settings["-buildmode"] != "exe" || settings["-compiler"] != "gc" {
		return ErrDenied
	}
	return nil
}

// Observe rechecks the kernel's live executable object and executable mappings.
// An observation failure permanently retires further appraisal; Close can retry
// resource cleanup. This does not inspect private instruction pages or establish
// the OS isolation that protects the supervisor and child from untrusted writers.
func (p *Process) Observe(ctx context.Context) (Observation, error) {
	if p == nil || p.retired.Load() || acquire(ctx, p.gate) != nil {
		return Observation{}, ErrDenied
	}
	defer func() { p.gate <- struct{}{} }()
	if p.retired.Load() || p.driver == nil {
		return Observation{}, ErrDenied
	}
	observed, err := p.driver.observe(ctx)
	if err != nil || p.retired.Load() || !active(ctx) || observed.SHA256 != p.digest || observed.PID <= 0 || observed.Architecture != runtime.GOARCH {
		p.retired.Store(true)
		return Observation{}, ErrDenied
	}
	return observed, nil
}

// CheckSnapshot requires the exact running image in both locally approved
// descriptors and verifies their owned artifact bytes before a fresh observation.
// This appraises the child executable only. It does not bind a parent calculator,
// approve a peer descriptor, implement parent-hop policy or grant admission.
func (p *Process) CheckSnapshot(ctx context.Context, snapshot *b.Snapshot, imagePath string) error {
	if !active(ctx) || p == nil || snapshot == nil || imagePath == "" {
		return ErrDenied
	}
	policy, manifest := snapshot.Policy(), snapshot.Manifest()
	if _, err := g.PolicyCommitment(policy); err != nil {
		return ErrDenied
	}
	if _, err := g.ManifestCommitment(manifest); err != nil {
		return ErrDenied
	}
	type file struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	type descriptor struct {
		Version string `json:"version"`
		Files   []file `json:"files"`
	}
	var m descriptor
	var pol struct {
		Artifacts descriptor `json:"artifacts"`
	}
	if json.Unmarshal(manifest, &m) != nil || json.Unmarshal(policy, &pol) != nil {
		return ErrDenied
	}
	artifacts := snapshot.Artifacts()
	verify := func(d descriptor, raw []byte) bool {
		image := false
		selected := []g.Artifact{}
		for _, f := range d.Files {
			image = image || (f.Path == imagePath && f.SHA256 == p.digest)
			for _, a := range artifacts {
				if a.Path == f.Path {
					selected = append(selected, a)
				}
			}
		}
		_, err := g.VerifyManifest(raw, selected)
		return image && err == nil
	}
	pd, err := json.Marshal(pol.Artifacts)
	if err != nil || !verify(m, manifest) || !verify(pol.Artifacts, pd) {
		return ErrDenied
	}
	_, err = p.Observe(ctx)
	return err
}

// Close permanently retires observations and terminates/reaps the owned child.
// Retire native hosts and finish accepted effects first. Emergency termination
// must retain uncertain completions in their journals; it never proves rollback.
// Cancellation retains resource ownership for a later cleanup retry.
func (p *Process) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.retired.Store(true)
	if acquire(ctx, p.gate) != nil {
		return ErrDenied
	}
	defer func() { p.gate <- struct{}{} }()
	if p.driver == nil {
		return nil
	}
	if err := p.driver.close(ctx); err != nil {
		return err
	}
	p.driver = nil
	return nil
}
