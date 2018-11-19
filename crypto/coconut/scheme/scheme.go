// scheme.go - Coconut signature scheme
// Copyright (C) 2018  Jedrzej Stuczynski.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package coconut provides the functionalities required by the Coconut Scheme.
package coconut

import (
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jstuczyn/CoconutGo/constants"

	"github.com/jstuczyn/CoconutGo/crypto/bpgroup"
	"github.com/jstuczyn/CoconutGo/crypto/coconut/utils"
	"github.com/jstuczyn/CoconutGo/crypto/elgamal"
	"github.com/jstuczyn/amcl/version3/go/amcl"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
)

// todo: rename and restructure PolynomialPoints struct + all its uses
// todo: comments with maths computation
// todo: comments with python sources
// todo: remove ShowBlindSignature and move it straight to BlindVerify?
// todo: include gamma to blindsignmats as per paper rather than as per python implementation?

var (
	// ErrSetupParams indicates incorrect parameters provided for Setup.
	ErrSetupParams = errors.New("Can't generate params for less than 1 attribute")

	// ErrSignParams indicates inconsistent parameters provided for Sign.
	ErrSignParams = errors.New("Invalid attributes/secret key provided")

	// ErrKeygenParams indicates incorrect parameters provided for Keygen.
	ErrKeygenParams = errors.New("Can't generate keys for less than 1 attribute")

	// ErrTTPKeygenParams indicates incorrect parameters provided for TTPKeygen.
	ErrTTPKeygenParams = errors.New("Invalid set of parameters provided to keygen")

	// ErrPrepareBlindSignParams indicates that number of attributes to sign is larger than q specified in Setup.
	ErrPrepareBlindSignParams = errors.New("Too many attributes to sign")

	// ErrPrepareBlindSignPrivate indicates lack of private attributes to blindly sign.
	ErrPrepareBlindSignPrivate = errors.New("No private attributes to sign")

	// ErrBlindSignParams indicates that number of attributes to sign is larger than q specified in Setup.
	ErrBlindSignParams = errors.New("Too many attributes to sign")

	// ErrBlindSignProof indicates that proof of corectness of ciphertext and cm was invalid
	ErrBlindSignProof = errors.New("Failed to verify the proof")

	// ErrShowBlindAttr indicates that either there were no private attributes provided
	// or their number was larger than the verification key supports
	ErrShowBlindAttr = errors.New("Invalid attributes provided")
)

// Setup generates the public parameters required by the Coconut scheme.
// q indicates the maximum number of attributes that can be embed in the credentials.
func Setup(q int) (*Params, error) {
	if q < 1 {
		return nil, ErrSetupParams
	}
	hs := make([]*Curve.ECP, q)
	for i := 0; i < q; i++ {
		hi, err := utils.HashStringToG1(amcl.SHA512, fmt.Sprintf("h%d", i))
		if err != nil {
			panic(err)
		}
		hs[i] = hi
	}

	G := bpgroup.New()
	return &Params{
		G:  G,
		p:  G.Order(),
		g1: G.Gen1(),
		g2: G.Gen2(),
		hs: hs,
	}, nil
}

// Keygen generates a single Coconut keypair ((x, y1, y2...), (g2, g2^x, g2^y1, ...)).
// It is not suitable for threshold credentials as all generated keys are independent of each other.
func Keygen(params *Params) (*SecretKey, *VerificationKey, error) {
	p, g2, hs, rng := params.p, params.g2, params.hs, params.G.Rng()

	q := len(hs)
	if q < 1 {
		return nil, nil, ErrKeygenParams
	}
	x := Curve.Randomnum(p, rng)
	y := make([]*Curve.BIG, q)
	sk := &SecretKey{x: x, y: y}

	for i := 0; i < q; i++ {
		y[i] = Curve.Randomnum(p, rng)
	}

	alpha := Curve.G2mul(g2, x)
	beta := make([]*Curve.ECP2, q)
	vk := &VerificationKey{g2: g2, alpha: alpha, beta: beta}

	for i := 0; i < q; i++ {
		beta[i] = Curve.G2mul(g2, y[i])
	}
	return sk, vk, nil
}

// TTPKeygen generates a set of n Coconut keypairs [((x, y1, y2...), (g2, g2^x, g2^y1, ...)), ...],
// such that they support threshold aggregation of t parties.
// It is expected that this procedure is executed by a Trusted Third Party.
func TTPKeygen(params *Params, t int, n int) ([]*SecretKey, []*VerificationKey, error) {
	p, g2, hs, rng := params.p, params.g2, params.hs, params.G.Rng()

	q := len(hs)
	if n < t || t <= 0 || q <= 0 {
		return nil, nil, ErrTTPKeygenParams
	}

	// polynomials generation
	v := utils.GenerateRandomBIGSlice(p, rng, t)
	w := make([][]*Curve.BIG, q)
	for i := range w {
		w[i] = utils.GenerateRandomBIGSlice(p, rng, t)
	}

	// secret keys
	sks := make([]*SecretKey, n)
	// we can use any is now, rather than 1,2...,n; might be useful if we have some authorities ids?
	for i := 1; i < n+1; i++ {
		iBIG := Curve.NewBIGint(i)
		x := utils.PolyEval(v, iBIG, p)
		ys := make([]*Curve.BIG, q)
		for j, wj := range w {
			ys[j] = utils.PolyEval(wj, iBIG, p)
		}
		sks[i-1] = &SecretKey{x: x, y: ys}
	}

	// verification keys
	vks := make([]*VerificationKey, n)
	for i := range sks {
		alpha := Curve.G2mul(g2, sks[i].x)
		beta := make([]*Curve.ECP2, q)
		for j, yj := range sks[i].y {
			beta[j] = Curve.G2mul(g2, yj)
		}
		vks[i] = &VerificationKey{g2: g2, alpha: alpha, beta: beta}

	}
	return sks, vks, nil
}

// Sign creates a Coconut credential under a given secret key on a set of public attributes only.
func Sign(params *Params, sk *SecretKey, pubM []*Curve.BIG) (*Signature, error) {
	p := params.p

	// allow using longer keys for shorter messages
	if len(pubM) > len(sk.y) {
		return nil, ErrSignParams
	}

	h := getBaseFromAttributes(pubM)

	K := Curve.NewBIGcopy(sk.x) // K = x
	for i := 0; i < len(pubM); i++ {
		tmp := Curve.Modmul(sk.y[i], pubM[i], p) // (ai * yi)
		K = K.Plus(tmp)                          // K = x + (a0 * y0) + ...
	}
	sig := Curve.G1mul(h, K) // sig = h^(x + (a0 * y0) + ... )

	return &Signature{h, sig}, nil
}

// PrepareBlindSign builds cryptographic material for blind sign.
// It returns commitment to the private and public attributes,
// encryptions of the private attributes
// and zero-knowledge proof asserting corectness of the above.
// nolint: lll
func PrepareBlindSign(params *Params, egPub *elgamal.PublicKey, pubM []*Curve.BIG, privM []*Curve.BIG) (*BlindSignMats, error) {
	G, p, g1, hs, rng := params.G, params.p, params.g1, params.hs, params.G.Rng()

	if len(privM) <= 0 {
		return nil, ErrPrepareBlindSignPrivate
	}
	attributes := append(privM, pubM...)
	if len(attributes) > len(hs) {
		return nil, ErrPrepareBlindSignParams
	}

	r := Curve.Randomnum(p, rng)
	cm := Curve.G1mul(g1, r)

	cmElems := make([]*Curve.ECP, len(attributes))
	for i := range attributes {
		cmElems[i] = Curve.G1mul(hs[i], attributes[i])

	}
	for _, elem := range cmElems {
		cm.Add(elem)
	}

	b := make([]byte, constants.ECPLen)
	cm.ToBytes(b, true)

	h, err := utils.HashBytesToG1(amcl.SHA512, b)
	if err != nil {
		return nil, err
	}

	encs := make([]*elgamal.Encryption, len(privM))
	ks := make([]*Curve.BIG, len(privM))
	// can't easily encrypt in parallel since random number generator object is shared between encryptions
	for i := range privM {
		c, k := elgamal.Encrypt(G, egPub, privM[i], h)
		encs[i] = c
		ks[i] = k
	}

	signerProof, err := ConstructSignerProof(params, egPub.Gamma, encs, cm, ks, r, pubM, privM)
	if err != nil {
		return nil, err
	}
	return &BlindSignMats{
		cm:    cm,
		enc:   encs,
		proof: signerProof,
	}, nil
}

// BlindSign creates a blinded Coconut credential on the attributes provided to PrepareBlindSign.
// nolint: lll
func BlindSign(params *Params, sk *SecretKey, blindSignMats *BlindSignMats, egPub *elgamal.PublicKey, pubM []*Curve.BIG) (*BlindedSignature, error) {
	// todo: can optimize by calculating first pubM * yj and then do single G1mul rather than two of them

	hs := params.hs

	if len(blindSignMats.enc)+len(pubM) > len(hs) {
		return nil, ErrBlindSignParams
	}
	if !VerifySignerProof(params, egPub.Gamma, blindSignMats) {
		return nil, ErrBlindSignProof
	}

	b := make([]byte, constants.ECPLen)
	blindSignMats.cm.ToBytes(b, true)

	h, err := utils.HashBytesToG1(amcl.SHA512, b)
	if err != nil {
		return nil, err
	}

	t1 := make([]*Curve.ECP, len(pubM))
	for i := range pubM {
		t1[i] = Curve.G1mul(h, pubM[i])
	}

	t2 := Curve.NewECP()
	for i := 0; i < len(blindSignMats.enc); i++ {
		t2.Add(Curve.G1mul(blindSignMats.enc[i].C1(), sk.y[i]))
	}

	t3 := Curve.G1mul(h, sk.x)
	tmpSlice := make([]*Curve.ECP, len(blindSignMats.enc))
	for i := range blindSignMats.enc {
		tmpSlice[i] = blindSignMats.enc[i].C2()
	}
	tmpSlice = append(tmpSlice, t1...)

	// tmpslice: all B + t1
	t3Elems := make([]*Curve.ECP, len(tmpSlice))
	for i := range tmpSlice {
		t3Elems[i] = Curve.G1mul(tmpSlice[i], sk.y[i])
	}

	for _, elem := range t3Elems {
		t3.Add(elem)
	}

	return &BlindedSignature{
		sig1:      h,
		sig2Tilda: elgamal.NewEncryptionFromPoints(t2, t3),
	}, nil
}

// Unblind unblinds the blinded Coconut credential.
func Unblind(params *Params, blindedSignature *BlindedSignature, egPk *elgamal.PrivateKey) *Signature {
	G := params.G
	sig2 := elgamal.Decrypt(G, egPk, blindedSignature.sig2Tilda)
	return &Signature{
		sig1: blindedSignature.sig1,
		sig2: sig2,
	}
}

// Verify verifies the Coconut credential that has been either issued exlusiviely on public attributes
// or all private attributes have been publicly revealed
func Verify(params *Params, vk *VerificationKey, pubM []*Curve.BIG, sig *Signature) VerificationResult {
	G := params.G

	// should not really fail because as long as key is longer than numAttr,
	// it can create a valid credential
	// if len(pubM) != len(vk.beta) {
	// 	return false
	// }
	if len(pubM) > len(vk.beta) {
		return false
	}

	if sig == nil || sig.Sig1() == nil || sig.Sig2() == nil {
		return false
	}

	K := Curve.NewECP2()
	K.Copy(vk.alpha) // K = X
	tmp := make([]*Curve.ECP2, len(pubM))

	for i := 0; i < len(pubM); i++ {
		tmp[i] = Curve.G2mul(vk.beta[i], pubM[i]) // (ai * Yi)

	}
	for i := 0; i < len(pubM); i++ {
		K.Add(tmp[i]) // K = X + (a1 * Y1) + ...
	}

	var Gt1 *Curve.FP12
	var Gt2 *Curve.FP12

	Gt1 = G.Pair(sig.sig1, K)
	Gt2 = G.Pair(sig.sig2, vk.g2)

	return VerificationResult(!sig.sig1.Is_infinity() && Gt1.Equals(Gt2))
}

// ShowBlindSignature builds cryptographic material required for blind verification.
// It returns kappa and nu - group elements needed to perform verification
// and zero-knowledge proof asserting corectness of the above.
// nolint: lll
func ShowBlindSignature(params *Params, vk *VerificationKey, sig *Signature, privM []*Curve.BIG) (*BlindShowMats, error) {
	p, rng := params.p, params.G.Rng()

	if len(privM) <= 0 || len(privM) > len(vk.beta) {
		return nil, ErrShowBlindAttr
	}

	t := Curve.Randomnum(p, rng)
	kappa := Curve.G2mul(vk.g2, t)
	kappa.Add(vk.alpha)
	for i := range privM {
		kappa.Add(Curve.G2mul(vk.beta[i], privM[i]))
	}
	nu := Curve.G1mul(sig.sig1, t)

	verifierProof := ConstructVerifierProof(params, vk, sig, privM, t)

	return &BlindShowMats{
		kappa: kappa,
		nu:    nu,
		proof: verifierProof,
	}, nil
}

// BlindVerify verifies the Coconut credential on the private and optional public attributes.
// nolint: lll
func BlindVerify(params *Params, vk *VerificationKey, sig *Signature, showMats *BlindShowMats, pubM []*Curve.BIG) VerificationResult {
	G := params.G

	privateLen := len(showMats.proof.rm)
	if len(pubM)+privateLen > len(vk.beta) || !VerifyVerifierProof(params, vk, sig, showMats) {
		return false
	}

	if sig == nil || sig.Sig1() == nil || sig.Sig2() == nil {
		return false
	}

	aggr := Curve.NewECP2() // new point is at infinity
	if len(pubM) > 0 {
		for i := 0; i < len(pubM); i++ {
			aggr.Add(Curve.G2mul(vk.beta[i+privateLen], pubM[i]))
		}
	}

	t1 := Curve.NewECP2()
	t1.Copy(showMats.kappa)
	t1.Add(aggr)

	t2 := Curve.NewECP()
	t2.Copy(sig.sig2)
	t2.Add(showMats.nu)

	Gt1 := G.Pair(sig.sig1, t1)
	Gt2 := G.Pair(t2, vk.g2)

	return VerificationResult(!sig.sig1.Is_infinity() && Gt1.Equals(Gt2))
}

// Randomize randomizes the Coconut credential such that it becomes indistinguishable
// from a fresh credential on different attributes
func Randomize(params *Params, sig *Signature) *Signature {
	p, rng := params.p, params.G.Rng()

	var rSig Signature
	t := Curve.Randomnum(p, rng)

	rSig.sig1 = Curve.G1mul(sig.sig1, t)
	rSig.sig2 = Curve.G1mul(sig.sig2, t)

	return &rSig
}

// AggregateVerificationKeys aggregates verification keys of the signing authorities.
// Optionally it does so in a threshold manner.
func AggregateVerificationKeys(params *Params, vks []*VerificationKey, pp *PolynomialPoints) *VerificationKey {
	t := len(vks)
	if t <= 0 {
		return nil
	}

	p := params.p
	alpha := Curve.NewECP2()
	beta := make([]*Curve.ECP2, len(vks[0].beta))
	for i := range beta {
		beta[i] = Curve.NewECP2()
	}

	if pp != nil {
		l := utils.GenerateLagrangianCoefficients(t, p, pp.xs, 0)

		for i := 0; i < len(vks); i++ {
			alpha.Add(Curve.G2mul(vks[i].alpha, l[i]))
		}

		for i := 0; i < len(vks); i++ {
			for j := 0; j < len(beta); j++ {
				beta[j].Add(Curve.G2mul(vks[i].beta[j], l[i]))
			}
		}

	} else {
		for i := 0; i < len(vks); i++ {
			alpha.Add(vks[i].alpha)
		}

		for i := 0; i < len(vks); i++ { // we already copied values from first set of keys
			for j := 0; j < len(beta); j++ {
				beta[j].Add(vks[i].beta[j])
			}
		}
	}

	return &VerificationKey{
		g2:    vks[0].g2,
		alpha: alpha,
		beta:  beta,
	}
}

// AggregateSignatures aggregates Coconut credentials on the same set of attributes
// that were produced by multiple signing authorities.
// Optionally it does so in a threshold manner.
func AggregateSignatures(params *Params, sigs []*Signature, pp *PolynomialPoints) *Signature {
	t := len(sigs)
	if t <= 0 {
		return nil
	}

	p := params.p
	sig2 := Curve.NewECP()

	if pp != nil {
		l := utils.GenerateLagrangianCoefficients(t, p, pp.xs, 0)

		for i := 0; i < t; i++ {
			sig2.Add(Curve.G1mul(sigs[i].sig2, l[i]))
		}
	} else {
		for i := 0; i < t; i++ {
			sig2.Add(sigs[i].sig2)
		}
	}

	return &Signature{
		sig1: sigs[0].sig1,
		sig2: sig2,
	}
}

// ToPEMFile writes out the secret key to a PEM file at path f.
func (sk *SecretKey) ToPEMFile(f string) error {
	b, err := sk.MarshalBinary()
	if err != nil {
		return err
	}
	blk := &pem.Block{
		Type:  constants.SecretKeyType,
		Bytes: b,
	}
	return ioutil.WriteFile(f, pem.EncodeToMemory(blk), 0600)
}

// FromPEMFile reads out the secret key from a PEM file at path f.
func (sk *SecretKey) FromPEMFile(f string) error {
	if buf, err := ioutil.ReadFile(filepath.Clean(f)); err == nil {
		blk, rest := pem.Decode(buf)
		if len(rest) != 0 {
			return fmt.Errorf("trailing garbage after PEM encoded secret key")
		}
		if blk.Type != constants.SecretKeyType {
			return fmt.Errorf("invalid PEM Type: '%v'", blk.Type)
		}
		if sk.UnmarshalBinary(blk.Bytes) != nil {
			return errors.New("failed to read secret key from PEM file")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ToPEMFile writes out the verification key to a PEM file at path f.
func (vk *VerificationKey) ToPEMFile(f string) error {
	b, err := vk.MarshalBinary()
	if err != nil {
		return err
	}
	blk := &pem.Block{
		Type:  constants.VerificationKeyType,
		Bytes: b,
	}
	return ioutil.WriteFile(f, pem.EncodeToMemory(blk), 0600)
}

// FromPEMFile reads out the secret key from a PEM file at path f.
func (vk *VerificationKey) FromPEMFile(f string) error {
	if buf, err := ioutil.ReadFile(filepath.Clean(f)); err == nil {
		blk, rest := pem.Decode(buf)
		if len(rest) != 0 {
			return fmt.Errorf("trailing garbage after PEM encoded secret key")
		}
		if blk.Type != constants.VerificationKeyType {
			return fmt.Errorf("invalid PEM Type: '%v'", blk.Type)
		}
		if vk.UnmarshalBinary(blk.Bytes) != nil {
			return errors.New("failed to read verification key from PEM file")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}