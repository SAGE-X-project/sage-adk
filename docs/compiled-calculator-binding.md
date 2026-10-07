# Protected compiled calculator binding

`core/guardcalculator` connects the existing ADK `CalculatorTool` to one fixed
approved-operation instance. It is a reusable **compiled-tool adapter**, not a
dynamic plugin loader, runtime-image attestation implementation or process sandbox.
The normative 0.10.0 protocol and transport are unchanged.

```go
factory, err := guardcalculator.NewFactory(guardcalculator.Config{
    Measurement: protectedRuntimeMeasurement,
    MaxInstances: 2,
})
if err != nil { return err }
operation, err := guardbinding.Open(ctx, capturedRequest, guardbinding.Config{
    Directory: protectedArtifactDirectory,
    Policy: approvedPolicyDescriptor,
    Manifest: approvedComponentManifest,
    Limits: guardbinding.Limits{FileBytes: 64 << 20, TotalBytes: 256 << 20},
    Factory: factory,
})
if err != nil { return err }
binding, err := operation.Binding(ctx)
if err != nil { return err }
// Supply operation to protected issuance/policy services and binding to toolhost.
// Retire/close native hosts first, then Operation, then Factory with bounded contexts.
```

The component manifest must include exact `calculator.json` bytes and the image
named by its independently approved configuration:

```json
{
  "version": "0.10.0",
  "tool": "calculator",
  "image_path": "images/protected-host",
  "operations": ["add", "divide", "multiply", "subtract"],
  "absolute_operand_limit": 1000000
}
```

This local configuration is closed: all five fields are required and additional
fields are denied. Operations are a nonempty, unique, sorted subset of the four
existing calculator operations. The finite positive operand bound is at most
one million. `image_path` must name a file in both the approved component manifest
and policy artifacts; it cannot name the rules or calculator configuration.
The policy still uses `sage-adk/exact-operation/1` and `rules.json`, including the
exact final arguments, recipient and key. An approved addition could use:

```json
{"a":2,"b":3,"operation":"add"}
```

Every call requires exactly those three argument members, number operands and
one configured operation. Unknown members, missing values, null/wrong types,
noncanonical JSON and out-of-bound operands are refused. No unsigned defaults
are added. Exact-operation policy independently requires the whole approved
argument object; a valid calculator schema does not approve different values.
Successful computation returns the existing ADK Result JSON, for example
`{"output":5,"success":true}`. A division-by-zero computation returns its existing
`{"error":"division by zero","success":false}` data object. Native `completed`
means the tool returned a durably recorded result, not that a domain-level
`success` field is true. Nonfinite/unserializable output remains refused.

## Actual runtime identity remains mandatory

`Measurement.Check(ctx, snapshot)` must establish that the running protected
host, exact policy evaluator, factory, compiled calculator and their dependencies
correspond to the approved image/artifacts. It must inspect the actual loaded
runtime identity using an appropriate protected deployment mechanism. The local
image file, source hashes, package names or supplied digests alone cannot establish
this. Measurement must also establish trusted verification of approved image and
dependency bytes **before loading**, and retention of that same immutable loaded
identity. A post-start hash of a mutable executable path supplies neither fact.
The protected launcher/image mechanism and its evidence remain provider work.
The image dependency set may need files beyond the executable; administration
and measurement must cover it in both relevant manifests.

The factory verifies descriptor commitments, exact component bytes and configuration
coverage, invokes measurement before constructing the tool, and retains one private
calculator instance. Measurement repeats before each actual arithmetic call under
its instance gate. It never reopens paths, executes snapshot files, resolves a Tool
Registry entry, installs a peer manifest or substitutes another tool instance.
The general guardbinding wrapper still checks current original capture and artifact
files. There is no unrestricted sign API, raw calculator getter or model-facing
factory service. Factory/Instance/Binding are trusted local capabilities; keep them
outside model, plugin and MCP custody. Ordinary ADK Tool Registry, Agent, A2A and
gRPC routes are not implicitly protected by constructing this adapter.

A failed runtime observation or provider panic permanently retires the instance.
Closure immediately refuses new factory loads and callbacks, waits for already
accepted bounded work, and releases retained instances. Cancellation of Close
preserves refusal and ownership; retry with a fresh bounded context. Providers
must honor cancellation and stay non-reentrant: the adapter cannot interrupt a
blocked provider or prove isolation against a hostile same-user process/kernel.
The core still owns current Registry authority, one-use approvals, durable fences,
terminal results and uncertain-completion reconciliation. Arithmetic error data
is not signing authority or permission to issue another call.

## Verification scope

Scenario units exercise all four real compiled calculator operations, arithmetic
error data, closed schema, approved-image coverage, unavailable measurement,
provider panic, changed configuration, bounded instance ownership, cancellation
and retirement while a controlled measurement callback is outstanding. Provider
failures are unit scenarios; no attack-capable runtime program is generated.

Both the main module and a separate importing consumer run the actual compiled
calculator through protected root issuance, native loopback MCP/HPKE, signed
result consumption, the same Client journal and completed-ledger recovery.
Original capture/files/journals and ephemeral keys are real local resources.
Registry observations and runtime-image appraisal remain explicit synthetic
providers. These tests demonstrate API/local integration, not a selected deployed
host, complete loaded-image proof, trusted blockchain finality or independent hop
execution. Existing Inspector source reports keep their historical pins; all
thirteen deployed host controls remain NOT_RUN and full conformance remains
NOT_ESTABLISHED.

Run the safe tests with:

```sh
go test -mod=readonly -race ./core/guardcalculator
go test -mod=readonly -race -timeout 1m ./core/toolhost -run TestCompiledCalculatorNativeRuntime
(cd verification/library-consumer && go test -mod=readonly -race -timeout 1m ./...)
```

Concrete protected runtime measurement, authoritative blockchain Source and
independent effect observation remain deployment work. Dynamic plugin/MCP loaders
need their own approved-byte/immutable-instance/isolation contracts; this adapter
supplies none implicitly and does not advance later demo or protocol upgrade work.

The later [sealed Linux image supervisor](sealed-host-image.md) supplies actual
pre-exec sealed-byte verification and kernel-backed child executable observations
for a restricted static Go profile. It is deliberately not a parent calculator's
Measurement implementation. A protected bridge to the same measured child's native
gate and actual deployment isolation remain required; the existing synthetic
calculator measurement tests are not promoted to deployed evidence.
