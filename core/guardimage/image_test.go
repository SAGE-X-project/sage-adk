// SPDX-License-Identifier: LGPL-3.0-or-later
package guardimage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sage-x-project/sage-adk/core/capture"
	b "github.com/sage-x-project/sage-adk/core/guardbinding"
)

var testImage []byte

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "sage-image-test-")
	if err != nil {
		panic(err)
	}
	path := filepath.Join(dir, "host")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	cmd := exec.CommandContext(ctx, "go", "build", "-mod=readonly", "-trimpath", "-o", path, "./testdata/host")
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
	if output, e := cmd.CombinedOutput(); e != nil {
		cancel()
		_ = os.RemoveAll(dir)
		panic(string(output) + e.Error())
	}
	cancel()
	testImage, err = os.ReadFile(path)
	if err != nil {
		_ = os.RemoveAll(dir)
		panic(err)
	}
	status := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(status)
}
func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func config() Config         { return Config{Image: testImage, SHA256: hash(testImage), Limit: MaxImageBytes} }

func TestApprovedStaticImageAndOwnedBytes(t *testing.T) {
	c := config()
	c.Image = append([]byte(nil), testImage...)
	raw, err := approved(c)
	if err != nil {
		t.Fatal(err)
	}
	c.Image[0] ^= 1
	if !bytes.Equal(raw, testImage) {
		t.Fatal("verified bytes shared with caller")
	}
	for _, mutate := range []func(*Config){
		func(c *Config) { c.Limit = 0 }, func(c *Config) { c.Limit = MaxImageBytes + 1 },
		func(c *Config) { c.Limit = int64(len(c.Image) - 1) }, func(c *Config) { c.Image = nil },
		func(c *Config) { c.SHA256 = "" }, func(c *Config) { c.SHA256 = hash([]byte("different")) },
	} {
		bad := config()
		mutate(&bad)
		if _, err := approved(bad); !errors.Is(err, ErrDenied) {
			t.Fatal("accepted invalid baseline/bounds", err)
		}
	}
}
func TestUnsupportedExecutableProfiles(t *testing.T) {
	f, err := elf.NewFile(bytes.NewReader(testImage))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	phoff := binary.LittleEndian.Uint64(testImage[32:40])
	stride := uint64(binary.LittleEndian.Uint16(testImage[54:56]))
	executable := -1
	for n, p := range f.Progs {
		if p.Type == elf.PT_LOAD && p.Flags&elf.PF_X != 0 {
			executable = n
			break
		}
	}
	if executable < 0 {
		t.Fatal("fixture has no executable segment")
	}
	for _, mode := range []string{"not-elf", "class", "arch", "pie", "entry", "interpreter", "dynamic", "writable-code", "segment-bounds", "missing-buildinfo"} {
		t.Run(mode, func(t *testing.T) {
			raw := append([]byte(nil), testImage...)
			off := int(phoff + uint64(executable)*stride)
			switch mode {
			case "not-elf":
				raw[0] = 0
			case "class":
				raw[4] = 1
			case "arch":
				binary.LittleEndian.PutUint16(raw[18:20], uint16(elf.EM_386))
			case "pie":
				binary.LittleEndian.PutUint16(raw[16:18], uint16(elf.ET_DYN))
			case "entry":
				binary.LittleEndian.PutUint64(raw[24:32], 0)
			case "interpreter":
				binary.LittleEndian.PutUint32(raw[off:off+4], uint32(elf.PT_INTERP))
			case "dynamic":
				binary.LittleEndian.PutUint32(raw[off:off+4], uint32(elf.PT_DYNAMIC))
			case "writable-code":
				binary.LittleEndian.PutUint32(raw[off+4:off+8], uint32(elf.PF_R|elf.PF_W|elf.PF_X))
			case "segment-bounds":
				binary.LittleEndian.PutUint64(raw[off+8:off+16], uint64(len(raw)+1))
			case "missing-buildinfo":
				s := f.Section(".go.buildinfo")
				if s == nil {
					t.Fatal("fixture lacks build info")
				}
				clear(raw[s.Offset : s.Offset+s.Size])
			}
			// Altered bytes are parsed only. No refusal fixture is ever executed.
			c := config()
			c.Image = raw
			c.SHA256 = hash(raw)
			if _, e := approved(c); !errors.Is(e, ErrDenied) {
				t.Fatal("accepted unsupported executable", e)
			}
		})
	}
	if staticGo(testImage, "386") == nil {
		t.Fatal("accepted unsupported architecture")
	}
}

type unitBackend struct {
	observation Observation
	err         error
	closeErr    error
	calls       atomic.Int64
	callback    func()
}

func (d *unitBackend) observe(context.Context) (Observation, error) {
	d.calls.Add(1)
	if d.callback != nil {
		d.callback()
	}
	return d.observation, d.err
}
func (d *unitBackend) close(context.Context) error { return d.closeErr }
func unitProcess(d *unitBackend) *Process {
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return &Process{gate: gate, driver: d, digest: hash(testImage)}
}
func unitDriver() *unitBackend {
	return &unitBackend{observation: Observation{SHA256: hash(testImage), Architecture: runtime.GOARCH, PID: 100}}
}
func TestObservationRefusalRetirementAndCleanup(t *testing.T) {
	for _, mode := range []string{"unavailable", "digest", "pid", "arch", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			d := unitDriver()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "unavailable":
				d.err = ErrDenied
			case "digest":
				d.observation.SHA256 = hash(nil)
			case "pid":
				d.observation.PID = 0
			case "arch":
				d.observation.Architecture = "unsupported"
			case "cancelled":
				d.callback = cancel
			}
			p := unitProcess(d)
			if _, e := p.Observe(ctx); !errors.Is(e, ErrDenied) {
				t.Fatal("accepted invalid observation")
			}
			d.err = nil
			d.observation = unitDriver().observation
			d.callback = nil
			if _, e := p.Observe(context.Background()); !errors.Is(e, ErrDenied) || d.calls.Load() != 1 {
				t.Fatal("re-enrolled retired observer")
			}
			d.closeErr = ErrDenied
			if p.Close(context.Background()) == nil || p.driver == nil {
				t.Fatal("lost pending cleanup ownership")
			}
			d.closeErr = nil
			if p.Close(context.Background()) != nil || p.driver != nil {
				t.Fatal("cleanup retry failed")
			}
		})
	}
	d := unitDriver()
	p := unitProcess(d)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p.Close(ctx) == nil || !p.retired.Load() || p.driver == nil {
		t.Fatal("cancelled close released ownership")
	}
	for range 2 {
		if p.Close(context.Background()) != nil {
			t.Fatal("close is not retryable/idempotent")
		}
	}
	var absent *Process
	if absent.Close(context.Background()) != nil {
		t.Fatal("nil close")
	}
	for _, q := range []*Process{nil, {}, unitProcess(unitDriver())} {
		if _, e := q.Observe(ctx); e == nil {
			t.Fatal("accepted cancelled/missing observer")
		}
	}
}
func TestCloseWinsOutstandingObservation(t *testing.T) {
	d := unitDriver()
	entered, release := make(chan struct{}), make(chan struct{})
	d.callback = func() { close(entered); <-release }
	p := unitProcess(d)
	finished := make(chan error, 1)
	go func() { _, e := p.Observe(context.Background()); finished <- e }()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p.Close(ctx) == nil {
		t.Fatal("cancelled close")
	}
	close(release)
	if err := <-finished; !errors.Is(err, ErrDenied) {
		t.Fatal("observation escaped retirement", err)
	}
	if p.Close(context.Background()) != nil {
		t.Fatal("cleanup")
	}
}

// The fake instance is only a way to obtain guardbinding's owned Snapshot for
// descriptor refusal units; it performs no tool and is not runtime attestation.
type snapshotFactory struct{ snapshot *b.Snapshot }

func (f *snapshotFactory) Load(_ context.Context, s *b.Snapshot) (b.Instance, error) {
	f.snapshot = s
	return snapshotInstance{}, nil
}

type snapshotInstance struct{}

func (snapshotInstance) Check(context.Context, string, string, string) error { return nil }
func (snapshotInstance) Execute(context.Context, []byte) ([]byte, error)     { return nil, ErrDenied }
func snapshot(t *testing.T, policyImage, componentImage bool, image []byte) *b.Snapshot {
	t.Helper()
	dir := t.TempDir()
	store, e := capture.OpenFileStore(filepath.Join(dir, "capture"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = store.Close() })
	host, e := capture.NewHost(store)
	if e != nil {
		t.Fatal(e)
	}
	request, e := host.Capture(context.Background(), [][]byte{[]byte("approved arithmetic")})
	if e != nil {
		t.Fatal(e)
	}
	encode := func(v any) []byte {
		r, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	artifacts := map[string][]byte{"host": image, "rules.json": encode(map[string]any{"version": "0.10.0", "recipient": "did:sage:web:agent.example:bob", "keyid": "did:sage:web:agent.example:alice#signing-1", "tool": "calculator", "arguments": map[string]any{"a": 2, "b": 3, "operation": "add"}, "max_lifetime": 300})}
	for n, r := range artifacts {
		if e := os.WriteFile(filepath.Join(dir, n), r, 0600); e != nil {
			t.Fatal(e)
		}
	}
	descriptor := func(names ...string) []byte {
		sort.Strings(names)
		files := []any{}
		for _, n := range names {
			files = append(files, map[string]any{"path": n, "sha256": hash(artifacts[n])})
		}
		return encode(map[string]any{"version": "0.10.0", "files": files})
	}
	pn, mn := []string{"rules.json"}, []string{"rules.json"}
	if policyImage {
		pn = append(pn, "host")
	}
	if componentImage {
		mn = append(mn, "host")
	}
	policy := encode(map[string]any{"version": "0.10.0", "issuer": "did:sage:web:agent.example:alice", "epoch": "00000000-0000-4000-8000-000000000001", "engine": b.Engine, "artifacts": json.RawMessage(descriptor(pn...))})
	factory := &snapshotFactory{}
	op, e := b.Open(context.Background(), request, b.Config{Directory: dir, Policy: policy, Manifest: descriptor(mn...), Limits: b.Limits{FileBytes: MaxImageBytes, TotalBytes: 256 << 20}, Factory: factory})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = op.Close(context.Background()) })
	return factory.snapshot
}
func TestSnapshotRequiresBothApprovedImageDescriptors(t *testing.T) {
	d := unitDriver()
	p := unitProcess(d)
	ctx := context.Background()
	if p.CheckSnapshot(ctx, snapshot(t, true, true, testImage), "host") != nil {
		t.Fatal("matching snapshot denied")
	}
	for _, r := range []struct {
		policy, component bool
		image             []byte
		path              string
	}{{false, true, testImage, "host"}, {true, false, testImage, "host"}, {true, true, []byte("different"), "host"}, {true, true, testImage, "missing"}} {
		if p.CheckSnapshot(ctx, snapshot(t, r.policy, r.component, r.image), r.path) == nil {
			t.Fatal("accepted uncovered/mismatched image")
		}
	}
	if d.calls.Load() != 1 {
		t.Fatal("descriptor rejection reached observer")
	}
	for _, s := range []*b.Snapshot{nil, {}} {
		if p.CheckSnapshot(ctx, s, "host") == nil {
			t.Fatal("accepted missing snapshot")
		}
	}
	if p.CheckSnapshot(ctx, snapshot(t, true, true, testImage), "") == nil {
		t.Fatal("accepted missing image path")
	}
}
func TestMapsRefuseUncoveredExecutableObjects(t *testing.T) {
	valid := []byte("400000-410000 r-xp 00000000 00:01 42 /memfd:sage-approved-host (deleted)\n700000-710000 r-xp 00000000 00:00 0 [vdso]\n")
	if !validMaps(valid, 0, 1, 42) {
		t.Fatal("covered executable mapping denied")
	}
	for _, r := range []string{"", "invalid\n", "400000-410000 rwxp 00000000 00:01 42 /host\n", "400000-410000 r-xp 00000000 00:01 43 /other\n", "400000-410000 r-xp 00000000 00:02 42 /other\n", "400000-410000 r-xp 00000000 00:00 0\n", "400000-410000 r-xp 00000000 00:00 0 [unknown]\n", "400000-410000 r-xp 00000000 xx:01 42 /host\n"} {
		if validMaps([]byte(r), 0, 1, 42) {
			t.Fatal("accepted uncovered/malformed mapping", r)
		}
	}
}
func TestUnsupportedPlatformAndStartupContext(t *testing.T) {
	for _, ctx := range []context.Context{nil, func() context.Context { c, cancel := context.WithCancel(context.Background()); cancel(); return c }()} {
		if _, e := Start(ctx, config()); e == nil {
			t.Fatal("accepted cancelled startup")
		}
	}
	if runtime.GOOS != "linux" {
		if _, e := Start(context.Background(), config()); !errors.Is(e, ErrUnsupported) {
			t.Fatal("unsupported platform silently downgraded", e)
		}
	}
}
