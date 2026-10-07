# Sealed Linux host image supervision

`core/guardimage` supplies a restricted local launch and observation capability
for an independently approved static Go host executable. It is preparation for
the [compiled calculator binding](compiled-calculator-binding.md), not the
calculator's child-to-supervisor measurement bridge or a selected deployment.
It changes no normative SAGE 0.10.0 profile, HTTP signature or MCP wire format.

The first candidate implementation supports Linux `amd64` and `arm64`. It needs
executable memfd sealing, pidfds and trusted procfs. macOS and other targets
return `ErrUnsupported`; unsupported kernel policy also refuses. There is no
fallback to an ordinary mutable executable path, digest echo or synthetic
observation. Development and platform-neutral units run on macOS; the Linux
CI runner must execute the real safe runtime tests. Choosing this implementation
candidate does not select Linux as the eventual production host.

## Launch and observation

Trusted administration supplies exact executable bytes, an independently approved
lowercase SHA-256 digest, an explicit size limit of at most 64 MiB, and optional
owned standard-stream file handles. There are no argument or environment overrides,
path lookup, tool selection or peer provisioning. The caller must not route model,
plugin or peer input into this configuration. Image approval remains a protected
administrative decision; calculating and supplying a hash does not approve code.

```go
process, err := guardimage.Start(ctx, guardimage.Config{
    Image: approvedHostBytes,
    SHA256: independentlyApprovedDigest,
    Limit: 64 << 20,
    Stdin: protectedInputHandle,
    Stdout: protectedOutputHandle,
})
if err != nil { return err }
observation, err := process.Observe(ctx)
if err != nil { return err }
// Record observation with exact host/build/observer provenance, not as admission.
// Retire the native host and finish accepted work before process.Close(cleanupCtx).
```

The image must be a native-architecture, little-endian ELF64 `ET_EXEC` Go
executable. Dynamic/interpreter segments, executable stacks, writable executable
load segments, invalid file/entry ranges, non-Go metadata, CGO and other build
modes are refused. Go build metadata is a restriction on this supported profile,
not independent build provenance or semantic validation of approved code.

The launcher owns a defensive byte copy and checks its digest before preparation.
It writes a private executable memfd, applies write/grow/shrink/execute-mode and
seal-set seals, verifies those seals and hashes the **sealed object's actual
bytes again before exec**. It executes that same retained object through its
inherited descriptor. Hashing source or a mutable path after startup is not used
as launch provenance. The resulting child is bound to a retained pidfd; a PID
number alone is not treated as an instance capability.

Each observation checks that the pidfd still denotes a live process, opens the
kernel's executable object for that child, compares its device/inode/size to the
retained sealed object, verifies all required seals and hashes exact bytes.
A bounded procfs mapping read refuses writable executable mappings, anonymous
executable mappings and other executable backing objects. Kernel vDSO/vsyscall
mappings are explicitly platform-trusted exceptions, not approved userspace
artifacts. Liveness is checked again after the observation. A failed observation
permanently retires subsequent observations; it does not silently approve a
repaired path or another process.

`CheckSnapshot(ctx, snapshot, imagePath)` verifies the owned component and policy
artifact descriptors, requires this exact image in **both** descriptors and then
performs a fresh observation. It observes the launched child. It intentionally
does not implement `guardcalculator.Measurement` for a calculator in the parent,
since that would measure one process while authorizing code in another.

## Trust boundary and resource lifetime

The kernel, procfs, supervisor, approved executable and baseline administration
are trusted. A deployment must protect the supervisor, its retained handles,
child runtime, original request, policy, signing keys and final admission gate
from ordinary plugin/model/MCP write or impersonation capabilities. File seals
alone do not establish that OS isolation. No private signing key or unrestricted
signing API is supplied by this package.

This observer checks executable backing objects and executable-map metadata. It
does not inspect private instruction pages, prevent arbitrary syscalls or prove
that all runtime effects are mediated. A pidfd binds process lifetime, not a
logical worker or exec generation. Observations are not atomic with child
admission and do not prove absence of transient changes between observations;
the protected bridge must bind the worker generation and final gate separately. It is no sandbox, remote attestation,
whole-host integrity proof, semantic-safety verifier or authorization gate. A
trusted approved program must await native admission before a protected effect;
starting an approved binary does not itself admit any tool call. Dynamic plugins,
scripts, JITs, mutable native dependencies and additional executables are outside
this single-image profile. Deployment proof must also pin the kernel, platform
trust assumptions, protected launch configuration and independent observers.

The context bounds startup. After a successful start, explicit `Close` owns
shutdown; a later cancellation of the startup context does not silently kill
accepted work. `Close` immediately retires appraisal, terminates/reaps the owned
child and releases retained handles. Normal administration must first retire the
native host and finish accepted effects. Emergency termination can leave uncertain
completions, which their durable journals must retain; killing a process proves
no rollback. A cancelled cleanup keeps ownership for retry with a fresh bounded
context. Standard streams are file handles, not arbitrary blocking Reader/Writer
callbacks. They remain caller-owned; no host signing or effect result is inferred
from stream output.

## Safe verification and remaining assembly

Platform-neutral units parse a real cross-built static Go image and exercise
baseline/bounds, ELF/build-profile refusals, owned-byte isolation, uncovered
executable-map metadata, descriptor coverage, failure retirement, cancelled
cleanup/retry and retirement during a controlled unit callback. Altered image
bytes and mapping refusals are **parsed only in units**; they are never executed.

Linux runtime tests launch a fixed harmless fixture containing ADK's actual
compiled calculator. They also run in the development Mac's network-disabled
Linux arm64 Docker environment with read-only source/module mounts; CI runs
native Linux amd64 and arm64 checks. The fixture waits on a private test pipe, computes only
`2 + 3`, returns `5` and exits. Tests compare repeated kernel-backed executable
observations, descriptor binding and exit/cleanup refusal. The fixture has no
keys, network access, dynamic tool selection or SAGE admission claim. A successful
Linux runtime is required; unsupported behavior fails the test rather than
substituting a fixture appraisal or skipping it. No vulnerability or host-bypass
program is introduced.

```sh
go test -mod=readonly -race -timeout 3m ./core/guardimage
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -mod=readonly -c ./core/guardimage
```

Next, connect a protected child-to-supervisor measurement channel to the same
child's native calculator gate, with exact instance/snapshot ownership, fresh
observations, cancellation and denial on missing supervisor. Then assemble the
host's original capture, policy, custody, journals, final effects and authoritative
blockchain Source. The existing Registry mapping obligations and later contract
upgrade stage remain unchanged. Inspector's three historical ADK source reports
retain their pins; a new reviewed source snapshot is needed for this package.
All thirteen deployed host controls and independent hop execution remain NOT_RUN;
full conformance remains NOT_ESTABLISHED. This is implementation preparation in
the approved order, not demo work or a normative A2A/DID upgrade.

References: [SAGE load binding](https://github.com/SAGE-X-project/sage-spec/blob/1820ab5eafb843e1c13f4c46c34aeeb28d934ac9/profiles/agent-mcp-security.md#6-component-manifest-and-load-binding--exec-06),
[Linux executable memfd policy](https://www.kernel.org/doc/html/latest/userspace-api/mfd_noexec.html),
[file sealing](https://man7.org/linux/man-pages/man2/F_ADD_SEALS.2const.html),
[proc executable object](https://man7.org/linux/man-pages/man5/proc_pid_exe.5.html),
[proc mappings](https://man7.org/linux/man-pages/man5/proc_pid_maps.5.html) and
[pidfd liveness](https://man7.org/linux/man-pages/man2/pidfd_open.2.html).
