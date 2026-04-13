// Package shamir implements Shamir Secret Sharing over GF(256).
// This file provides finite field arithmetic in GF(2^8) using the
// AES/Rijndael irreducible polynomial x^8 + x^4 + x^3 + x + 1 (0x11B).
package shamir

// logTable and expTable are lookup tables for GF(256) multiplication and division.
// They are populated once at init time using generator element 3.
var logTable [256]byte
var expTable [256]byte

// gfMulNoTable performs GF(256) multiplication using the shift-and-XOR
// (Russian peasant) algorithm with reduction by 0x1B. Used only during
// init() to build the log/exp tables; after init, gfMul uses table lookups.
func gfMulNoTable(a, b byte) byte {
	var result byte
	for b > 0 {
		if b&1 != 0 {
			result ^= a
		}
		carry := a & 0x80
		a <<= 1
		if carry != 0 {
			a ^= 0x1B
		}
		b >>= 1
	}
	return result
}

func init() {
	// Generate log/exp tables using generator element 3 in GF(256).
	// The generator 3 produces all 255 non-zero elements of the field.
	x := byte(1)
	for i := range 255 {
		expTable[i] = x
		logTable[x] = byte(i)
		x = gfMulNoTable(x, 3)
	}
	// Wrap: expTable[255] = expTable[0] for modular arithmetic convenience.
	expTable[255] = expTable[0]
	// logTable[0] remains 0 (undefined; log(0) is never used in valid operations).
}

// gfAdd returns the GF(256) sum of a and b (XOR in GF(2^n)).
func gfAdd(a, b byte) byte {
	return a ^ b
}

// gfMul returns the GF(256) product of a and b using log/exp table lookup.
// Returns 0 if either operand is zero (since log(0) is undefined).
func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return expTable[(int(logTable[a])+int(logTable[b]))%255]
}

// gfDiv returns the GF(256) quotient a/b. Panics if b is zero.
// Returns 0 if a is zero (0 divided by anything is 0).
func gfDiv(a, b byte) byte {
	if b == 0 {
		panic("shamir: division by zero in GF(256)")
	}
	if a == 0 {
		return 0
	}
	return expTable[(int(logTable[a])-int(logTable[b])+255)%255]
}

// gfInv returns the multiplicative inverse of a in GF(256). Panics if a is zero.
func gfInv(a byte) byte {
	if a == 0 {
		panic("shamir: inverse of zero in GF(256)")
	}
	return expTable[255-int(logTable[a])]
}
