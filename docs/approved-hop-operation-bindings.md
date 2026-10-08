# Approved downstream operation and loader binding

`core/guardbinding.OpenHop` connects an actual retained admitted-hop original
from `core/capture` to independent exact-operation policy, approved artifact
bytes and one immutable loaded instance. `Open` remains the explicit root entry
point and continues to refuse a parent ID. No SAGE 0.10.0 wire, RFC carriage,
policy descriptor, component manifest or local rules schema changes.

The protected native coordinator receives the actual `guard010.Invocation`
inside its admitted worker, captures it with `Host.CaptureHop`, then uses:

```go
operation, err := guardbinding.OpenHop(ctx, capturedHop, guardbinding.Config{
    Directory: protectedArtifactDirectory,
    Policy: independentlyApprovedLocalPolicy,
    Manifest: independentlyApprovedComponentManifest,
    Limits: guardbinding.Limits{FileBytes: 64 << 20, TotalBytes: 256 << 20},
    Factory: protectedLoader,
})
if err != nil { return err }
// issuerServices contains this host's independent authority, clock, protected
// signer, recipient and key configuration; no upstream permission is inherited.
issuerServices.Client.Policy = operation
issuerServices.Policy = operation
issuerServices.Measurement = operation
issuer, err := capturedHop.NewIntentIssuer(ctx, issuerServices)
if err != nil { return err }
// Authorize and Issue through the native issuer, successfully close its Client,
// then use capturedHop.OpenMCPClient on that exact existing protected journal.
```

The policy issuer must equal the authenticated inbound recipient verified by
`HopRequest.Inputs`. Local rules independently fix the next recipient, own
signing key, tool, exact canonical arguments and lifetime bound. Upstream
arguments or permission do not install the downstream rules. The original digest
commits to exact authenticated inbound bytes; semantic application approval still
belongs to independent local policy.

`ApproveIntent` checks the closed seventeen-field intent and requires the exact
retained parent call ID. Null, another ID, wrong type or a child call ID equal to
the parent is refused. Root approval still requires null. The helper cannot
mint or reconstruct native parent admission: the public constructor accepts only
`*capture.HopRequest`, and a nil, zero, completed or permanently retired capture
fails before loading. Recreating stored history cannot restore a finished parent.

The private retained-input interface lets the same artifact/policy/instance
checks support both root and hop capture without exposing a raw root Request or
accepting arbitrary external original providers. Original checks run before
reading approval and before loading, and the existing final checks surround
loaded-instance appraisal. Each later binding, approval, measurement and effect
rechecks the retained capture. Observed parent/upstream or storage inconsistency
permanently retires the capture and operation; re-enabling policy does not revive
it. Cancellation before gate entry refuses without claiming rollback.

`Factory.Load` still receives only defensive Snapshot copies, never native
Invocation, admission, capture, signer, key custody or host authority. The
[compiled calculator](compiled-calculator-binding.md) factory works with both
constructors, requiring approved configuration and mandatory measurement of the
same protected running evaluator/tool/dependencies. It is not a dynamic loader
or deployment attestation. An ordinary `LoadedTool` still receives arguments
only; coordinator admission and signing custody must not be moved into a plugin.

`Operation.Binding` supplies the same private loaded-tool wrapper for trusted
native configuration, not a model-facing unsigned dispatcher. For a remote
recipient, separately approved receiver policy, loader and actual effects remain
required; the local issuing operation does not attest a remote host. All claimed
final effects need their own binding. Administration still protects baseline
provenance, artifact storage, actual runtime/isolation measurement and custody,
and owns durable epochs and cross-host retirement. This helper supplies no
atomic transaction across providers, universal sandbox or automatic deployment
configuration.

Retire native workers and drain accepted effects before closing Operation and
Factory. Close refuses new callbacks immediately, serializes with the accepted
bounded effect and retains cleanup ownership on timeout. A coordinator may close
its per-call operation after that call's downstream connection/effect has ended.
Unknown completion remains fenced; cleanup does not prove rollback or authorize
a fresh call ID. RestoreHop requires the same actually live parent separately.

## Verification

Scenario units test nil/zero public hop capabilities, exact parent matching,
root/hop separation, local issuer mismatch and each independently fixed identity,
key, commitment, tool, arguments and profile field. Existing artifact, loader,
panic, cancellation and retirement units apply to the shared implementation.
Synthetic local-policy units do not supply a native admission.

Safe native A-to-B-to-A loopback tests use actual encrypted sessions, ephemeral
registered signing/KEM keys, durable original/client/ledger records and the real
compiled calculator. They check allowed completion, independent policy refusal,
unavailable measurement, retired operation and upstream policy loss with no
revival. Every refusal requires zero child signing and zero attempted protected
receiver callbacks; the allowed case signs once, invokes the same bound callback
once and returns verified arithmetic output. Finished parent capture cannot open
another operation or reach its loader. A foreign local issuer is refused before
measurement/loading while the real parent is live.

The approved-hop fixture samples one real elapsed monotonic origin shared by both
native legs; concurrent polling does not double-advance injected time. Workers
are bounded to thirty seconds and request/client/test lifetime to one minute to
cover durable checks on race-enabled runners. Setup/frame waits use a finite
ten-second budget because the core applies connection Timeout to frame IO too.
Nonsecret listener diagnostics record timeouts without reading frame contents.
No production timer, freshness
rule, validation or successful-completion expectation is changed. Historical
capture-only scenarios retain their original injected-clock configuration.

Both downstream issuance and receiver bindings are co-located in this bounded
fixture, and Registry/policy baseline provenance and loaded-runtime measurement
remain synthetic. These tests are library integration evidence, not independent
cross-core hop, selected-host isolation, blockchain finality or deployment
conformance. No attack-capable reproduction or host-bypass program is added.

```sh
go test -mod=readonly -race ./core/guardbinding
go test -mod=readonly -race -timeout 2m ./core/toolhost \
  -run 'TestApprovedHopOperationNativeRuntime|TestAdmittedHopCaptureNativeRuntime'
(cd verification/library-consumer && go test -mod=readonly -race -timeout 1m ./...)
```

The next step is a separate Inspector snapshot for this exact merged source
revision, preserving the six historical catalogs/reports. Then continue actual
authoritative blockchain Source and selected protected host/providers, followed
by independent effect/deployment observation in the approved order. All thirteen
deployed controls and independent hop execution remain `NOT_RUN`, and full
conformance remains `NOT_ESTABLISHED`. Later contract upgrades, demo and A2A/DID
work keep their existing order.
