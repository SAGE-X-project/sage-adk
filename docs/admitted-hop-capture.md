# Capturing an admitted downstream call

`core/capture.HopRequest` connects the existing native Go core hop issuer and
MCP Client to a retained ADK original. It is separate from the root `Request`.
The trusted B host captures the exact authenticated A-to-B envelope as one new
local input, with a fresh request ID and its own original commitment. B-to-C
requires B's independently approved policy, recipient, key and loaded component;
a parent ID is causal metadata, never inherited user authorization.

This uses the existing SAGE 0.10.0 EXEC-01/02 rules and native core APIs. It adds
no wire field, delegation credential, RFC requirement or normative amendment.

## Trusted native entry

Only protected host code running inside the core's admitted `MCPExecutor.Run`
receives the live `Invocation`. Keep invocation, capture store, checkpoint,
upstream providers, own policy, loaded-code measurement, journals and signing
custody outside models/plugins. Providers must be bounded, concurrent-safe and
non-reentrant and outlive accepted work. Use a finite native-worker context.
Do not pass the invocation to an ordinary loaded tool callback.

```go
hop, err := capturer.CaptureHop(ctx, capture.HopServices{
    Recipient: localDID,
    Invocation: admittedInvocation,
    Authority: protectedUpstreamAuthority,
    Policy: protectedUpstreamPolicy,
})
if err != nil { return err }
issuer, err := hop.NewIntentIssuer(ctx, protectedOwnIssuerServices)
if err != nil { return err }
// Authorize the exact own-policy proposal, then Issue once to a stable journal.
```

`CaptureHop` verifies the exact live native parent and authenticated envelope
before persistence and again after readback. It checks the local configured
recipient and invocation digest against core verification. A zero or ordinary
unadmitted invocation grants no capture and never reaches the store. The
returned type exposes no root Request or parent admission handle.

`Inputs` re-verifies current upstream authority/policy and native parent
admission before and after exact durable readback. The stored list must contain
exactly the same one envelope; upstream request identity cannot substitute for
the new local identity. An observed failure or provider panic permanently retires
the capability. Pre-entry cancellation refuses without a new operation. An
already returned observation is not atomic with future parent retirement.

`NewIntentIssuer` requires own issuer DID to equal this B host. It wraps current
capture/parent checks around local policy and signing, and calls the core's
`NewHopIntentIssuer`. Core one-use approval, signing-role checks, fresh call ID,
causal parent ID, canonical arguments, durable issuance fence and post-signing
verification remain mandatory. No permissive signer or policy is supplied.
Original storage, descriptor identity and loaded-runtime assurance remain
protected deployment providers; retaining bytes alone does not establish them.

## Journal transfer and lifetime

After issuing, retain exact signed bytes and successfully close the Client
before transfer to an authenticated downstream native connection:

```go
err = hop.OpenMCPClient(ctx, nativeConnection, stableJournalPath,
    exactIssuedIntent, protectedOwnClientServices)
```

The transfer rechecks current retained original and parent and calls the core's
owner-bound `OpenHopClient` with `create=false`. Missing history refuses; there
is no unsigned transport, caller-selected sender, substituted original or root
fallback. Native Client checks continue to use wrapped capture/parent policy.
Close/reopen transfer is serialized host administration, not an atomic
cross-process transaction. Keep issuance fence and journal protected against
replacement/rollback and preserve uncertain results.

`RestoreHop` requires trusted checkpoint ID/digest, the exact original envelope
and a currently admitted parent again. It allocates no new request ID and
provides no authority by itself. An expired, completed or UNKNOWN parent cannot
be resurrected through storage recovery. A process restart normally has no live
parent: deny until the authorized workflow can establish the required state.
Do not create another capture/path/call merely to evade refusal or UNKNOWN;
independent policy must distinguish a new request from a retry, and reconciliation
must preserve the stable operation identity.

Keep the parent worker running until accepted downstream work and dependent
cleanup end. Cancellation/retirement can leave uncertain downstream effects;
parent loss does not undo an already accepted effect, and no exactly-once claim
is made. Drain native owners before closing capture stores or replacing providers.

## Safe verification and scope

Units refuse absent/unadmitted invocations before persistence and invalid,
retired or cancelled capabilities without key use or transport. Existing
capture wrapper units and root runtime tests preserve root behavior.

Safe runtime tests use a fixed A-to-B-to-A loopback, where A also acts as the
separately configured final receiver. They exercise actual admitted native
workers, two authenticated encrypted MCP connections, protected issuance,
separate policy commitments, durable Client handoff, signed terminal results
and fixed `2 + 3 = 5` only. Allowed and same-live-parent restore paths each sign
once and execute one final effect. Upstream policy loss, own-policy denial and
missing Client history yield no final arithmetic effect. Completed parent
capabilities refuse reuse. Per-message metadata is not silently converted into
root authority. The responder key tuple selects the registered signing key;
X25519 KEM selection remains a distinct role.

Only ephemeral keys, local private journals and explicit fixture Registry,
policy and measurement providers are used. There is no adversarial sender,
attack-capable reproduction, external service or host-bypass program.

```sh
go test -mod=readonly -race -timeout 3m ./core/capture ./core/toolhost
```

This completes the retained hop capture/issuer/native Client library connection.
The existing `guardbinding.Open` exact-operation helper remains root-only;
concrete independently approved hop operation/loader binding is the next assembly
item. The ordinary `toolhost.LoadedTool` intentionally receives only exact
arguments, not native invocation or signing authority. Trusted hop orchestration
needs a native host coordinator, not an unsigned model tool dispatcher.

Inspector's five historical ADK snapshots keep their pins; this change requires
another explicit reviewed source snapshot. One-core fixture runtime evidence
is separate from independent cross-core hop execution, authoritative blockchain
Source, selected protected host/isolation and independent deployment observation.
All thirteen deployed controls and independent hop execution remain `NOT_RUN`;
full conformance remains `NOT_ESTABLISHED`. Preserve the approved order before
later demos, contract upgrades and normative A2A/DID work.
