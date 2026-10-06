# Guard clock and signing bindings

`core/guardservices` connects the public Guard signer ports to fixed protected
Ed25519 custody and current Registry authority. It supplies no default policy,
Registry Source, key generation, key export, HSM or operating-system isolation.

Share one `NewSystemClock()` instance with every Registry Gate, HPKE endpoint,
native host and Client in the process. `Now` supplies seconds plus monotonic
milliseconds; `Sample` supplies UTC and monotonic milliseconds. Local OS time
is the authority being observed, not an external time attestation. Wall or
monotonic rollback permanently refuses samples. Do not reset the clock, replay
history or operation identity after a failure. Normal journal recovery and the
core replay quarantine remain necessary across process restarts.

Create a Registry Gate with an explicitly selected and validating `Source`,
protected durable Store and that clock, then create a RegistryAuthority for the
exact DID and signing-key URL. A Source must validate complete records, proofs,
network/source binding, readiness and finality. Remote booleans or local fixture
assertions do not establish those facts. The authority pins the first selected
key bytes; changing keys requires trusted lifecycle reconfiguration.

Implement `Ed25519Backend` around one fixed protected key, using bounded,
context-aware `PublicKey` and `Sign` calls. It exposes no private-key export. The
actual backend must honor cancellation, return owned bytes and remain immutable,
concurrency-safe and non-reentrant. No wrapper can interrupt a blocked provider
that ignores cancellation or prove that a local process isolated its credentials.

```go
import providers "github.com/sage-x-project/sage-adk/core/guardservices"

clock := providers.NewSystemClock()
// Build issuerAuthority/executorAuthority from selected validating Registry gates.
intentSigner, err := providers.NewIntentSigner(ctx, issuerAuthority,
    issuerDID, issuerKeyURL, issuerCustody)
if err != nil { return err }
resultSigner, err := providers.NewResultSigner(ctx, executorAuthority,
    executorDID, executorKeyURL, executorCustody)
if err != nil { return err }
// Pass intentSigner only to capture.Request.NewIntentIssuer services.
// Pass resultSigner to toolhost.Services, with the same executor authority.
// Supply independently selected policy, original store and measured instances.
```

Construction checks currently active role-bound registered Ed25519 bytes against
the backend. Signing repeats this check immediately before key use and before
publishing the verified proof. Key mismatch, revocation, expiration, unavailable
observations, invalid proof, provider failure, panic and cancellation refuse the
operation. Once signing started, failure does not undo that key use; retain the
core issuance fence and reconcile instead of re-signing under a new path.
X25519 KEM keys and alternate signing suites are never substituted.

The request adapter accepts only `sage-execution-intent|0.10.0` followed by the
NUL separator and canonical object bytes. The result adapter accepts only
`sage-tool-result|0.10.0` with the same separator. Both enforce the configured
issuer/key URL and exact version/Ed25519 fields. These are low-level custody
adapters, not an approval API or complete unsigned-message validator. The core
issuer and result owner still validate the complete closed protocol message,
policy, opaque one-use approval, operation identity, admission and durable state.
Never expose either signer/backend to model or plugin-controlled code. HPKE
handshake endpoint custody remains a separately configured protected factory;
these adapters do not replace its seed/KEM requirements.

The existing native runtime and external consumer now use these public adapters
for actual intent/result signing with ephemeral Ed25519 keys and fixture Registry
observations. Clock regression and provider failures are controlled unit scenarios.
This is library binding evidence only. The actual blockchain Source still needs
its chain ID, RPC, Registry address, ABI/bytecode version and finality policy pinned
before deployment observation can run. Actual loaded-instance/effect inventory
and independent observer remain necessary. Historical Inspector NOT_RUN verdicts
and the normative 0.10.0 pin are unchanged.
