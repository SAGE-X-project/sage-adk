# Owned protected A2A HTTP carriage

The `core/a2abinding` package provides `NewHTTPTransportOwner` and
`NewHTTPIngress` for the adopted SAGE 0.10.0 A2A 1.0 binding. These are explicit
library assemblies over the public Go core, not an unsigned JSON-RPC wrapper.
Supply a completed, unused session, exact registered HTTPS interface, trusted
TLS configuration, message TTL and bounded IO timeout. The constructor transfers
session ownership with `TakeHTTPApplication`; old session aliases cannot send,
receive or reclaim it. Copies of the new wrapper share one private state.

A Client still requires a configured `Verifier` and its authoritative fresh
Source. The Client privately retains exact owned card bytes, original record
version and selected interface for repeated local verification. Neither a card
key, received DID document, alias, header, JSON hint nor an LLM decision supplies
Registry authority. Source, clock, signing custody, immutable history, readiness
and deployment identity remain explicit trusted host dependencies. Core and
Verifier clocks must have the same Unix meaning; monotonic comparisons use only
the original core clock domain. Fixtures do not establish these operating facts.

## Client operation

Pass the concrete owner to `NewClient(verifier, owner)` and use `Client.Call`.
The owner establishes a dedicated TCP connection and verified TLS before preparing
the signed request. It serializes a complete private HTTP frame, then rechecks
both core participants and the exact card/interface. The Client's `finalCheck`
is called once; a separate private fresh-card callback supports the final local
bound and post-response checks without calling that once-only gate twice.

The final interval starts before all final core/card Source observations. One
`LocalStamp` from the original core clock then checks the five-second budget,
session/key liveness, retained signed message bounds and card/key expiry. Card
validity has no grace; wire timing retains the adopted 30-second created skew.
Only a single direct TLS application write follows. There are no subsequent
routing, signing, Source, buffering, serialization or discovery callbacks before
that write. No redirect, proxy, automatic auth, transport retry or downgrade is
available. A successful local admission does not promise complete transmission,
remote receipt or future authority during a blocking TLS/TCP write.

After reading the actual bounded HTTP response, the core checks both signatures,
AEAD, exact request hash/IDs/session/participants and transactional replay. The
adapter accepts a usable result only for signed transport success and HTTP 200,
then freshly verifies the current card/core/time before publishing Sender/Payload.
`Client.Call` checks JSON-RPC correlation. Independent response metadata may be
absent or different: the request's A2A version/extension fields are not invented
as mandatory response echoes. JSON-RPC application errors are inside an
authenticated successful SAGE response; a SAGE transport failure is not a usable
application result.

## Receiver operation

`NewHTTPIngress` takes the actual configured A2A `Server` and its unused responder
session. `ServeTLS(ctx, listener)` takes exclusive ownership of a listener and
serves a sequential, fixed-peer/session TLS-only path. It does not export a bare
JSON-RPC network handler or register another binding for that agent. The host must
keep the trusted library `Server.Handle`, Handler and key providers inaccessible
to untrusted processes and must not install an alternate unprotected endpoint.
This assembly is not a general multi-agent/session routing service.

Ingress obtains method, raw target, authority, field occurrences and body from
its owned TLS/HTTP connection. It inspects bounded raw header bytes before a
standard-library parser can combine duplicate lengths. Only after core request
acceptance does it pass the owned payload/metadata/recipient to the real
`Server`. The private final hook encloses the final Server card verification in
the same five-second core/card interval and takes its last local sample before
actual Handler admission. The hook must complete exactly one successful
verification. Tenant/card callbacks run outside the core endpoint lock.

Handler effects still require the host's Execution Guard and local policy.
Message authentication is not authority to perform effects. Untrusted plugin or
MCP code must not be able to call the Handler or signing custody around that
boundary. A locally admitted operation may already have effects when a later
response fails; its outcome is uncertain and is not automatically retried.

Core acceptance is never rolled back by policy refusal. Unsupported methods
produce signed JSON-RPC errors without Handler dispatch. Other policy refusal
can complete one signed `policy_denied` transport response when current authority
still permits it. That fully emitted authenticated denial completes the request
and can leave the server session alive; it does not free its ID/nonce. An
accepted request whose response fails or is uncertain retires the ingress state.
Framing/unauthenticated input refusal closes only that connection unless the core
itself has retired the session, limiting unnecessary session-wide denial of
service. Nothing revives a core failure. If authority to sign is missing, ingress
fails closed instead of manufacturing a signed result. Unsigned 400/401 failures
are generic and never authorize result use, retry or downgrade.

## Explicit local subset and shutdown

This implementation uses TLS 1.3 with exact HTTP/1.1 ALPN, one fixed-length
request per connection, header limit 16384 and envelope body limit 32768. It
rejects duplicate headers, noncanonical lengths, transfer/content encodings,
trailers, Upgrade, Expect, proxy/Forwarded routing and target normalization.
These are local subset choices, not new universal SAGE/RFC requirements. The
core's complete AAD 4033, record 16384 and plaintext 16348 limits still apply;
full chapter 08 large-message/task-field coverage is not claimed. Successful
result publication also requires the original request deadline to remain live,
a stricter local exchange policy. Configured URLs must be exactly representable,
canonical core targets; no implicit slash, default port or host-case rewriting.

TLS trust pools and public certificate buffers are copied. Verification override,
key logging, custom Rand/Time and alternate protocol configuration are refused;
private certificate custody/callbacks remain trusted. Timeout must be positive
and at most one minute, and TTL is one to 300 seconds. The dedicated dialer is
trusted host routing and cannot be supplied by discovery content.

Close sets shared terminal state and cancels the active operation before waiting
for serialization. Cancellation is registered before DNS/TCP setup. It closes
active connection/listener IO first; a late connection is refused. Each context
hook captures its own connection, so old cancellation cannot close a subsequent
explicit call. Partial/uncertain writes are terminal and are never retried.
Trusted synchronous providers must honor their bounded, non-reentrant contracts;
the owner cannot preempt a callback that violates its contract or promise
instantaneous key erasure while that callback holds a core lock.

## Validation scope

Owned-key unit scenarios cover framing, final verification timing, copy/closure,
context, unavailable Source, malformed input and partial writes. Mutated framing
and cryptographic/alias/document controls are unit-only. Runtime tests use real
Client, dedicated TLS owner, ingress, Server and an inert Handler; they carry
normal requests, signed unsupported-method responses and operating failure or
expiry/cancellation conditions. No attack tool or external probe is provided.

CI retains exact source/tree/compiler/module information, actual test output,
coverage and test binary hash. Those facts do not establish an operating Registry,
measured host, complete Inspector conformance, historical DIDP04 closure or an
organizationally independent audit. Fresh Inspector observations remain a
separate integration task.

## Memory-only ingress refusal evidence

The grouped ingress controls pair one positive calibration with bare JSON-RPC, unsigned legacy, unsigned HTTP/1.1 gRPC-media and h2-only ALPN inputs. Every refused input stays in `_test.go` over owned `net.Pipe` TLS; there is no packet CLI or OS listener for these controls. Existing normal loopback runtime scenarios remain separate.

Observations bind actual server/completion identities, configured target and fixture Source service. TLS uses its explicit `localhost` fixture certificate over the memory pipe; the configured HTTP authority remains `agents.example.com`. This is not proof of a deployed endpoint or alternate gRPC/legacy listener. The h2 control requires the pinned toolchain's no-application-protocol alert, with a live context, rather than treating a timeout or arbitrary handshake error as successful refusal.

Each control starts with an actual pending response context. The positive path must reach the inert Handler and authenticate the exact response; refused paths must record zero effects/accepted requests/accepted responses. Actual cleartext failure framing and unsuccessful response admission are recorded. No parser or OpenRequest callback is exposed by connection handling, so their trace remains NOT_OBSERVED; failed h2 negotiation is NOT_REACHED. Three separate fresh-fixture direct parser/core calibrations are explicitly distinguished from these connection traces.

The evidence parser reconstructs split Go output per exact test leaf and checks typed raw frame hashes, source/role/URL facts, counters, actual verified payload, interface negotiation, calibration scopes and failure framing. Both full and focused test runs must independently provide all five witnesses. These are bounded memory-unit observations, not whole conformance, operating Registry/custody/host assurance, or closure of the historical Inspector production-ingress unsupported cases.

Positive raw request and response envelopes are checked for exact fixture DID/key roles and versions, content digests, profile signature framing, context/session equality and request identifier/hash correlation. These are bounded structural and byte-consistency checks; actual signature authentication and plaintext acceptance are observed through the real core calls. The synthetic parser fixtures contain deliberately invalid cryptographic material and establish no runtime result.
