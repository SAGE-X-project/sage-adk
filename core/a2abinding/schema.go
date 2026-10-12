// SPDX-License-Identifier: LGPL-3.0-or-later
package a2abinding

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Generated field table from A2A v1.0.1 a2a.proto, SHA-256 e195bf96ab630c69797851970203e1b2b6b19528f2e9803b7d904b91a5104016.
//
//go:embed schema.json
var schemaBytes []byte
var schema = loadSchema()

type field struct {
	Name     string `json:"name"`
	JSON     string `json:"json"`
	Type     string `json:"type"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
	Optional bool   `json:"optional"`
	Oneof    string `json:"oneof"`
}

func loadSchema() map[string][]field {
	var m map[string][]field
	if json.Unmarshal(schemaBytes, &m) != nil {
		panic("invalid embedded A2A schema")
	}
	return m
}
func normalize(m map[string]any, typ string) (map[string]any, error) {
	out := map[string]any{}
	choices := map[string]bool{}
	seen := map[string]bool{}
	for name, v := range m {
		var f *field
		for _, candidate := range schema[typ] {
			if candidate.Name == name || candidate.JSON == name {
				c := candidate
				f = &c
				break
			}
		}
		if f == nil {
			return nil, ErrRejected
		}
		if seen[f.JSON] {
			return nil, ErrRejected
		}
		seen[f.JSON] = true
		// ProtoJSON null leaves known fields unset; Value retains a null datum.
		if v == nil && f.Type != "google.protobuf.Value" {
			continue
		}
		if f.Oneof != "" {
			if choices[f.Oneof] {
				return nil, ErrRejected
			}
			choices[f.Oneof] = true
		}
		result, e := normalizeField(v, *f)
		if e != nil {
			return nil, e
		}
		out[f.JSON] = result
	}
	for _, f := range schema[typ] {
		v, exists := out[f.JSON]
		if f.Required && (!exists || (f.Kind == "repeated" && len(v.([]any)) == 0)) {
			return nil, ErrRejected
		}
		if exists && !f.Required && !f.Optional && f.Oneof == "" && isDefault(v, f) {
			delete(out, f.JSON)
		}
	}
	return out, nil
}
func normalizeField(v any, f field) (any, error) {
	switch f.Kind {
	case "repeated":
		a, ok := v.([]any)
		if !ok {
			return nil, ErrRejected
		}
		out := []any{}
		for _, value := range a {
			x, e := normalizeScalar(value, f.Type)
			if e != nil {
				return nil, e
			}
			out = append(out, x)
		}
		return out, nil
	case "map":
		m, ok := v.(map[string]any)
		if !ok {
			return nil, ErrRejected
		}
		out := map[string]any{}
		for k, value := range m {
			x, e := normalizeScalar(value, f.Type)
			if e != nil {
				return nil, e
			}
			out[k] = x
		}
		return out, nil
	default:
		return normalizeScalar(v, f.Type)
	}
}
func normalizeScalar(v any, typ string) (any, error) {
	switch typ {
	case "bytes":
		if s, ok := v.(string); ok {
			for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
				if _, e := encoding.DecodeString(s); e == nil {
					return v, nil
				}
			}
		}
	case "string":
		if _, ok := v.(string); ok {
			return v, nil
		}
	case "bool":
		if _, ok := v.(bool); ok {
			return v, nil
		}
	case "int32":
		input := v
		if text, ok := v.(string); ok {
			raw, _ := json.Marshal(text)
			var wrapper wrapperspb.Int32Value
			if protojson.Unmarshal(raw, &wrapper) != nil {
				return nil, ErrRejected
			}
			input = json.Number(text)
			// A quoted zero is not a negative-zero numeric token. Preserve the
			// ProtoJSON semantic zero, after the standard parser accepts it.
			mantissa, _, _ := strings.Cut(strings.ToLower(strings.TrimPrefix(text, "-")), "e")
			if wrapper.Value == 0 && strings.Trim(mantissa, "0.") == "" {
				input = json.Number("0")
			}
		}
		i, ok := integerSigned(input)
		if ok && i >= -2147483648 && i <= 2147483647 {
			return json.Number(strconv.FormatInt(i, 10)), nil
		}
	case "Role":
		if x, ok := v.(string); ok && (x == "ROLE_UNSPECIFIED" || x == "ROLE_USER" || x == "ROLE_AGENT") {
			return v, nil
		}
		if i, ok := integerSigned(v); ok && i >= 0 && i <= 2 {
			return v, nil
		}
	case "TaskState":
		names := []string{"TASK_STATE_UNSPECIFIED", "TASK_STATE_SUBMITTED", "TASK_STATE_WORKING", "TASK_STATE_COMPLETED", "TASK_STATE_FAILED", "TASK_STATE_CANCELED", "TASK_STATE_INPUT_REQUIRED", "TASK_STATE_REJECTED", "TASK_STATE_AUTH_REQUIRED"}
		if x, ok := v.(string); ok {
			for _, name := range names {
				if name == x {
					return v, nil
				}
			}
		}
		if i, ok := integerSigned(v); ok && i >= 0 && i <= 8 {
			return v, nil
		}
	case "google.protobuf.Timestamp":
		if x, ok := v.(string); ok && strings.HasSuffix(x, "Z") && timestampPattern.MatchString(x) {
			t, e := time.Parse(time.RFC3339Nano, x)
			if e == nil && t.Year() >= 1 {
				return v, nil
			}
		}
	case "google.protobuf.Value":
		return v, nil
	case "google.protobuf.Struct":
		if _, ok := v.(map[string]any); ok {
			return v, nil
		}
	default:
		if m, ok := v.(map[string]any); ok {
			return normalize(m, typ)
		}
	}
	return nil, ErrRejected
}
func isDefault(v any, f field) bool {
	switch x := v.(type) {
	case string:
		return x == ""
	case bool:
		return !x
	case []any:
		return len(x) == 0
	case map[string]any:
		return f.Kind == "map" && len(x) == 0
	}
	return false
}

var timestampPattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$`)
