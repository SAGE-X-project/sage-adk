// SPDX-License-Identifier: LGPL-3.0-or-later
package a2abinding

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/sage-x-project/did-go/advanced/sage"
	didgo "github.com/sage-x-project/did-go/did"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// BindingURI is the exact protected A2A transport binding identifier, never fetched.
const BindingURI = "https://sage-x-project.github.io/sage-spec/a2a/0.10.0/binding"

// ExtensionURI is the exact required SAGE extension identifier, never fetched.
const ExtensionURI = "https://sage-x-project.github.io/sage-spec/a2a/0.10.0/extension"

// CardType is the SAGE JWS protected header type.
const CardType = "sage-a2a-card+jws"

// ErrRejected is the sole public binding verification failure.
var ErrRejected = errors.New("authentication failed")

// Interface is a card interface eligible against the same fresh record.
type Interface struct {
	URL    string
	Tenant string
}

// VerifierConfig supplies locally trusted dependencies. Source must acquire and
// validate complete fresh REG-01 records, proofs, readiness and immutable history.
// A peer-supplied Source or self-asserted card key is never sufficient.
type VerifierConfig struct {
	Source       sage.Source
	Clock        registry010.Clock
	ExpectedPeer string
}

// Verifier serializes clock and record observations, never caching positive authority.
type Verifier struct {
	mu   sync.Mutex
	cfg  VerifierConfig
	last *registry010.Stamp
}

// VerifiedCard is owned public discovery data. It confers no execution authority.
type VerifiedCard struct {
	raw           []byte
	did, version  string
	interfaces    []Interface
	otherRequired []string
	bounds        cardBounds
}

// Private retained bounds come only from successful local card verification.
// They are data for the adapter's last same-clock check, never an effect permit.
type cardBounds struct {
	issued, expires, keyExpires int64
	keyHasExpiry                bool
}

func (b cardBounds) live(unix int64) bool {
	return b.issued <= unix && unix < b.expires && (!b.keyHasExpiry || unix < b.keyExpires)
}

// JSON returns an owned copy of the normalized, verified card consumers may expose.
func (c *VerifiedCard) JSON() []byte { return append([]byte(nil), c.raw...) }

// DID is the subject authenticated by the configured Source.
func (c *VerifiedCard) DID() string { return c.did }

// RecordVersion is the exact observed registry version, never a reused grant.
func (c *VerifiedCard) RecordVersion() string { return c.version }

// Interfaces returns owned eligible interfaces in card order.
func (c *VerifiedCard) Interfaces() []Interface { return append([]Interface(nil), c.interfaces...) }

// NewVerifier selects no Source, clock or expected identity implicitly.
func NewVerifier(c VerifierConfig) (*Verifier, error) {
	if c.Source == nil || c.Clock == nil {
		return nil, ErrRejected
	}
	if c.ExpectedPeer != "" {
		if _, e := sagedid.ParseDID010(c.ExpectedPeer); e != nil {
			return nil, ErrRejected
		}
	}
	return &Verifier{cfg: c}, nil
}
func (v *Verifier) sample() (registry010.Stamp, error) {
	t, e := v.cfg.Clock.Now()
	if e != nil || t.MonoMS < 0 || t.Unix < 0 || t.Unix > 9007199254740991 || (v.last != nil && (t.MonoMS < v.last.MonoMS || t.Unix < v.last.Unix)) {
		return t, ErrRejected
	}
	v.last = &t
	return t, nil
}
func canonical(v any) ([]byte, error) {
	b, e := jcs.Marshal(v)
	if e != nil {
		return nil, ErrRejected
	}
	return b, nil
}
func binary(s string) ([]byte, bool) {
	b, e := base64.RawURLEncoding.DecodeString(s)
	return b, e == nil && base64.RawURLEncoding.EncodeToString(b) == s
}
func version(s string) bool {
	n, e := strconv.ParseUint(s, 10, 64)
	return e == nil && n > 0 && strconv.FormatUint(n, 10) == s
}
func https(s string) bool {
	u, e := url.Parse(s)
	return e == nil && strings.EqualFold(u.Scheme, "https") && u.Host != "" && u.User == nil && u.Fragment == "" && !strings.Contains(s, "#")
}
func declaration(m map[string]any) (map[string]any, error) {
	caps, ok := m["capabilities"].(map[string]any)
	if !ok {
		return nil, ErrRejected
	}
	for _, name := range []string{"streaming", "pushNotifications", "extendedAgentCard"} {
		if x, exists := caps[name]; exists && x != false {
			return nil, ErrRejected
		}
	}
	exts, _ := caps["extensions"].([]any)
	var params map[string]any
	for _, x := range exts {
		ext := x.(map[string]any)
		if text(ext, "uri") != ExtensionURI {
			continue
		}
		if params != nil || ext["required"] != true {
			return nil, ErrRejected
		}
		params, _ = ext["params"].(map[string]any)
		if len(params) != 5 {
			return nil, ErrRejected
		}
		for _, n := range []string{"sageVersion", "did", "recordVersion", "issued", "expires"} {
			if _, ok := params[n]; !ok {
				return nil, ErrRejected
			}
		}
	}
	if params == nil || text(params, "sageVersion") != "0.10.0" || !version(text(params, "recordVersion")) {
		return nil, ErrRejected
	}
	if _, e := sagedid.ParseDID010(text(params, "did")); e != nil {
		return nil, ErrRejected
	}
	issued, a := integer(params["issued"])
	expires, b := integer(params["expires"])
	if !a || !b || expires <= issued || expires-issued > 86400 {
		return nil, ErrRejected
	}
	return params, nil
}

type signature struct {
	protected, alg, kid string
	value               []byte
}

func selectSignature(m map[string]any) (signature, error) {
	entries, _ := m["signatures"].([]any)
	var chosen signature
	found := false
	for _, entry := range entries {
		s := entry.(map[string]any)
		raw, ok := binary(text(s, "protected"))
		if !ok {
			continue
		}
		h, e := strictObject(raw, 262144)
		if e != nil {
			var hint map[string]any
			if json.Unmarshal(raw, &hint) == nil && text(hint, "typ") == CardType {
				return chosen, ErrRejected
			}
			continue
		}
		if text(h, "typ") != CardType {
			continue
		}
		_, unprotected := s["header"]
		if found || unprotected || len(h) != 3 || (text(h, "alg") != "Ed25519" && text(h, "alg") != "ES256") {
			return chosen, ErrRejected
		}
		for _, name := range []string{"alg", "kid", "typ"} {
			if _, ok := h[name].(string); !ok {
				return chosen, ErrRejected
			}
		}
		value, ok := binary(text(s, "signature"))
		if !ok || len(value) != 64 {
			return chosen, ErrRejected
		}
		if _, e := sagedid.ParseDIDURL010(text(h, "kid")); e != nil {
			return chosen, ErrRejected
		}
		chosen = signature{text(s, "protected"), text(h, "alg"), text(h, "kid"), value}
		found = true
	}
	if !found {
		return chosen, ErrRejected
	}
	return chosen, nil
}

// Verify validates shape before observing the configured Source, checks one
// exact signing key and record version, and samples time again after crypto.
// Descriptions and skills in the returned card are data, never instructions.
func (v *Verifier) Verify(ctx context.Context, raw []byte) (*VerifiedCard, error) {
	if v == nil || ctx == nil || ctx.Err() != nil {
		return nil, ErrRejected
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	m, e := strictObject(raw, 262144)
	if e != nil {
		return nil, ErrRejected
	}
	m, e = normalize(m, "AgentCard")
	if e != nil {
		return nil, ErrRejected
	}
	params, e := declaration(m)
	if e != nil {
		return nil, ErrRejected
	}
	sig, e := selectSignature(m)
	if e != nil {
		return nil, ErrRejected
	}
	did := text(params, "did")
	key, e := sagedid.ParseDIDURL010(sig.kid)
	if e != nil || strings.SplitN(sig.kid, "#", 2)[0] != did || (v.cfg.ExpectedPeer != "" && did != v.cfg.ExpectedPeer) {
		return nil, ErrRejected
	}
	list, _ := m["supportedInterfaces"].([]any)
	if len(list) == 0 {
		return nil, ErrRejected
	}
	for _, x := range list {
		in := x.(map[string]any)
		if text(in, "protocolBinding") != BindingURI || text(in, "protocolVersion") != "1.0" || !https(text(in, "url")) {
			return nil, ErrRejected
		}
	}
	start, e := v.sample()
	if e != nil {
		return nil, ErrRejected
	}
	issued, _ := integer(params["issued"])
	expires, _ := integer(params["expires"])
	if issued > start.Unix || start.Unix >= expires {
		return nil, ErrRejected
	}
	d, e := didgo.Parse(did)
	if e != nil {
		return nil, ErrRejected
	}
	r, e := v.cfg.Source.Observe(ctx, d)
	if e != nil || ctx.Err() != nil || r.DID.String() != did || r.State != sage.StateActive || r.Version != text(params, "recordVersion") {
		return nil, ErrRejected
	}
	var signing *sage.Key
	for _, k := range r.Keys {
		if k.Name == key.KeyID {
			if signing != nil {
				return nil, ErrRejected
			}
			candidate := k
			signing = &candidate
		}
	}
	if signing == nil || signing.State != sage.KeyAccepted || (signing.Expires != nil && start.Unix >= *signing.Expires) {
		return nil, ErrRejected
	}
	eligible := []Interface{}
	for _, x := range list {
		in := x.(map[string]any)
		for _, service := range r.Services {
			if service.Type == "A2AInterface" && service.URI == text(in, "url") {
				eligible = append(eligible, Interface{service.URI, text(in, "tenant")})
				break
			}
		}
	}
	if len(eligible) == 0 {
		return nil, ErrRejected
	}
	// Exclude only top-level signatures; nested names remain signed.
	signedCard := map[string]any{}
	for name, x := range m {
		if name != "signatures" {
			signedCard[name] = x
		}
	}
	body, e := canonical(signedCard)
	if e != nil {
		return nil, ErrRejected
	}
	input := []byte(sig.protected + "." + base64.RawURLEncoding.EncodeToString(body))
	if !verifySignature(sig.alg, *signing, input, sig.value) {
		return nil, ErrRejected
	}
	end, e := v.sample()
	if e != nil || end.MonoMS-start.MonoMS > 5000 || end.Unix >= expires || issued > end.Unix || r.Observation.Time < 0 || r.Observation.Time > end.Unix || r.Observation.Time < end.Unix-5 || (signing.Expires != nil && end.Unix >= *signing.Expires) || ctx.Err() != nil {
		return nil, ErrRejected
	}
	data, e := canonical(m)
	if e != nil || len(data) > 262144 {
		return nil, ErrRejected
	}
	required := []string{}
	caps := m["capabilities"].(map[string]any)
	extensions, _ := caps["extensions"].([]any)
	for _, entry := range extensions {
		ext := entry.(map[string]any)
		if ext["required"] == true && text(ext, "uri") != ExtensionURI {
			required = append(required, text(ext, "uri"))
		}
	}
	bounds := cardBounds{issued: issued, expires: expires}
	if signing.Expires != nil {
		bounds.keyHasExpiry = true
		bounds.keyExpires = *signing.Expires
	}
	return &VerifiedCard{raw: data, did: did, version: r.Version, interfaces: eligible, otherRequired: required, bounds: bounds}, nil
}

func parsePeer(s string) (sagedid.DID010, error) { return sagedid.ParseDID010(s) }
