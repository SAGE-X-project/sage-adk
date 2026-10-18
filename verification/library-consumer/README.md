# External library consumer check

Run `go test -mod=readonly -race -timeout 1m ./...` from this directory.
Only the ADK checkout under review is replaced. All SAGE and A2A dependencies
resolve to pinned public modules; no sibling repository is needed.

This exercises public builder/configuration/key APIs, PEM/JWK key roundtrips,
and one loopback A2A request to the ADK handler with an inert fixed input.
The standard suite also exercises the owned HTTP start/stop lifecycle.
No LLM, external chain, database or real tool effect is used.

The consumer also exercises the public original-capture boundary with exact UTF-8
inputs, a durable private FileStore, an independent commitment calculation and
restart recovery through a protected checkpoint fixture. The root unit suite
exercises actual local Ed25519 issuance and failure scenarios.

The consumer also runs the native guarded tool fixture through only public APIs:
capture, issuance, same-journal transfer, encrypted loopback setup, one inert sum
effect, signed verified delivery and ledger recovery.
`toolhost_fixture_test.go` and `toolhost_runtime_test.go` mirror the root positive
fixture except for the package name. They check public packaging compatibility,
not an independent protocol oracle. Both intent and result key use now run through public Registry-bound signing
adapters; fixture observations and ephemeral custody remain explicit. The public
shared-clock check exercises both clock interfaces and concurrent samples.
Root unit scenarios cover denials, including terminal handoff refusal and its
required connection retirement.

These are local fixtures. They do not verify effect routes outside the explicit
native host, authoritative registry ownership or deployed conformance.
The A2A reply methods remain placeholders.

## Adopted DID and card consumer

`did_consumer_test.go` imports only public ADK, SAGE and DID module APIs. Its
complete-record reader, clock, journal and signed records are owned fixtures,
not an operational chain binding. It checks explicit metadata for all three
states, owned TLS GET/POST success and negotiation diagnostics, and A2A card
verification against a fresh core Source. Inactive records may be displayed but
cannot make the card eligible. Card verification grants no message-execution
authority; the production protected transport remains separate work.

Only the ADK checkout under review is replaced. The core and both DID modules
come from checksum-verified public versions, with GOWORK disabled. CI retains
selected versions/checksums, verification results and fresh focused race test
events, while preserving the existing external consumer and full-suite gates.

The exact public core and DID versions/sums are in `did-dependencies.json`; CI
rejects any different selection or replacement of these three dependencies.

The Linux missing-channel runtime fixture checks descriptor kind before giving
OpenChild ownership. An unsupervised process can otherwise find the Go poller
at descriptor 4 and close an unowned resource while refusing the channel. Both
no inherited descriptor and an inherited ordinary file must deny without a
runtime fatal error or arithmetic effect. The production OpenChild ownership
and peer-authentication contracts are unchanged. Linux CI retains these two
refusal events separately from the ten DID consumer checks.
