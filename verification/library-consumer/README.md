# External library consumer check

Run `go test -mod=readonly -race -timeout 1m ./...` from this directory.
Only the ADK checkout under review is replaced. All SAGE and A2A dependencies
resolve to pinned public modules; no sibling repository is needed.

This exercises public builder/configuration/key APIs, PEM/JWK key roundtrips,
and one loopback A2A request to the ADK handler with an inert fixed input.
The standard suite also exercises the owned HTTP start/stop lifecycle.
No LLM, external chain, database or real tool effect is used.

This is build and legacy transport compatibility evidence. It does not verify
SAGE 0.10.0 Guard capture, authorization, final effect mediation, authoritative
registry ownership or deployed conformance. The A2A reply methods remain placeholders.
