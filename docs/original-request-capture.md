# Original request capture

Use `core/capture` at the trusted input boundary, before parsing, model expansion
or plugin processing. Supply the exact ordered UTF-8 input byte list; do not
re-serialize an already decoded Message as the original.

```go
store, err := capture.OpenFileStore("/protected-host/originals")
if err != nil { return err }
defer store.Close()
host, err := capture.NewHost(store)
if err != nil { return err }
request, response, err := agent.ProcessOriginal(ctx, myAgent, host, rawInputs, decode)
if err != nil { return err }
// Keep request with the trusted host; response is the ordinary handler response.
issuer, err := request.NewIntentIssuer(ctx, trustedIssuerServices)
if err != nil { return err }
defer issuer.Retire()
```

Imports are `github.com/sage-x-project/sage-adk/core/capture` and
`github.com/sage-x-project/sage-adk/core/agent`. The directory's ancestors must
already exist and be protected; the final directory is created with mode 0700.
On Linux and macOS, records are exclusive regular files with mode 0600, bounded
reads and file/directory synchronization. Unsupported platforms refuse FileStore
construction. A failed write can retain an incomplete record; never automatically
overwrite or delete it to retry. The host owns reconciliation and retention.

For a non-agent consumer, use `host.Capture(ctx, rawInputs)` directly. The returned
Request has a fresh UUIDv4, the original commitment, `Inputs(ctx)` to reload and
verify retained bytes, and `NewIntentIssuer` to bind the capture to Guard.
Core commitment limits are 1024 inputs and 1 MiB total valid UTF-8, without Unicode
normalization. Empty lists and empty input items retain their exact framing.

`agent.ProcessOriginal` persists before calling the host decoder and rechecks
storage before the handler. The decoder receives a copy. Decode/handler failures
can return an already persisted Request for trusted audit; they do not erase it.
A capture error always returns no response. It must stop the request.

Both issuer and Client policies recheck the retained original. A policy binding
must match its digest; changes and store failures refuse approval. The signer
wrapper also checks retained input immediately before delegating key use. All
core policy, authority, clock, sender, key custody and measurement providers are
still required and must enforce their own rules. No permissive default is added.

For recovery, persist `request.ID()` and `request.Digest()` in the protected host
operation checkpoint. A restarted host can call `Restore(ctx, id, digest)` on the
same Store. Restore validates bytes against that checkpoint and rejects a changed
original. The core's stable issuance path and Client journal remain necessary;
Restore does not grant approval, reset a fence or permit automatic re-signing.

These APIs require trusted custody. Do not expose Host, Store, Request, issuer or
key/policy capabilities to model or plugin code. Providers must be bounded,
honor cancellation and remain immutable during an operation. Unix file permissions
do not isolate code running as the same user. Protect storage and checkpoints
against untrusted writers and rollback. Stored original inputs are plaintext;
access and retention must account for sensitive request contents.

Use the [guarded tool host](guarded-tool-host.md) for explicit native MCP effect
admission. Its authenticated connection can reopen the already issued Client
through `request.OpenMCPClient(ctx, connection, journalPath, signedIntent, services)`.
First close the issued Client successfully, keep the same protected path and fence,
and serialize ownership in the host. Reopening never creates missing history and
does not re-sign. The original is reloaded here and by the bound Client policy
before later handoffs. This close-and-reopen transfer is not atomic across processes.

This is an explicit original-input boundary, not a fully protected agent host.
Ordinary Process, transport servers, direct LLM calls and tools do not
implicitly enter it. Effects outside that explicit native host, deployment controls and Inspector
runtime evidence still require binding. The historical readiness evidence is
not upgraded by these local tests.
