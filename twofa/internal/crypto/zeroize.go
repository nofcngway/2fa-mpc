package crypto

// Zeroize overwrites all bytes in the slice with zeros.
// Used to clear TOTP secrets and share data from memory after use.
func Zeroize(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
