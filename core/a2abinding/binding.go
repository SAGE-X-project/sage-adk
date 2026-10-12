// SPDX-License-Identifier: LGPL-3.0-or-later
package a2abinding

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"unicode"
)

// ProtectedCall is an owned request for a locally configured transport owner.
// Its bytes are unchanged JSON-RPC payload; metadata belongs in the signed
// chapter 08 envelope, never HTTP headers or query parameters.
type ProtectedCall struct {
	target    Interface
	recipient string
	payload   []byte
	metadata  map[string]string
}

// Target returns the exact authenticated eligible HTTPS interface.
func (c ProtectedCall) Target() Interface { return c.target }

// Recipient returns the DID that the request envelope must name.
func (c ProtectedCall) Recipient() string { return c.recipient }

// Payload returns an owned copy; the owner must preserve the exact bytes.
func (c ProtectedCall) Payload() []byte { return append([]byte(nil), c.payload...) }

// Metadata returns an owned copy of the signed envelope service parameters.
func (c ProtectedCall) Metadata() map[string]string {
	out := map[string]string{}
	for k, v := range c.metadata {
		out[k] = v
	}
	return out
}

// AuthenticatedResponse comes only from the trusted transport owner after
// both signatures, exact request hash/IDs, recipient, timing, replay and AEAD.
// Peer values never establish these facts. A transport error returns no view.
type AuthenticatedResponse struct {
	Sender  string
	Payload []byte
}

// TransportOwner is mandatory locally trusted chapter 03/08 carriage. It must
// use HTTPS POST, both signatures, replay durability and exact response binding,
// never retry or downgrade, and call finalCheck once at its final send boundary
// after all blocking setup. The caller's ownership boundary must enforce this
// contract; implementing this interface alone does not establish protection.
type TransportOwner interface {
	Exchange(context.Context, ProtectedCall, func(context.Context) error) (AuthenticatedResponse, error)
}

// Client requires a verifier and a protected owner; there is no legacy selector.
type Client struct {
	verifier *Verifier
	owner    TransportOwner
}

// NewClient enables only the explicitly protected path.
func NewClient(v *Verifier, owner TransportOwner) (*Client, error) {
	if v == nil || owner == nil {
		return nil, ErrRejected
	}
	return &Client{v, owner}, nil
}

// Call checks a card freshly, selects its first eligible interface, preserves
// exact payload and checks the authenticated response. Failure is terminal.
func (c *Client) Call(ctx context.Context, card, rpc []byte) ([]byte, error) {
	if c == nil || ctx == nil || ctx.Err() != nil {
		return nil, ErrRejected
	}
	card = append([]byte(nil), card...)
	rpc = append([]byte(nil), rpc...)
	request, e := parseRPC(rpc, false)
	if e != nil {
		return nil, ErrRejected
	}
	verified, e := c.verifier.Verify(ctx, card)
	if e != nil {
		return nil, ErrRejected
	}
	if len(verified.otherRequired) > 0 {
		return nil, ErrRejected
	}
	selected := verified.interfaces[0]
	tenant, tenantPresent := tenantOf(request)
	if (selected.Tenant == "" && tenantPresent) || (selected.Tenant != "" && (!tenantPresent || tenant != selected.Tenant)) {
		return nil, ErrRejected
	}
	call := ProtectedCall{selected, verified.did, rpc, map[string]string{"a2a_version": "1.0", "a2a_extensions": ExtensionURI}}
	var attempted, admitted, invalid atomic.Bool
	finalCheck := func(checkCtx context.Context) error {
		if checkCtx == nil || checkCtx.Err() != nil || !attempted.CompareAndSwap(false, true) {
			invalid.Store(true)
			return ErrRejected
		}
		current, e := c.verifier.Verify(ctx, card)
		if e != nil || current.did != verified.did || current.version != verified.version {
			return ErrRejected
		}
		found := false
		for _, in := range current.interfaces {
			found = found || in == selected
		}
		if !found {
			return ErrRejected
		}
		if checkCtx.Err() != nil {
			invalid.Store(true)
			return ErrRejected
		}
		admitted.Store(true)
		return nil
	}
	response, e := c.owner.Exchange(ctx, call, finalCheck)
	if e != nil || !admitted.Load() || invalid.Load() || ctx.Err() != nil || response.Sender != verified.did {
		return nil, ErrRejected
	}
	parsed, e := parseRPC(response.Payload, true)
	if e != nil || !sameID(request["id"], parsed["id"]) {
		return nil, ErrRejected
	}
	return append([]byte(nil), response.Payload...), nil
}

// AuthenticatedRequest is the trusted owner's decrypted, authenticated request
// view. The owner must acquire crypto/replay acceptance before this boundary;
// it must derive URL, recipient and metadata from authenticated carriage/body.
type AuthenticatedRequest struct {
	URL, Recipient string
	Metadata       map[string]string
	Payload        []byte
}

// Handler processes one accepted A2A request and returns one JSON-RPC result.
// Effects still require the host's Execution Guard and local authorization.
type Handler interface {
	Handle(context.Context, []byte) ([]byte, error)
}

// TenantRouter is trusted routing configuration, never a value asserted by peers.
type TenantRouter func(context.Context, string) (string, error)

// ServerConfig binds one local agent, card, interface and application handler.
// Card returns a locally published card afresh. No request can change this choice.
type ServerConfig struct {
	DID, URL string
	Verifier *Verifier
	Card     func(context.Context) ([]byte, error)
	Tenant   TenantRouter
	Handler  Handler
}

// Server checks binding policy after its owner's cryptographic acceptance.
type Server struct{ cfg ServerConfig }

// NewServer does not create a second unsigned listener or select a transport.
func NewServer(c ServerConfig) (*Server, error) {
	if c.Verifier == nil || c.Card == nil || c.Handler == nil || !https(c.URL) {
		return nil, ErrRejected
	}
	if _, e := parsePeer(c.DID); e != nil {
		return nil, ErrRejected
	}
	return &Server{c}, nil
}

// Handle is only for a locally trusted authenticated transport owner. Public
// endpoints must hide ErrRejected behind the generic authentication failure.
// A2A unsupported methods return an error object inside a successful protected
// transport response, not a SAGE transport error or application dispatch.
func (s *Server) Handle(ctx context.Context, in AuthenticatedRequest) ([]byte, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || in.Recipient != s.cfg.DID || in.URL != s.cfg.URL || !validMetadata(in.Metadata) {
		return nil, ErrRejected
	}
	payload := append([]byte(nil), in.Payload...)
	request, e := parseRPC(payload, false)
	if e != nil {
		return nil, ErrRejected
	}
	raw, e := s.cfg.Card(ctx)
	if e != nil {
		return nil, ErrRejected
	}
	card, e := s.cfg.Verifier.Verify(ctx, raw)
	if e != nil || card.did != s.cfg.DID {
		return nil, ErrRejected
	}

	found := false
	tenant, tenantPresent := tenantOf(request)
	for _, i := range card.interfaces {
		if i.URL == s.cfg.URL && ((i.Tenant == "" && !tenantPresent) || (i.Tenant != "" && tenantPresent && i.Tenant == tenant)) {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrRejected
	}
	if tenantPresent {
		if s.cfg.Tenant == nil {
			return nil, ErrRejected
		}
		did, e := s.cfg.Tenant(ctx, tenant)
		if e != nil || did != in.Recipient {
			return nil, ErrRejected
		}
	}
	for _, required := range card.otherRequired {
		active := false
		for _, uri := range strings.Split(in.Metadata["a2a_extensions"], ",") {
			active = active || uri == required
		}
		if !active {
			return rpcError(request["id"], -32008, "Extension support required")
		}
	}
	method := text(request, "method")
	switch method {
	case "SendStreamingMessage", "SubscribeToTask", "CreateTaskPushNotificationConfig", "GetTaskPushNotificationConfig", "ListTaskPushNotificationConfigs", "DeleteTaskPushNotificationConfig", "GetExtendedAgentCard":
		return rpcError(request["id"], -32004, "Unsupported operation")
	}
	params, _ := request["params"].(map[string]any)
	typ := map[string]string{"SendMessage": "SendMessageRequest", "GetTask": "GetTaskRequest", "ListTasks": "ListTasksRequest", "CancelTask": "CancelTaskRequest"}[method]
	validatedParams, e := normalize(params, typ)
	if e != nil {
		return rpcError(request["id"], -32602, "Invalid params")
	}
	if method == "SendMessage" {
		conf, _ := validatedParams["configuration"].(map[string]any)
		if _, present := conf["taskPushNotificationConfig"]; present {
			return rpcError(request["id"], -32003, "Push notification not supported")
		}
	}
	// Routing can block; revalidate card/record again immediately before dispatch.
	current, e := s.cfg.Verifier.Verify(ctx, raw)
	if e != nil || current.did != card.did || current.version != card.version || ctx.Err() != nil {
		return nil, ErrRejected
	}
	response, e := s.cfg.Handler.Handle(ctx, payload)
	if e != nil {
		return rpcError(request["id"], -32603, "Internal error")
	}
	parsed, e := parseRPC(response, true)
	if e != nil || !sameID(request["id"], parsed["id"]) {
		return rpcError(request["id"], -32603, "Internal error")
	}
	return append([]byte(nil), response...), nil
}

var metadataName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

func validMetadata(m map[string]string) bool {
	if len(m) > 32 || m["a2a_version"] != "1.0" {
		return false
	}
	for k, v := range m {
		if !metadataName.MatchString(k) || len(v) > 1024 {
			return false
		}
	}
	b, e := canonical(m)
	if e != nil || len(b) > 4096 {
		return false
	}
	list := m["a2a_extensions"]
	if list == "" || strings.IndexFunc(list, unicode.IsSpace) >= 0 {
		return false
	}
	count := 0
	for _, uri := range strings.Split(list, ",") {
		u, e := url.Parse(uri)
		if e != nil || u.Scheme == "" {
			return false
		}
		if uri == ExtensionURI {
			count++
		}
	}
	return count == 1
}
func tenantOf(m map[string]any) (string, bool) {
	p, _ := m["params"].(map[string]any)
	value, present := p["tenant"]
	s, _ := value.(string)
	return s, present
}

func parseRPC(raw []byte, response bool) (map[string]any, error) {
	m, e := strictObjectPolicy(raw, 8<<20, false)
	if e != nil || text(m, "jsonrpc") != "2.0" {
		return nil, ErrRejected
	}
	id, exists := m["id"]
	if !exists || id == nil {
		return nil, ErrRejected
	}
	switch x := id.(type) {
	case string:
	case json.Number:
		if _, ok := integerSigned(x); !ok {
			return nil, ErrRejected
		}
	default:
		return nil, ErrRejected
	}
	if response {
		if len(m) != 3 {
			return nil, ErrRejected
		}
		_, result := m["result"]
		errorValue, failure := m["error"]
		if result == failure {
			return nil, ErrRejected
		}
		if failure {
			err, ok := errorValue.(map[string]any)
			if !ok || (len(err) != 2 && len(err) != 3) {
				return nil, ErrRejected
			}
			if _, ok := integerSigned(err["code"]); !ok {
				return nil, ErrRejected
			}
			if _, ok := err["message"].(string); !ok {
				return nil, ErrRejected
			}
			for k := range err {
				if k != "code" && k != "message" && k != "data" {
					return nil, ErrRejected
				}
			}
		}
	} else {
		if len(m) < 3 || len(m) > 4 {
			return nil, ErrRejected
		}
		method := text(m, "method")
		if !knownMethod(method) {
			return nil, ErrRejected
		}
		for k := range m {
			if k != "jsonrpc" && k != "id" && k != "method" && k != "params" {
				return nil, ErrRejected
			}
		}
		if p, present := m["params"]; present {
			params, ok := p.(map[string]any)
			if !ok {
				return nil, ErrRejected
			}
			if tenant, exists := params["tenant"]; exists {
				if _, ok := tenant.(string); !ok {
					return nil, ErrRejected
				}
			}
		}
	}
	return m, nil
}
func knownMethod(s string) bool {
	switch s {
	case "SendMessage", "GetTask", "ListTasks", "CancelTask", "SendStreamingMessage", "SubscribeToTask", "CreateTaskPushNotificationConfig", "GetTaskPushNotificationConfig", "ListTaskPushNotificationConfigs", "DeleteTaskPushNotificationConfig", "GetExtendedAgentCard":
		return true
	}
	return false
}
func sameID(a, b any) bool {
	x, e := canonical(a)
	y, f := canonical(b)
	return e == nil && f == nil && bytes.Equal(x, y)
}
func rpcError(id any, code int, message string) ([]byte, error) {
	return canonical(map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}})
}
