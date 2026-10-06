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

These are local library/capture fixtures. They do not verify final effect
mediation, authoritative registry ownership or deployed conformance.
The A2A reply methods remain placeholders.
