# Approved operation policy and artifact binding

Use `core/guardbinding` to connect a protected captured Request, independently
approved policy/component descriptors and one fixed loaded instance to the
native tool host. This supplies a concrete exact-operation evaluator and file
snapshot checks; it does not supply OS isolation or deployment attestation.

```go
operation, err := guardbinding.Open(ctx, capturedRequest, guardbinding.Config{
    Directory: protectedArtifactDirectory,
    Policy: approvedPolicyDescriptor,
    Manifest: approvedComponentManifest,
    Limits: guardbinding.Limits{FileBytes: 64 << 20, TotalBytes: 256 << 20},
    Factory: protectedLoader,
})
if err != nil { return err }
binding, err := operation.Binding(ctx)
if err != nil { return err }
// Use operation as both core IssuancePolicy/IntentPolicy and IntentMeasurement.
// Use binding only in toolhost.Open's protected native configuration.
// ExpectedRecipient and KeyID must match operation.Recipient() and KeyID().
// After retiring/closing native workers, close operation with bounded context.
// Timeout retains refusal and ownership; retry Close with a fresh context.
```

The policy descriptor remains the existing EXEC-02 version/issuer/epoch/engine/
artifacts object. The component manifest remains the existing EXEC-06 version/
files object. The supported local engine is `sage-adk/exact-operation/1` and
its policy artifact `rules.json` has exactly these fields:

```json
{
  "version": "0.10.0",
  "recipient": "did:sage:web:agent.example:bob",
  "keyid": "did:sage:web:agent.example:alice#signing-1",
  "tool": "sum",
  "arguments": {"a": 2, "b": 3},
  "max_lifetime": 300
}
```

These local rules are independently approved administrator configuration, not
model output, wire permissions or a new protocol field. Both manifests must
cover their actual evaluator/tool, rules/configuration and loadable dependencies.
The helper requires a nonempty component baseline; empty built-in baselines need
another explicitly reviewed binding. Effective policy changes require a new
approved epoch. This volatile helper does not persist epochs or authorize reuse
of an old epoch after a change/recovery. Administrators remain responsible for
mapping installation and retirement across receivers and restarts.

The evaluator permits only the fixed recipient, signing key, tool, canonical
argument object and lifetime bound. `ApproveIntent` checks every field in the
closed canonical root intent, including the exact captured request, original,
policy and manifest commitments. Core issuance still checks current authority/
time and owns one-use approval, permanent issuance fencing and the Client journal.
This binding supports root operations; it refuses a carried parent ID. Admitted
hop policy must be bound separately. The rules do not infer semantic equivalence
from natural-language input; the original digest is an audit commitment. A
trusted application must select an appropriate approved policy for the request.
Human confirmation is optional under that policy; skipping confirmation does
not skip verification.

The artifact reader uses a confined directory handle, rejects missing files,
symlinks, non-regular files, inconsistent shared paths and byte-limit/hash drift,
and takes an owned exact snapshot. Final file opens on Linux/macOS refuse
symlinks and cannot block on a replaced FIFO before regular-file inspection.
Other operating systems refuse artifact opening. Administration must protect
and serialize directory/file mutation and use bounded local storage. The API
is not isolation against a concurrently hostile kernel or same-user process.
Extra files outside the declared artifact set are not loaded by the helper;
Factory must not load undeclared dependencies or resolve mutable paths again.

`Factory.Load` receives only defensive artifact/descriptor copies, never capture,
policy authority, signers, reservations or host handles. It returns one fixed
`Instance`, whose mandatory `Check(policyDigest, manifestDigest, tool)` must verify
actual loaded evaluator/tool/dependency identity. Echoing hashes or hashing source
files cannot prove which code executes. Factory retains administrative resource
ownership, cleans partial loading, and remains alive until native host workers and
Operation close successfully; retire/close it only afterwards. No universal
plugin/interpreter loader or hardware attestation is fabricated by this adapter.

The private native Binding wrapper rechecks original input, approved files and
the same loaded instance immediately before passing a copy of the exact arguments.
Execution and retirement share a context-aware gate. Close immediately refuses
new callbacks, waits for the already accepted bounded effect/cleanup and then
closes artifact custody. Changed capture/files or failed instance measurement
permanently refuse that operation. Do not silently restore it or issue a new call
ID after uncertain completion. Core durable state owns reconciliation.

Operation exports no raw instance getter or unsigned Execute method. Binding,
Factory and Instance remain trusted host capabilities; do not expose or call
Binding.Tool as a model-facing unsigned dispatcher. Ordinary Tool Registry,
Agent, A2A, gRPC and lifecycle hooks remain separate unmediated routes. All claimed
final effects need explicit binding; constructing an operation does not wrap them.

Units cover configuration, source drift, closed full-intent checks, snapshot
copies, provider panic/failure, cancellation and retirement with in-flight inert
execution. The safe native root/external consumer runtime uses actual private
artifact files, temporary journals, ephemeral Ed25519/X25519 keys and one inert
arithmetic effect, then verifies signed output and recovery. Loaded-instance and
Registry providers are explicit fixtures. It is packaging/local integration
evidence, not a selected host, blockchain observation, independent hop execution
or full protocol conformance. Inspector's historical ADK source inventory keeps
its older pin and is not silently replaced by these tests.

Validation also runs build, vet, dependency verification, full safe race tests
and the external import consumer. With golangci-lint 2.13.2, the new package
has zero findings. An uncapped comparison against clean ADK
`afa469cdd8539992185235008ac1591c74012f7f` found 303 existing full-project
lint diagnostics; this change adds none and fixes the two touched runtime
cleanup diagnostics, leaving 301. Legacy lint cleanup remains a separate
refactoring obligation; full-project lint is not reported as passing. The
existing Linux/macOS CI gates build, vet, full race tests and the external
consumer. CI also enforces new-package lint and at least 90 percent coverage;
it does not currently gate the legacy full-project lint backlog.

The [compiled calculator binding](compiled-calculator-binding.md) provides a
concrete factory for the existing static ADK calculator, with approved closed
configuration, image coverage, mandatory protected runtime measurement and one
private instance. Its local signed MCP/consumer tests execute real arithmetic;
image appraisal and Registry providers remain synthetic. It is not a universal
loader or deployed-host attestation and does not change historical Inspector verdicts.
