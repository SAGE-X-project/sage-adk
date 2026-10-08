// SPDX-License-Identifier: LGPL-3.0-or-later
// Package guardchannel connects sealed-child observations to that same child's
// compiled calculator measurement. It supplies neither isolation nor admission.
package guardchannel

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"sync/atomic"
	"time"

	b "github.com/sage-x-project/sage-adk/core/guardbinding"
	calc "github.com/sage-x-project/sage-adk/core/guardcalculator"
	image "github.com/sage-x-project/sage-adk/core/guardimage"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

const recordBytes = 112

// Config is protected independent administration. Both descriptors must cover
// ImagePath with Image.SHA256 and the supplied owned artifact bytes. All fields
// are mandatory except the image standard streams. MaxChecks bounds one session
// (1..1000000); Timeout bounds each observation/response (1ms..5s).
// MeasurementChannel must be nil: Start exclusively creates the private pair.
// No model, plugin or peer may choose this baseline or obtain channel handles.
type Config struct {
	Image            image.Config
	ImagePath        string
	Policy, Manifest []byte
	Artifacts        []g.Artifact
	MaxChecks        uint64
	Timeout          time.Duration
}

type record struct {
	kind             byte
	epoch            [32]byte
	sequence         uint64
	policy, manifest [32]byte
}
type endpoint interface {
	send(context.Context, record) error
	receive(context.Context) (record, error)
	close() error
	bindPeer(int) error
}

// Supervisor owns one sealed child, private endpoint and connection generation.
// Retire child native hosts/operations and drain accepted effects before Close.
// It owns cleanup after startup, even when observation permanently refuses.
type Supervisor struct {
	process      *image.Process
	endpoint     endpoint
	baseline     record
	timeout      time.Duration
	limit        uint64
	observations atomic.Uint64
	failure      atomic.Uint32
	cancel       context.CancelFunc
	done         chan struct{}
	retired      atomic.Bool
	gate         chan struct{}
	closed       bool
}

// Measurement is a child-only fixed connection capability. Check first requires
// Local's actual protected loaded-code/isolation assurance, then a fresh parent
// observation of this same child and exact approved commitments. Local cannot be
// nil, a digest echo or an appraisal of the parent. The channel alone does not
// fulfill that assurance. Keep this capability in native host configuration.
type Measurement struct {
	endpoint endpoint
	baseline record
	local    calc.Measurement
	timeout  time.Duration
	sequence uint64
	retired  atomic.Bool
	gate     chan struct{}
	closed   bool
}

var _ calc.Measurement = (*Measurement)(nil)

func active(ctx context.Context) bool   { return ctx != nil && ctx.Err() == nil }
func validTimeout(t time.Duration) bool { return t >= time.Millisecond && t <= 5*time.Second }
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
func digest(s string) (out [32]byte, err error) {
	raw, e := hex.DecodeString(s)
	if e != nil || len(raw) != 32 || hex.EncodeToString(raw) != s {
		return out, b.ErrDenied
	}
	copy(out[:], raw)
	return out, nil
}
func commitments(policy, manifest []byte) (p, m [32]byte, err error) {
	pd, e := g.PolicyCommitment(policy)
	if e != nil {
		return p, m, b.ErrDenied
	}
	md, e := g.ManifestCommitment(manifest)
	if e != nil {
		return p, m, b.ErrDenied
	}
	p, e = digest(pd)
	if e != nil {
		return p, m, b.ErrDenied
	}
	m, e = digest(md)
	return p, m, e
}
func approved(c Config) (record, error) {
	var r record
	if c.Image.MeasurementChannel != nil || c.ImagePath == "" || c.MaxChecks < 1 || c.MaxChecks > 1000000 || !validTimeout(c.Timeout) || len(c.Policy) > 1<<20 || len(c.Manifest) > 1<<20 || len(c.Artifacts) == 0 || len(c.Artifacts) > 4096 {
		return r, b.ErrDenied
	}
	p, m, e := commitments(c.Policy, c.Manifest)
	if e != nil {
		return r, e
	}
	imageSeen := false
	total := int64(0)
	seen := map[string]bool{}
	owned := make([]g.Artifact, 0, len(c.Artifacts))
	for _, a := range c.Artifacts {
		total += int64(len(a.Bytes))
		if seen[a.Path] || len(a.Bytes) > 64<<20 || total > 256<<20 {
			return r, b.ErrDenied
		}
		seen[a.Path] = true
		if a.Path == c.ImagePath {
			imageSeen = bytes.Equal(a.Bytes, c.Image.Image)
		}
		owned = append(owned, g.Artifact{Path: a.Path, Bytes: append([]byte(nil), a.Bytes...)})
	}
	if !imageSeen {
		return r, b.ErrDenied
	}
	type entry struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	type desc struct {
		Version string  `json:"version"`
		Files   []entry `json:"files"`
	}
	var policy struct {
		Artifacts desc `json:"artifacts"`
	}
	var manifest desc
	if json.Unmarshal(c.Policy, &policy) != nil || json.Unmarshal(c.Manifest, &manifest) != nil {
		return r, b.ErrDenied
	}
	declared := map[string]bool{}
	verify := func(d desc, raw []byte) bool {
		imageCovered := false
		selected := []g.Artifact{}
		for _, f := range d.Files {
			declared[f.Path] = true
			imageCovered = imageCovered || (f.Path == c.ImagePath && f.SHA256 == c.Image.SHA256)
			for _, a := range owned {
				if a.Path == f.Path {
					selected = append(selected, a)
				}
			}
		}
		_, err := g.VerifyManifest(raw, selected)
		return imageCovered && err == nil
	}
	pd, e := json.Marshal(policy.Artifacts)
	if e != nil || !verify(manifest, c.Manifest) || !verify(policy.Artifacts, pd) || len(declared) != len(owned) {
		return r, b.ErrDenied
	}
	r = record{kind: 'B', policy: p, manifest: m}
	if _, e = rand.Read(r.epoch[:]); e != nil || r.epoch == [32]byte{} {
		return record{}, b.ErrDenied
	}
	return r, nil
}

// Start approves owned baseline artifacts before launching the same sealed image
// with one exclusively owned child endpoint at fd 4. Linux only; no path/network
// listener, registry authority, signing interface or unsigned tool dispatcher.
func Start(ctx context.Context, c Config) (*Supervisor, error) {
	if !active(ctx) {
		return nil, b.ErrDenied
	}
	baseline, e := approved(c)
	if e != nil {
		return nil, e
	}
	startup, stop := context.WithTimeout(ctx, c.Timeout)
	process, ep, e := launch(startup, c.Image)
	stop()
	if e != nil {
		return nil, e
	}
	lifetime, cancel := context.WithCancel(context.Background())
	s := &Supervisor{process: process, endpoint: ep, baseline: baseline, timeout: c.Timeout, limit: c.MaxChecks, cancel: cancel, done: make(chan struct{}), gate: gate()}
	go s.serve(lifetime)
	return s, nil
}
func (s *Supervisor) serve(ctx context.Context) {
	defer func() { s.retired.Store(true); _ = s.endpoint.close(); close(s.done) }()
	bootstrap, stop := context.WithTimeout(ctx, s.timeout)
	observed, e := s.process.Observe(bootstrap)
	if e == nil {
		e = s.endpoint.bindPeer(observed.PID)
	}
	if e == nil {
		e = s.endpoint.send(bootstrap, s.baseline)
	}
	stop()
	if e != nil {
		s.failure.Store(1)
		return
	}
	for seq := uint64(1); seq <= s.limit; seq++ {
		request, e := s.endpoint.receive(ctx)
		expected := s.baseline
		expected.kind = 'Q'
		expected.sequence = seq
		if e != nil || request != expected || s.retired.Load() {
			s.failure.Store(2)
			return
		}
		check, stop := context.WithTimeout(ctx, s.timeout)
		_, e = s.process.Observe(check)
		if e != nil {
			s.failure.Store(3)
		}
		if e == nil && !s.retired.Load() {
			expected.kind = 'A'
			e = s.endpoint.send(check, expected)
			if e == nil {
				s.observations.Store(seq)
			} else {
				s.failure.Store(4)
			}
		}
		stop()
		if e != nil {
			return
		}
	}
}

// OpenChild takes ownership of fd 4 only in the supervised child. Local is the
// mandatory additional child-runtime assurance provider, not a permissive
// replacement for missing deployment isolation. Timeout bounds every Check;
// its callbacks must honor context, be non-reentrant and outlive Measurement.
// Errors close the supplied file. Successful initialization marks it CLOEXEC.
func OpenChild(ctx context.Context, file *os.File, local calc.Measurement, timeout time.Duration) (*Measurement, error) {
	if !active(ctx) || file == nil || absent(local) || !validTimeout(timeout) {
		if file != nil {
			_ = file.Close()
		}
		return nil, b.ErrDenied
	}
	ep, e := child(file)
	if e != nil {
		return nil, e
	}
	return establish(ctx, ep, local, timeout)
}
func establish(ctx context.Context, ep endpoint, local calc.Measurement, timeout time.Duration) (*Measurement, error) {
	bounded, stop := context.WithTimeout(ctx, timeout)
	defer stop()
	baseline, e := ep.receive(bounded)
	if e != nil || baseline.kind != 'B' || baseline.sequence != 0 || baseline.epoch == [32]byte{} || baseline.policy == [32]byte{} || baseline.manifest == [32]byte{} {
		_ = ep.close()
		return nil, b.ErrDenied
	}
	return &Measurement{endpoint: ep, baseline: baseline, local: local, timeout: timeout, gate: gate()}, nil
}

// Check serializes exact-snapshot checks and a fresh single-use observation.
// Sequence/generation, peer credentials, truncation, timeout or provider failure
// permanently retire the connection; no stale acknowledgement or fallback.
func (m *Measurement) Check(ctx context.Context, snapshot *b.Snapshot) (err error) {
	if m == nil || !active(ctx) || !validTimeout(m.timeout) || m.retired.Load() {
		return b.ErrDenied
	}
	bounded, stop := context.WithTimeout(ctx, m.timeout)
	defer stop()
	if enter(bounded, m.gate) != nil {
		return b.ErrDenied
	}
	defer func() { m.gate <- struct{}{} }()
	defer func() {
		if recover() != nil {
			m.retired.Store(true)
			err = b.ErrDenied
		}
	}()
	fail := func() error { m.retired.Store(true); return b.ErrDenied }
	if m.retired.Load() || m.closed || snapshot == nil || absent(m.local) || m.sequence >= 1000000 {
		return fail()
	}
	p, md, e := commitments(snapshot.Policy(), snapshot.Manifest())
	if e != nil || p != m.baseline.policy || md != m.baseline.manifest {
		return fail()
	}
	if m.local.Check(bounded, snapshot) != nil || !active(bounded) || m.retired.Load() {
		return fail()
	}
	m.sequence++
	request := m.baseline
	request.kind = 'Q'
	request.sequence = m.sequence
	if m.endpoint.send(bounded, request) != nil {
		return fail()
	}
	ack, e := m.endpoint.receive(bounded)
	request.kind = 'A'
	if e != nil || ack != request || !active(bounded) || m.retired.Load() {
		return fail()
	}
	return nil
}

// Close retires further checks and waits for accepted local providers, then
// closes the owned endpoint. Cancelled cleanup retains ownership for retry.
func (m *Measurement) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.retired.Store(true)
	if enter(ctx, m.gate) != nil {
		return b.ErrDenied
	}
	defer func() { m.gate <- struct{}{} }()
	if m.closed {
		return nil
	}
	if m.endpoint.close() != nil {
		return b.ErrDenied
	}
	m.closed = true
	return nil
}

// Close retires the private session before closing its socket and owned child.
// Drain accepted native effects first; termination is not rollback evidence.
func (s *Supervisor) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.retired.Store(true)
	if s.cancel != nil {
		s.cancel()
	}
	if enter(ctx, s.gate) != nil {
		return b.ErrDenied
	}
	defer func() { s.gate <- struct{}{} }()
	if s.closed {
		return nil
	}
	select {
	case <-s.done:
	case <-ctx.Done():
		return b.ErrDenied
	}
	if s.endpoint.close() != nil {
		return b.ErrDenied
	}
	if s.process.Close(ctx) != nil {
		return b.ErrDenied
	}
	s.closed = true
	return nil
}
func encode(r record) []byte {
	raw := make([]byte, recordBytes)
	raw[0] = r.kind
	copy(raw[1:8], "SGMC001")
	copy(raw[8:40], r.epoch[:])
	binary.BigEndian.PutUint64(raw[40:48], r.sequence)
	copy(raw[48:80], r.policy[:])
	copy(raw[80:], r.manifest[:])
	return raw
}
func decode(raw []byte) (record, error) {
	var r record
	if len(raw) != recordBytes || string(raw[1:8]) != "SGMC001" || (raw[0] != 'B' && raw[0] != 'Q' && raw[0] != 'A') {
		return r, b.ErrDenied
	}
	r.kind = raw[0]
	copy(r.epoch[:], raw[8:40])
	r.sequence = binary.BigEndian.Uint64(raw[40:48])
	copy(r.policy[:], raw[48:80])
	copy(r.manifest[:], raw[80:])
	return r, nil
}
