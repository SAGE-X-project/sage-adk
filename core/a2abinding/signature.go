// SPDX-License-Identifier: LGPL-3.0-or-later
package a2abinding

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"filippo.io/edwards25519"
	"github.com/sage-x-project/did-go/advanced/sage"
	"math/big"
)

func prime(encoded []byte) bool {
	if len(encoded) != 32 {
		return false
	}
	p, e := new(edwards25519.Point).SetBytes(encoded)
	if e != nil || !bytes.Equal(p.Bytes(), encoded) || p.Equal(edwards25519.NewIdentityPoint()) == 1 {
		return false
	}
	order, _ := new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	acc := edwards25519.NewIdentityPoint()
	for i := order.BitLen() - 1; i >= 0; i-- {
		acc.Add(acc, acc)
		if order.Bit(i) != 0 {
			acc.Add(acc, p)
		}
	}
	return acc.Equal(edwards25519.NewIdentityPoint()) == 1
}
func verifySignature(alg string, k sage.Key, input, sig []byte) bool {
	if len(sig) != 64 {
		return false
	}
	switch {
	case alg == "Ed25519" && k.Alg == sage.AlgEd25519:
		if !prime(k.Material) || !prime(sig[:32]) {
			return false
		}
		if _, e := new(edwards25519.Scalar).SetCanonicalBytes(sig[32:]); e != nil {
			return false
		}
		return ed25519.Verify(ed25519.PublicKey(k.Material), input, sig)
	case alg == "ES256" && k.Alg == sage.AlgP256:
		p, e := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), k.Material)
		if e != nil {
			return false
		}
		r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
		n := elliptic.P256().Params().N
		if r.Sign() <= 0 || r.Cmp(n) >= 0 || s.Sign() <= 0 || s.Cmp(new(big.Int).Rsh(new(big.Int).Set(n), 1)) > 0 {
			return false
		}
		h := sha256.Sum256(input)
		return ecdsa.Verify(p, h[:], r, s)
	default:
		return false
	}
}
