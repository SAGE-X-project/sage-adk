// SPDX-License-Identifier: LGPL-3.0-or-later
package guardbinding

import (
	"context"
	"os"
	"sync/atomic"

	adkerrors "github.com/sage-x-project/sage-adk/pkg/errors"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

// Engine identifies the built-in exact-operation rules evaluator/version.
const Engine = "sage-adk/exact-operation/1"

// ErrDenied refuses missing, inconsistent or retired protected bindings.
var ErrDenied = &adkerrors.Error{Category: adkerrors.CategorySecurity, Code: "APPROVED_OPERATION_DENIED", Message: "approved operation unavailable or inconsistent"}

// Limits bounds regular local artifact reads. Both bounds are mandatory. File
// bytes are at most 64 MiB and the unique combined snapshot at most 256 MiB.
type Limits struct{ FileBytes, TotalBytes int64 }

// Config is independent protected administration, never peer/model input.
// Policy artifacts must include rules.json and the actual evaluator/configuration
// and dependencies. Manifest covers the actual tool and its dependencies. Factory
// and local storage must be bounded, trusted, immutable and non-reentrant.
type Config struct {
	Directory        string
	Policy, Manifest []byte
	Limits           Limits
	Factory          Factory
}

// Factory loads exactly one immutable instance from owned approved artifact
// copies. It must not resolve mutable paths again or substitute another instance.
// Factory retains administrative resource ownership, cleans partial loading,
// and must remain alive until host and Operation close successfully.
// Instance.Check must establish actual evaluator/tool/dependency load identity,
// not merely echo digests. No private authority is passed to the loader.
type Factory interface {
	Load(context.Context, *Snapshot) (Instance, error)
}

// Instance owns the same verified evaluator and tool instance throughout use.
// Check validates both approved commitments and the tool binding. Execute accepts
// exact canonical arguments and finishes its effect/cleanup before returning.
// Both calls must be bounded, cancellation-aware, concurrent-safe and non-reentrant.
type Instance interface {
	Check(context.Context, string, string, string) error
	Execute(context.Context, []byte) ([]byte, error)
}

// Snapshot contains exact owned artifact bytes and canonical approved descriptors.
// Accessors return copies. It is not an authorization, execution or signing token.
type Snapshot struct {
	artifacts        []g.Artifact
	policy, manifest []byte
}

// Artifacts returns copies of the sorted unique policy/component artifact union.
func (s *Snapshot) Artifacts() []g.Artifact {
	if s == nil {
		return nil
	}
	a := make([]g.Artifact, len(s.artifacts))
	for i, f := range s.artifacts {
		a[i] = g.Artifact{Path: f.Path, Bytes: append([]byte(nil), f.Bytes...)}
	}
	return a
}

// Policy returns the canonical approved policy descriptor.
func (s *Snapshot) Policy() []byte {
	if s == nil {
		return nil
	}
	return append([]byte(nil), s.policy...)
}

// Manifest returns the canonical approved component manifest.
func (s *Snapshot) Manifest() []byte {
	if s == nil {
		return nil
	}
	return append([]byte(nil), s.manifest...)
}

type file struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type descriptor struct {
	Version string `json:"version"`
	Files   []file `json:"files"`
}
type rules struct {
	Version   string `json:"version"`
	Recipient string `json:"recipient"`
	KeyID     string `json:"keyid"`
	Tool      string `json:"tool"`
	Arguments []byte
	Lifetime  int64
}

// retainedInput is private: only the explicit root/hop constructors accept
// capture capabilities. External loaders cannot supply a replacement original.
type retainedInput interface {
	ID() string
	Digest() string
	Inputs(context.Context) ([][]byte, error)
}

// Operation binds one protected root or admitted-hop capture, or a separate
// receiver's provisioned mapping (OpenReceiver), to a fixed operation.
// Zero values are invalid. It exports no raw instance or Execute method. Close
// permanently retires admission and serializes with the same final effect gate.
// Administration still owns durable epochs and retirement across hosts/restarts.
type Operation struct {
	gate                                 chan struct{}
	retiring                             atomic.Bool
	closed                               bool
	root                                 *os.Root
	request                              retainedInput
	receiver                             bool
	parentID                             string
	limits                               Limits
	policy, manifest                     []byte
	policyDigest, manifestDigest, issuer string
	entries                              []file
	rules                                rules
	instance                             Instance
}

// Recipient returns the fixed receiver from approved rules, or empty for nil.
func (o *Operation) Recipient() string {
	if o == nil {
		return ""
	}
	return o.rules.Recipient
}

// KeyID returns the fixed intent signing key from approved rules, or empty for nil.
func (o *Operation) KeyID() string {
	if o == nil {
		return ""
	}
	return o.rules.KeyID
}
