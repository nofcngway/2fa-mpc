// Package crypto provides cryptographic utilities for secure memory handling.
package crypto

import "runtime"

// Zeroize overwrites all bytes in the slice with zeros.
// Used to clear TOTP secrets and share data from memory after use.
// runtime.KeepAlive prevents the compiler from optimizing away clear()
// via dead store elimination.
func Zeroize(b []byte) {
	clear(b)
	runtime.KeepAlive(b)
}
