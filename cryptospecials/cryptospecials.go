/*
*	This package contains mechanisms that will allow for VRF and OPRF calculations.
*
*	OPRF: https://eprint.iacr.org/2017/111
*
*	RSA-VRF: https://eprint.iacr.org/2017/099.pdf
*
*		-Brian
 */

package cryptospecials

import (
	"bytes"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"math/big"
)

// big.Int representation of Zero & One
var (
	zero = big.NewInt(int64(0))
	one  = big.NewInt(int64(1))
)

//RSAVRF is an exportable struct
type RSAVRF struct {
	Proof []byte
	Beta  []byte
	*rsa.PrivateKey
}

//ECCVRF is an exportable struct
type ECCVRF struct {
	Proof []byte
	Beta  []byte
	elliptic.Curve
}

//OPRF is an exportable struct
type OPRF struct {
	RSecret    []byte
	RSecretInv []byte
	elliptic.Curve
}

// Hash2curve is an exportable function
/*
*  Warning: Try & Increment is not a constant-time algorithm
*  Warning: This function requres cryptographic vetting!
*  hash2curve implements the Try & Increment method for hashing into an Elliptic
*  Curve. Note: hash2curve only works on weierstrass curves at this point
 */
func Hash2curve(data []byte, h hash.Hash, eCurve *elliptic.CurveParams, curveType int, verbose bool) error {

	/*
	*  Curve Type:
	*		(1) Weierstrass - NIST Curves
	*		(2) Others - Not currently supported
	 */

	var (
		xByte   []byte
		counter int
	)

	x := big.NewInt(0)
	y := big.NewInt(0)
	one := new(big.Int).SetInt64(int64(1))
	h.Write(data)
	xByte = h.Sum(nil)
	x.SetBytes(xByte)
	x.Mod(x, eCurve.P)

	if verbose {
		fmt.Println("Length of xByte:", len(xByte))
		fmt.Println("P:", eCurve.P)
		fmt.Println("B:", eCurve.B)
	}

	if curveType == 1 {
		/*
		* Determine (x^3 -3x + b)^(1/2) as defined by the weierstrass Elliptic Curve. Parts taken from:
		*    https://golang.org/src/crypto/elliptic/elliptic.go?s=2054:2109#L55
		 */

		x3 := big.NewInt(0)
		threeX := big.NewInt(0)
		for {

			x3.Mul(x, x)
			x3.Mul(x3, x)

			threeX.Lsh(x, 1)
			threeX.Add(threeX, x)

			x3.Sub(x3, threeX)
			x3.Add(x3, eCurve.B)
			x3.Mod(x3, eCurve.P) // x^3 -3x + b
			//fmt.Println("x3:", x3)

			// Use Jacobi symbols to determine if x is a quadratic residue in F_p
			if big.Jacobi(x3, eCurve.Params().P) == 1 {
				break
			}

			x.Add(x, one)
			counter++
		}
		/*
		*  ModSqrt does not account for degenerate roots
		 */
		y.ModSqrt(x3, eCurve.P)

	} else {
		return fmt.Errorf("Error: Unsupported curve type. Currently support only Weierstrass curves")
	}

	// Double check that the point (x,y) is on the provided elliptic curve
	if eCurve.IsOnCurve(x, y) != true {
		return fmt.Errorf("Error: Unable to hash data onto curve! Point (x,y) not on given elliptic curve")
	}

	if verbose {
		fmt.Printf("Number of Try & Increment iterations: %d\n", counter)
		fmt.Println("x-xoordinate:", x)
		fmt.Println("x-coordinate bit length:", x.BitLen())
		fmt.Println("y-coordinate:", y)
		fmt.Println("y-coordinate bit length:", y.BitLen())
	}

	return nil
}

func (rep OPRF) Recv() {

}

func (rep OPRF) Send() {

}

func (rep ECCVRF) Generate() {

}

func (rep ECCVRF) Verify() {

}

// Generate is an exportable method
/*
*  This function has NOT been tested for cryptographic soundness at this point.
*  Do not use for cryptographic applications until it has been properly vetted.
*  The RSA-VRF below is based on: https://eprint.iacr.org/2017/099.pdf. Alpha is
*  the VRF mutual reference string.
 */
func (vrf RSAVRF) Generate(alpha []byte, verbose bool) ([]byte, []byte, error) {

	// Check that vrf has values for N (big Int) and D (big Int) or return error
	if vrf.N == zero || vrf.D == zero {
		return nil, nil, errors.New("Error: No private key supplied")
	}

	var (
		modLength int
		proof     big.Int
		outInt    big.Int
		beta      []byte
	)

	/*
	*  ModLength is the BYTE length of the modulus N.  output should be
	*  byte_length(RSA_Modulus) - 1. The "minus 1" is EXTREMLY important.
	*  If output is set to the length of N then the result of mgf1XOR could
	*  be bigger than N which in tern will cause issues with RSA verification.
	 */
	modLength = ((vrf.N.BitLen() + 8) / 8) - 1
	output := make([]byte, modLength-1)
	hash256 := sha256.New()

	mgf1XOR(output, hash256, alpha)

	/*
	*  Warning: SetBytes interprets Bytes as Big-endian!
	*  Exponentiate = Proof (pi) = MGF1(alpha)^D (mod N)
	*  D is an RSA Secret!
	 */
	outInt.SetBytes(output)
	proof.Exp(&outInt, vrf.D, vrf.N)

	hash256.Reset()
	hash256.Write(proof.Bytes())
	beta = hash256.Sum(nil) //Outputs in big-endian

	if verbose {
		fmt.Printf("****VRF.generate - Verbose Output****\n")

		fmt.Println("RSA Pub Mod - N (big Int):", vrf.N)
		fmt.Println("RSA Pub Exp - E (int):", vrf.E)
		fmt.Println("SECRET - RSA Priv Exp - D (big Int):", vrf.D)

		fmt.Printf("Alpha (string): %s\n", alpha)
		fmt.Printf("Alpha (hex): %x\n", alpha)

		fmt.Println("Proof (big Int):", proof)
		fmt.Printf("Proof (hex): %x\n", proof.Bytes())
		fmt.Printf("H(proof) (hex): %x\n", hash256.Sum(nil))
		fmt.Printf("Beta = H(proof) (hex): %x\n", beta)

		fmt.Printf("Length of MGF1 output (Should be < len(N)): %d", len(output))
		fmt.Printf("MGF1 Output (hex): %x\n", output)
		fmt.Println("MGF1 Output as big.Int:", outInt)
	}

	return proof.Bytes(), beta, nil

}

// Verify is an exportable method
/*
*  This function has NOT been tested for cryptographic soundness at this point.
*  Do not use for cryptographic applications until it has been properly vetted.
*  The RSA-VRF below is based on: https://eprint.iacr.org/2017/099.pdf. Alpha is
*  the VRF mutual reference string.
 */
func (vrf RSAVRF) Verify(alpha []byte, pubKey *rsa.PublicKey, verbose bool) (bool, error) {

	// Check that vrf has values for proof and beta or return error
	if vrf.Proof == nil || vrf.Beta == nil {
		return false, errors.New("Error: H(beta) or Beta not supplied")
	}
	// Check that the pub key has values for N (bit Int) and E (int) or return error
	if pubKey.N == zero || pubKey.E == 0 {
		return false, errors.New("Error: No public key not supplied")
	}

	var (
		modLength int
		betaCheck []byte
		mgf1Alpha []byte
		intCheck  big.Int
		mgf1Check big.Int
	)

	/*
	*  ModLength is the BYTE length of the modulus N.  output should be
	*  byte_length(RSA_Modulus) - 1. The "minus 1" is EXTREMLY important.
	*  If output is set to the length of N then the result of mgf1XOR could
	*  be bigger than N which in tern will cause issues with RSA verification.
	 */
	modLength = ((vrf.N.BitLen() + 8) / 8) - 1
	mgf1Alpha = make([]byte, modLength-1)
	hash256 := sha256.New()

	mgf1XOR(mgf1Alpha, hash256, alpha)

	hash := sha256.New()
	hash.Write(vrf.Proof)
	betaCheck = hash.Sum(nil)

	/*
	*  Generate: mgf1Check = (MGF1(alpha)^d)^e
	*  Eventually, want to check: mgf1Check ?= MGF1(alpha)
	 */
	e := big.NewInt(int64(pubKey.E))
	intCheck.SetBytes(vrf.Proof)
	mgf1Check.Exp(&intCheck, e, pubKey.N)

	if verbose {
		fmt.Printf("****VRF.verify - Verbose Output****\n")

		fmt.Println("RSA Pub Mod - N (big Int):", pubKey.N)
		fmt.Println("RSA Pub Exp - E (int)", e)

		fmt.Println("vrf.proof (big Int)", intCheck)
		fmt.Printf("H(vrf.proof) - should be equal to Beta (hex): %x\n", betaCheck)
		fmt.Printf("Beta (hex): %x\n", vrf.Beta)

		fmt.Printf("MGF1(Alpha) (hex): %x\n", mgf1Alpha)
		fmt.Println("MGF1Check (Big Int):", mgf1Check)
		fmt.Printf("MGF1Check (hex): %x\n", mgf1Check.Bytes())
	}

	// Check: compare the bytes of betaCheck = H(vrf.proof) ?= beta
	if bytes.Compare(betaCheck, vrf.Beta) != 0 {
		if verbose {
			fmt.Println("FAIL - Could not verify: Beta == H(proof)")
		}
		return false, nil
	}

	//Check: compare the bytes of (mgf1Check == trial_MGF1(alpha)) ?= (mgf1Alpha = MGF1(alpha))
	if bytes.Compare(mgf1Check.Bytes(), mgf1Alpha) != 0 {
		if verbose {
			fmt.Println("FAIL - Could not verify: trial_MGF1(alpha) == MGF1(alpha)")
		}
		return false, nil
	}

	return true, nil
}

/*
* Taken from Golang source:
*   https://golang.org/src/crypto/rsa/rsa.go?s=11736:11844#L308
* incCounter increments a four byte, big-endian counter.
 */
func incCounter(c *[4]byte) {
	if c[3]++; c[3] != 0 {
		return
	}
	if c[2]++; c[2] != 0 {
		return
	}
	if c[1]++; c[1] != 0 {
		return
	}

	c[0]++
}

/*
*  Taken from Golang source:
*    https://golang.org/src/crypto/rsa/rsa.go?s=11736:11844#L323
*  mgf1XOR XORs the bytes in out with a mask generated using the MGF1 function
*  specified in PKCS#1 v2.1.
 */
func mgf1XOR(out []byte, hash hash.Hash, seed []byte) {

	var counter [4]byte
	var digest []byte

	done := 0
	for done < len(out) {
		hash.Write(seed)
		hash.Write(counter[0:4])
		digest = hash.Sum(digest[:0])
		hash.Reset()

		for i := 0; i < len(digest) && done < len(out); i++ {
			out[done] ^= digest[i]
			done++
		}
		incCounter(&counter)
	}
}
