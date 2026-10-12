// Copyright (C) 2025 SAGE-X-project. UTF-16 escape checks adapted from guard010/json.go.
// SPDX-License-Identifier: LGPL-3.0-or-later
package a2abinding

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

func strictObject(raw []byte, limit int) (map[string]any, error) {
	return strictObjectPolicy(raw, limit, true)
}
func strictObjectPolicy(raw []byte, limit int, denyNull bool) (map[string]any, error) {
	if len(raw) > limit || !utf8.Valid(raw) || bytes.HasPrefix(raw, []byte{0xef, 0xbb, 0xbf}) || !validEscapes(raw) {
		return nil, ErrRejected
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	v, e := readValue(d, 0, denyNull)
	if e != nil {
		return nil, ErrRejected
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, ErrRejected
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, ErrRejected
	}
	return m, nil
}
func readValue(d *json.Decoder, depth int, denyNull bool) (any, error) {
	t, e := d.Token()
	if e != nil {
		return nil, ErrRejected
	}
	switch x := t.(type) {
	case nil:
		if denyNull {
			return nil, ErrRejected
		}
		return nil, nil
	case json.Delim:
		if depth >= 64 {
			return nil, ErrRejected
		}
		switch x {
		case '{':
			m := map[string]any{}
			for d.More() {
				token, e := d.Token()
				if e != nil {
					return nil, ErrRejected
				}
				name, ok := token.(string)
				if !ok {
					return nil, ErrRejected
				}
				if _, exists := m[name]; exists {
					return nil, ErrRejected
				}
				v, e := readValue(d, depth+1, denyNull)
				if e != nil {
					return nil, e
				}
				m[name] = v
			}
			t, e = d.Token()
			if e != nil || t != json.Delim('}') {
				return nil, ErrRejected
			}
			return m, nil
		case '[':
			a := []any{}
			for d.More() {
				v, e := readValue(d, depth+1, denyNull)
				if e != nil {
					return nil, e
				}
				a = append(a, v)
			}
			t, e = d.Token()
			if e != nil || t != json.Delim(']') {
				return nil, ErrRejected
			}
			return a, nil
		default:
			return nil, ErrRejected
		}
	case json.Number:
		f, e := strconv.ParseFloat(string(x), 64)
		if e != nil || math.IsInf(f, 0) || math.IsNaN(f) || (f == 0 && math.Signbit(f)) {
			return nil, ErrRejected
		}
	}
	return t, nil
}
func text(m map[string]any, name string) string { s, _ := m[name].(string); return s }
func integer(v any) (int64, bool)               { i, ok := integerSigned(v); return i, ok && i >= 0 }
func validEscapes(b []byte) bool {
	inside := false
	for i := 0; i < len(b); i++ {
		if b[i] == '"' {
			inside = !inside
			continue
		}
		if !inside || b[i] != '\\' {
			continue
		}
		i++
		if i >= len(b) {
			return false
		}
		if b[i] != 'u' {
			continue
		}
		if i+4 >= len(b) {
			return false
		}
		n, e := strconv.ParseUint(string(b[i+1:i+5]), 16, 16)
		if e != nil {
			return false
		}
		i += 4
		if n >= 0xDC00 && n <= 0xDFFF {
			return false
		}
		if n < 0xD800 || n > 0xDBFF {
			continue
		}
		if i+6 >= len(b) || b[i+1] != '\\' || b[i+2] != 'u' {
			return false
		}
		v, e := strconv.ParseUint(string(b[i+3:i+7]), 16, 16)
		if e != nil || v < 0xDC00 || v > 0xDFFF {
			return false
		}
		i += 6
	}
	return true
}

// Protocol integers are checked in decimal digits without constructing powers
// from peer-supplied exponents. Work and allocation stay bounded by input size.
func integerSigned(v any) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	s := string(n)
	negative := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	mantissa, expText, hasExp := strings.Cut(strings.ToLower(s), "e")
	whole, fraction, hasFraction := strings.Cut(mantissa, ".")
	if !hasFraction {
		fraction = ""
	}
	digits := strings.TrimLeft(whole+fraction, "0")
	if digits == "" {
		return 0, !negative
	}
	exponent := int64(0)
	if hasExp {
		var e error
		exponent, e = strconv.ParseInt(expText, 10, 64)
		if e != nil {
			return 0, false
		}
	}
	if exponent < -int64(len(digits))-int64(len(fraction)) || exponent > int64(len(fraction))+16 {
		return 0, false
	}
	scale := exponent - int64(len(fraction))
	if scale < 0 {
		for scale < 0 && strings.HasSuffix(digits, "0") {
			digits = strings.TrimSuffix(digits, "0")
			scale++
		}
		if scale < 0 {
			return 0, false
		}
	}
	if int64(len(digits))+scale > 16 {
		return 0, false
	}
	digits += strings.Repeat("0", int(scale))
	if negative {
		digits = "-" + digits
	}
	i, e := strconv.ParseInt(digits, 10, 64)
	return i, e == nil && i >= -9007199254740991 && i <= 9007199254740991
}
