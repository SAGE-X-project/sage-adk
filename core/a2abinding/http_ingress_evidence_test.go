package a2abinding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	didgo "github.com/sage-x-project/did-go/did"
)

type ingressCalibration struct {
	Scope            string `json:"scope"`
	ParserError      string `json:"parser_error"`
	BodyHex          string `json:"body_hex"`
	OpenError        string `json:"open_error"`
	ReturnedRequest  bool   `json:"returned_request"`
	AcceptedRequests int    `json:"accepted_requests"`
}
type ingressMemoryObservation struct {
	Case                  string              `json:"case"`
	Transport             string              `json:"transport"`
	Target                string              `json:"target"`
	ServerDID             string              `json:"server_did"`
	LocalDID              string              `json:"local_did"`
	PeerDID               string              `json:"peer_did"`
	SourceDID             string              `json:"source_did"`
	SourceVersion         string              `json:"source_version"`
	SourceService         string              `json:"source_service"`
	TLSName               string              `json:"tls_name"`
	Offered               []string            `json:"offered"`
	Negotiated            string              `json:"negotiated"`
	HandshakeFailed       bool                `json:"handshake_failed"`
	HandshakeError        string              `json:"handshake_error"`
	InputHex              string              `json:"input_hex"`
	InputSHA              string              `json:"input_sha256"`
	Written               int                 `json:"written"`
	ResponseFrameHex      string              `json:"response_frame_hex"`
	ResponseStatus        int                 `json:"response_status"`
	ResponseBodyHex       string              `json:"response_body_hex"`
	AuthenticatedResponse bool                `json:"authenticated_response"`
	ResponseError         string              `json:"response_error"`
	VerifiedPayloadHex    string              `json:"verified_payload_hex"`
	VerifiedSuccess       bool                `json:"verified_success"`
	VerifiedCode          string              `json:"verified_code"`
	HandlerCalls          int                 `json:"handler_calls"`
	HandlerPayloads       []string            `json:"handler_payloads_hex"`
	AcceptedRequests      int                 `json:"accepted_requests"`
	AcceptedResponses     int                 `json:"accepted_responses"`
	ParserTrace           string              `json:"parser_trace"`
	CoreTrace             string              `json:"core_trace"`
	Calibration           *ingressCalibration `json:"calibration"`
}

func ingressUnsignedFrame(mode string) []byte {
	// Fixed inert unit data only: no socket listener, packet runner or input CLI.
	body := `{"jsonrpc":"2.0","id":"one","method":"SendMessage","params":{"message":{"messageId":"memory","role":"ROLE_USER","parts":[{"text":"inert"}]}}}`
	media, extra := "application/json", "A2A-Version: 1.0\r\nA2A-Extensions: "+ExtensionURI+"\r\n"
	if mode == "legacy-unsigned" {
		extra = ""
	}
	if mode == "grpc-media-http1" {
		media = "application/grpc"
		body = "inert unsigned unit bytes"
	}
	return []byte(fmt.Sprintf("POST /a2a HTTP/1.1\r\nHost: agents.example.com\r\nContent-Type: %s\r\nContent-Length: %d\r\nConnection: close\r\n%s\r\n%s", media, len(body), extra, body))
}
func ingressReplayCount(r *protectedReplay) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.accepted
}
func ingressDirectCalibration(t *testing.T, raw []byte) *ingressCalibration {
	t.Helper()
	pair := protectedSessions(t, endpoint, true)
	owner, e := pair.right.TakeHTTPApplication(endpoint)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(owner.Close)
	target, e := protectedTarget(endpoint)
	if e != nil {
		t.Fatal(e)
	}
	message, parseErr := protectedReadRequest(bytes.NewReader(raw), &protectedHTTPState{target: target, exact: endpoint})
	out := &ingressCalibration{Scope: "separate-fresh-memory-fixture"}
	if parseErr != nil {
		out.ParserError = parseErr.Error()
	} else {
		out.BodyHex = hex.EncodeToString(message.Body)
		value, openErr := owner.OpenRequest(context.Background(), message)
		out.ReturnedRequest = value != nil
		if openErr != nil {
			out.OpenError = openErr.Error()
		}
	}
	out.AcceptedRequests = ingressReplayCount(pair.rightReplay)
	if out.ParserError != "" || out.BodyHex == "" || out.OpenError != "authentication failed" || out.ReturnedRequest || out.AcceptedRequests != 0 {
		t.Fatal("standalone parser/core calibration", out)
	}
	return out
}

// All refused inputs remain within in-memory TLS pipes. Only the separate
// existing TestProtectedHTTPRuntime010 performs normal loopback network calls.
func TestProtectedHTTPIngressRefusalEvidence010(t *testing.T) {
	for _, mode := range []string{"positive-calibration", "bare-jsonrpc", "legacy-unsigned", "grpc-media-http1", "h2-only"} {
		t.Run(mode, func(t *testing.T) {
			pair := protectedSessions(t, endpoint, true)
			left, e := pair.left.TakeHTTPApplication(endpoint)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(left.Close)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			card := pair.card(t)
			var effects atomic.Int32
			payloads := []string{}
			server, e := NewServer(ServerConfig{DID: peer, URL: endpoint, Verifier: pair.verifier(t), Card: func(context.Context) ([]byte, error) { return card, nil }, Handler: handlerFunc(func(_ context.Context, p []byte) ([]byte, error) {
				effects.Add(1)
				payloads = append(payloads, hex.EncodeToString(p))
				return reply(), nil
			})})
			if e != nil {
				t.Fatal(e)
			}
			cert, roots := protectedCertificate(t)
			ingress, e := NewHTTPIngress(HTTPIngressConfig{Transport: HTTPTransportConfig{Session: pair.right, URL: endpoint, TLS: &tls.Config{Certificates: []tls.Certificate{cert}}, TTL: 30, Timeout: 3 * time.Second}, Server: server})
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(ingress.Close)
			payload := rpc("GetTask", map[string]any{"id": "task"})
			// Establish an actual pending response context even for the unsigned
			// failure controls, so lack of a pending request is not the oracle.
			message, e := left.SealRequest(ctx, payload, metadata(), 30)
			if e != nil {
				t.Fatal(e)
			}
			frame, e := protectedRequestFrame(ctx, message)
			if e != nil {
				t.Fatal(e)
			}
			if mode != "positive-calibration" && mode != "h2-only" {
				frame = ingressUnsignedFrame(mode)
			}
			if mode == "h2-only" {
				frame = nil
			}
			config := &tls.Config{RootCAs: roots, ServerName: "localhost", MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, NextProtos: []string{"http/1.1"}}
			if mode == "h2-only" {
				config.NextProtos = []string{"h2"}
			}
			out := ingressMemoryObservation{Case: mode, Transport: "net.Pipe/TLS", Target: ingress.state.transport.exact, ServerDID: server.cfg.DID, LocalDID: ingress.state.transport.local, PeerDID: ingress.state.transport.peer, TLSName: config.ServerName, Offered: append([]string(nil), config.NextProtos...), InputHex: hex.EncodeToString(frame), HandlerPayloads: []string{}, ParserTrace: "NOT_OBSERVED", CoreTrace: "NOT_OBSERVED"}
			inputDigest := sha256.Sum256(frame)
			out.InputSHA = hex.EncodeToString(inputDigest[:])
			record, e := pair.fixture.Observe(ctx, mustIngressDID(t, peer))
			if e != nil {
				t.Fatal(e)
			}
			out.SourceDID = record.DID.String()
			out.SourceVersion = record.Version
			for _, service := range record.Services {
				if service.Type == "A2AInterface" {
					out.SourceService = service.URI
				}
			}
			clientRaw, serverRaw := net.Pipe()
			t.Cleanup(func() { _ = clientRaw.Close(); _ = serverRaw.Close() })
			done := make(chan struct{})
			go func() { defer close(done); ingress.state.connection(ctx, serverRaw) }()
			if e = clientRaw.SetDeadline(time.Now().Add(3 * time.Second)); e != nil {
				t.Fatal(e)
			}
			client := tls.Client(clientRaw, config)
			handshakeErr := client.HandshakeContext(ctx)
			out.HandshakeFailed = handshakeErr != nil
			if handshakeErr != nil {
				out.HandshakeError = handshakeErr.Error()
			}
			out.Negotiated = client.ConnectionState().NegotiatedProtocol
			if mode == "h2-only" {
				if handshakeErr == nil || handshakeErr.Error() != "remote error: tls: no application protocol" || ctx.Err() != nil {
					t.Fatal("expected live-context no-application-protocol refusal", handshakeErr, ctx.Err())
				}
				out.ParserTrace = "NOT_REACHED"
				out.CoreTrace = "NOT_REACHED"
			} else {
				if handshakeErr != nil {
					t.Fatal(handshakeErr)
				}
				n, writeErr := client.Write(frame)
				out.Written = n
				if writeErr != nil || n != len(frame) {
					t.Fatal("in-memory request write", n, writeErr)
				}
				var captured bytes.Buffer
				response, readErr := protectedReadResponse(io.TeeReader(client, &captured))
				if readErr != nil {
					t.Fatal("in-memory response frame", readErr)
				}
				out.ResponseFrameHex = hex.EncodeToString(captured.Bytes())
				out.ResponseStatus = response.Status
				out.ResponseBodyHex = hex.EncodeToString(response.Body)
				if ctx.Err() != nil {
					t.Fatal("expired unit context before response admission", ctx.Err())
				}
				verified, openErr := left.OpenResponse(ctx, response)
				if ctx.Err() != nil {
					t.Fatal("expired unit context during response admission", ctx.Err())
				}
				out.AuthenticatedResponse = verified != nil && openErr == nil
				if verified != nil {
					out.VerifiedPayloadHex = hex.EncodeToString(verified.Payload())
					out.VerifiedSuccess = verified.Success()
					out.VerifiedCode = verified.Error()
				}
				if openErr != nil {
					out.ResponseError = openErr.Error()
				}
				if mode == "positive-calibration" {
					if !out.AuthenticatedResponse || !verified.Success() || !bytes.Equal(verified.Payload(), reply()) || response.Status != 200 {
						t.Fatal("positive memory ingress calibration", openErr)
					}
				} else if out.AuthenticatedResponse || verified != nil || openErr == nil || response.Status != 401 || string(response.Body) != "authentication failed\n" {
					t.Fatal("unsigned failure became a usable response", openErr, response.Status)
				}
			}
			select {
			case <-done:
			case <-ctx.Done():
				t.Fatal("in-memory ingress did not finish")
			}
			if ctx.Err() != nil {
				t.Fatal("expired unit context at witness boundary", ctx.Err())
			}
			out.HandlerCalls = int(effects.Load())
			out.HandlerPayloads = append(out.HandlerPayloads, payloads...)
			out.AcceptedRequests = ingressReplayCount(pair.rightReplay)
			out.AcceptedResponses = ingressReplayCount(pair.leftReplay)
			if mode != "positive-calibration" && mode != "h2-only" {
				out.Calibration = ingressDirectCalibration(t, frame)
			}
			if out.ServerDID != peer || out.LocalDID != peer || out.PeerDID != protectedAlice || out.Target != endpoint || out.SourceDID != peer || out.SourceVersion != "3" || out.SourceService != endpoint {
				t.Fatal("configured authority/identity binding changed")
			}
			if mode == "positive-calibration" {
				if out.HandlerCalls != 1 || out.AcceptedRequests != 1 || out.AcceptedResponses != 1 || len(out.HandlerPayloads) != 1 || out.HandlerPayloads[0] != hex.EncodeToString(payload) {
					t.Fatal("positive memory phases", out)
				}
			} else if out.HandlerCalls != 0 || out.AcceptedRequests != 0 || out.AcceptedResponses != 0 || len(out.HandlerPayloads) != 0 {
				t.Fatal("refusal reached effects or acceptance", out)
			}
			raw, e := json.Marshal(out)
			if e != nil {
				t.Fatal(e)
			}
			t.Log("INGRESS_MEMORY_OBSERVATION " + string(raw))
		})
	}
}
func mustIngressDID(t *testing.T, text string) didgo.DID {
	t.Helper()
	did, e := didgo.Parse(text)
	if e != nil {
		t.Fatal(e)
	}
	return did
}
