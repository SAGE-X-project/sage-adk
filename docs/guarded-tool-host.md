# Guarded native tool host

Use `core/toolhost` to connect a fixed, protected tool instance to the public SAGE
0.10.0 native MCP owner. The owner authenticates the encrypted connection, admits
signed intents, owns durable execution reservations and signs terminal results.
The ADK adapter freshly checks the live admission, signed intent, current registry
authority, policy and loaded instance immediately before invoking the tool.

```go
import (
    "github.com/sage-x-project/sage-adk/core/toolhost"
    guard "github.com/sage-x-project/sage/pkg/agent/guard010"
)

host, err := toolhost.Open(ledgerPath, createNewScope, recipient,
    toolhost.Services{
        IntentAuthority: intentAuthority,
        ResultAuthority: resultAuthority,
        Policy: policy,
        Signer: resultSigner,
        Clock: clock,
    }, []toolhost.Binding{{
        Name: "sum", ManifestDigest: approvedManifestDigest, Tool: pinnedInstance,
    }}, bounds)
if err != nil { return err }
// Supply a trusted MCPConnectionHandler and finite MCPHostBounds. Its endpoint
// factory must return a fresh exclusively owned CompletionEndpoint010 per call.
err = host.Serve(ctx, listener, workers, connectionConfig, handler)
// Retain host and retry Close with a fresh context if cleanup times out.
closeErr := host.Close(cleanupCtx)
```

See [clock and signing bindings](guard-provider-bindings.md) for shared local
clock and registry-bound custody adapters. Actual Source/custody remain protected
deployment providers.

See [approved operation bindings](approved-operation-bindings.md) for a fixed
policy evaluator and exact artifact snapshots connected to the same instance.
Its Factory still supplies actual loaded-code attestation and remains protected.

All providers and bounds are mandatory; no permissive policy or signing default
is supplied. Use `Connect` for one owned outgoing connection, `Serve` for bounded
incoming connections, and `Close` to retire admission and await workers/provider
cleanup. The supported carriage is the core's private four-byte-length-framed
MCP stream. HTTP, stdio, A2A and legacy `tools.Registry` routes require separate
integration and do not acquire protection by constructing this host.

`LoadedTool.Check(ctx, manifestDigest, toolName)` and
`LoadedTool.Execute(ctx, canonicalArguments)` belong to the same fixed instance.
The manifest is approved protected configuration; do not derive it from an
untrusted peer or merely hash whatever code is currently present. Check must
verify the baseline, actual loaded instance and relevant dependencies. A file
hash by itself does not establish what code is executing. Bindings are copied;
providers and instances must remain immutable, bounded, concurrency-safe and
cancellation-aware until Close succeeds. Never let callbacks re-enter their owner.

Execute receives only a copy of the exact canonical JSON argument object. No
unsigned map conversion, default arguments, signer, host or admission token is
given to it. It must finish the actual effect and dependent cleanup before
returning a JSON object. The core bounds and canonicalizes output to at most
1 MiB. Measurement/policy failure prevents execution. Exceptions, cancellation
and invalid output yield UNKNOWN rather than consumable successful output. An
effect might already have happened in those latter cases; do not retry it
automatically. Results reach the caller only through verified journaled delivery.

For original requests, use `capture.Host.Capture` or `agent.ProcessOriginal` at
the trusted raw-input boundary, then `request.NewIntentIssuer`. Authorize a
proposal and Issue it to the stable protected Client journal path. Retain the
signed bytes from `JournaledIntent`, close that Client successfully, and inside
the authenticated connection handler call:

```go
err := request.OpenMCPClient(ctx, connection, sameJournalPath, signedIntent,
    guard.MCPClientServices{
        IntentAuthority: intentAuthority, ResultAuthority: resultAuthority,
        Policy: policy, Clock: clock,
    })
if err != nil { return err }
delivery, err := connection.Exchange()
// Pending is polled within core UTC/monotonic limits. Only the first verified
// completed terminal delivery exposes Output(); UNKNOWN never exposes output.
```

This reopens existing history with creation disabled and binds fresh capture
checks to later Client handoffs. Keep the issuance fence, journal, original and
checkpoint in protected custody; serialize the close-and-reopen handoff in the
host. It is not an atomic transfer across processes. Do not replace the operation
path after a failure. Missing/partial history is refused. Completed/uncertain
receiver reservations survive ordinary reopen, and terminal Clients refuse
another handoff. New replay journals require the core's 360-second UTC and
monotonic quarantine before setup can complete; do not bypass it in deployment.

Model/plugin code must receive proposals and schemas only. Keep raw tools,
Host/connection/issuer handles, registry/policy providers, signing keys and
protected storage outside its custody. Inventory every effect route, including
callbacks and alternate transports. A Go wrapper and same-user file permissions
do not establish operating-system isolation. The runtime fixture uses ephemeral
keys, private temporary journals and one inert arithmetic effect over loopback;
it proves this explicit library path, not authoritative deployment providers,
protection of other routes or complete Inspector conformance. Historical readiness
evidence remains unchanged until those bindings are verified against a pinned
deployment subject.
