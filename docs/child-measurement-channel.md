# Supervised child measurement channel

`core/guardchannel` connects `core/guardimage` observations to
`core/guardcalculator.Measurement` in the same sealed Linux child. It requires
an additional protected child-local measurement provider. It supplies no
operating-system isolation, signing authority, registry selection or tool
admission of its own. No protocol, RFC carriage or SAGE 0.10.0 wire requirement
changes; this is a private local supervisor/host capability.

## Protected integration

The supervisor independently approves image bytes and SHA-256, image path,
policy/component descriptors and their exact artifact union. Both descriptors
must contain the image with the same approved digest. The copied artifacts
must verify against those descriptors; unknown or duplicate artifacts refuse.
Image bytes are bounded by the image library (64 MiB); each artifact is at most
64 MiB, the union at most 256 MiB/4096 entries and each descriptor at most 1 MiB.
A digest derived from arbitrary caller bytes is not independent approval.

```go
supervisor, err := guardchannel.Start(ctx, guardchannel.Config{
    Image: guardimage.Config{
        Image: approvedImageBytes, SHA256: approvedImageSHA256,
        Limit: approvedImageByteLimit,
        Stdin: ownedInputFile, Stdout: ownedOutputFile, Stderr: ownedErrorFile,
    },
    ImagePath: approvedImagePath,
    Policy: approvedPolicy, Manifest: approvedManifest,
    Artifacts: approvedOwnedArtifactUnion,
    MaxChecks: 1000, Timeout: 5 * time.Second,
})
if err != nil { return err }
// Retain supervisor until Close succeeds, including failed child bootstrap.
```

The launcher supplies exactly one private measurement socket as descriptor 4,
beside the sealed executable at descriptor 3. It accepts no caller replacement
socket, network listener or mutable socket path. Successful startup establishes
resource ownership; it does not mean that child bootstrap or any execution has
been accepted. A failed bootstrap closes the channel and retains the child for
explicit supervisor cleanup. Standard-stream handles remain caller-owned.

Only the approved child's trusted native initialization takes that endpoint:

```go
measurement, err := guardchannel.OpenChild(ctx,
    os.NewFile(4, "protected-child"), protectedChildLoadedRuntime, 5*time.Second)
if err != nil { return err }
factory, err := guardcalculator.NewFactory(guardcalculator.Config{
    Measurement: measurement, MaxInstances: 2,
})
if err != nil { return err }
// Connect factory to guardbinding.Open, then its private Binding to toolhost.Open.
// Only the core's authenticated admitted worker invokes that binding's Execute.
```

`protectedChildLoadedRuntime` is mandatory, bounded, cancellation-aware,
concurrency-safe and non-reentrant. It verifies the approved evaluator, factory,
compiled calculator and dependencies actually loaded in this child, with the
required protected pre-load/retained runtime identity and isolation assurance.
It must never be a digest echo, synthetic success or appraisal of the parent.
It remains a deployment provider requirement. The channel cannot make partial
backing-object observations into that complete assurance.

Keep supervisor, endpoint, measurement/factory/operation/native-owner handles,
original input, policies, journals and signing capabilities outside model,
plugin and MCP custody. Models get proposals and schemas only. `Measurement.Check`
and the private binding are not unsigned model-facing tool dispatch APIs.

## Identity, freshness and refusal

A Linux Unix-domain `SOCK_SEQPACKET` socket pair carries fixed 112-byte records:
a closed format/kind, random 32-byte connection generation, monotonically
increasing sequence and the two approved 32-byte commitments. These records
are local appraisal requests/acknowledgements, never authorization tokens or
transferable attestation. No signer or original request data is transmitted.

The supervisor obtains the child PID from a fresh retained-pidfd executable
observation before bootstrap. Every incoming message must carry exactly one
kernel-checked sender credential matching that child PID, real UID and GID.
The child checks its parent PID/UID/GID on every response. This uses per-message
`SO_PASSCRED`/`SCM_CREDENTIALS`, rather than a socket-pair creation-time peer
credential mistaken for the later inherited child. Missing, foreign, duplicate
or truncated credentials refuse. Unsolicited received descriptors are closed
on refusal and never retained as capabilities. Kernel credential checks require
protection from privileged credential impersonation capabilities as well as
ordinary descriptor theft.

Child initialization verifies the inherited socket domain/type and sets and
rechecks `FD_CLOEXEC`. The private generation is new per launch; neither another
process nor a stale acknowledgement can substitute for the fixed endpoint,
next sequence and exact commitments. A connection generation is not itself a
proof of logical MCP worker or exec generation. Core native admission retains
its own current worker generation, reservation and final effect gate.

Each `Check` verifies the child's exact approved snapshot commitments and the
mandatory local provider, then asks the supervisor for a fresh observation of
that same child. The supervisor rechecks executable object identity, seals,
content and executable-map metadata before acknowledging the exact request.
Pidfd liveness retries only an interrupted zero-timeout poll, at most three
times; each retry still refuses an exit event or other error.
Gate waiting, local provider and IPC share the child's finite per-check deadline.
Supervisor observations/responses also have finite deadlines. `Timeout` is
1 ms–5 s; one private session permits 1–1,000,000 checks. Idle endpoint ownership
lasts until explicit shutdown, without issuing an approval from a positive cache.

Local provider failure/panic, wrong snapshot, stale/wrong acknowledgement,
EOF, exhausted budget, cancellation during an exchange/provider or failed observation
permanently retires that capability. There is no measurement fallback or automatic
reconnection. A pre-entry cancellation or bounded gate-wait timeout refuses the attempt without issuing a request; cleanup uses a fresh
bounded context. Callbacks must honor their bounds; a Go wrapper cannot forcibly
interrupt an uncooperative trusted callback.

Backing-object/proc map checks are not private instruction-page attestation,
remote attestation, a sandbox or complete effect mediation. An observation and
later child admission are not an atomic cross-process transaction and do not
exclude transient changes. Protected child-local assurance and native final
admission remain necessary. Do not use this channel to measure a calculator in
the supervisor or treat file seals as operating-system isolation.

## Resource lifetime and safe verification

Retire/drain native hosts, accepted effects and operations before closing the
factory and child measurement. Normal child exit releases the inherited socket;
the supervisor then retires its observer session. `Supervisor.Close` retires IPC,
awaits the bounded observer, closes its endpoint and terminates/reaps only its
owned child. Cancelled cleanup preserves resource ownership for retry. Emergency
termination can leave uncertain journal completions; successful cleanup proves
no effect rollback. Parent shutdown does not revoke an acknowledgement atomically
with an already admitted child effect.

Scenario units exercise exact baseline/union, typed-nil assurance, closed
bootstrap/record formats, generation/sequence/snapshot mismatches, timeout,
provider panic/failure, retirement and cancelled cleanup/retry. Credential and
unsolicited-descriptor refusal scenarios use controlled unit-only parser input;
no adversarial IPC sender, impersonation or host-bypass program is generated.

Linux runtime tests build a fixed static child from `testdata/native`, launch
its exact sealed object and connect the real supervisor. In that child the
actual compiled calculator is behind `guardbinding`, native intent issuance,
authenticated encrypted loopback MCP, durable reservations and verified signed
terminal delivery. It calculates only `2 + 3` and verifies `5`. The parent
requires multiple fresh observations, child exit and bounded cleanup. Separate
safe runs refuse an exhausted check budget before arithmetic, missing channel,
and cancelled supervisor cleanup before the fixed run command. Linux CI must
execute the real backend; unsupported behavior fails instead of being skipped.

The fixed test program uses ephemeral keys, private temporary journals and
explicit fixture Registry/local-assurance providers. It does not establish
production isolation, authoritative blockchain binding or complete conformance.
Its injected test clock preserves replay quarantine and the Client's required
one-second reopen/poll interval; finite worker bounds accommodate repeated full
executable checks. Core admission and authentication conditions are unchanged.

```sh
go test -mod=readonly -race -timeout 3m ./core/guardchannel
```

Native Linux amd64/arm64 is the only supported profile. Other platforms refuse.
The development Mac can run safe Linux arm64 tests in an unprivileged container
with networking disabled, source/modules read-only and executable temporary
build storage. macOS units separately verify protocol and unsupported behavior.

Inspector's four previous ADK source snapshots retain their existing pins; this
change needs a separate reviewed source snapshot. Production host/Source
selection, protected deployment providers, parent-hop assembly and independent
observations remain in the approved order. Thirteen deployed host controls and
independent hop execution remain `NOT_RUN`; full conformance remains
`NOT_ESTABLISHED`. This is no demo, contract upgrade or normative A2A/DID change.

References: [Linux Unix credentials and socket types](https://man7.org/linux/man-pages/man7/unix.7.html),
[receive truncation and descriptor ownership](https://man7.org/linux/man-pages/man2/recvmsg.2.html),
[interrupted polling](https://man7.org/linux/man-pages/man2/poll.2.html),
[sealed child trust boundary](sealed-host-image.md) and
[compiled calculator integration](compiled-calculator-binding.md).
