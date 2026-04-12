package mpcService

import (
	"bytes"
	"crypto/rand"
	"testing"

	"gotest.tools/v3/assert"
)

// testKey returns a fixed 32-byte AES-256 key for testing.
func testKey() []byte {
	return []byte("01234567890123456789012345678901") // exactly 32 bytes
}

func newEncryptService() *MPCService {
	return NewMPCService(nil, testKey(), 1, nil)
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	svc := newEncryptService()
	plaintext := []byte("secret share data for testing")

	ciphertext, nonce, err := svc.encrypt(plaintext)
	assert.NilError(t, err)
	assert.Assert(t, len(ciphertext) > 0, "ciphertext should not be empty")
	assert.Assert(t, len(nonce) == 12, "nonce should be 12 bytes, got %d", len(nonce))

	decrypted, err := svc.decrypt(ciphertext, nonce)
	assert.NilError(t, err)
	assert.Assert(t, bytes.Equal(decrypted, plaintext),
		"decrypted data should match original plaintext")
}

func TestEncryptProducesDifferentNonces(t *testing.T) {
	svc := newEncryptService()
	plaintext := []byte("same data twice")

	_, nonce1, err := svc.encrypt(plaintext)
	assert.NilError(t, err)

	_, nonce2, err := svc.encrypt(plaintext)
	assert.NilError(t, err)

	assert.Assert(t, !bytes.Equal(nonce1, nonce2),
		"two encrypt calls should produce different nonces")
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	svc := newEncryptService()
	plaintext := []byte("same data twice")

	ct1, _, err := svc.encrypt(plaintext)
	assert.NilError(t, err)

	ct2, _, err := svc.encrypt(plaintext)
	assert.NilError(t, err)

	assert.Assert(t, !bytes.Equal(ct1, ct2),
		"two encrypt calls should produce different ciphertexts")
}

func TestDecryptWrongKey(t *testing.T) {
	svc := newEncryptService()
	plaintext := []byte("secret data")

	ciphertext, nonce, err := svc.encrypt(plaintext)
	assert.NilError(t, err)

	// Create a service with a different 32-byte key.
	wrongKey := make([]byte, 32)
	_, err = rand.Read(wrongKey)
	assert.NilError(t, err)
	wrongSvc := NewMPCService(nil, wrongKey, 1, nil)

	_, err = wrongSvc.decrypt(ciphertext, nonce)
	assert.Assert(t, err != nil, "decrypt with wrong key should fail")
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	svc := newEncryptService()
	plaintext := []byte("secret data")

	ciphertext, nonce, err := svc.encrypt(plaintext)
	assert.NilError(t, err)

	// Corrupt one byte of the ciphertext.
	corrupted := make([]byte, len(ciphertext))
	copy(corrupted, ciphertext)
	corrupted[0] ^= 0xFF

	_, err = svc.decrypt(corrupted, nonce)
	assert.Assert(t, err != nil, "decrypt with corrupted ciphertext should fail")
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	svc := newEncryptService()
	plaintext := []byte{}

	ciphertext, nonce, err := svc.encrypt(plaintext)
	assert.NilError(t, err)
	assert.Assert(t, len(ciphertext) > 0, "ciphertext for empty plaintext should contain GCM tag")

	decrypted, err := svc.decrypt(ciphertext, nonce)
	assert.NilError(t, err)
	assert.Assert(t, len(decrypted) == 0,
		"decrypted empty plaintext should be empty, got %d bytes", len(decrypted))
}

func TestDecryptInvalidNonce(t *testing.T) {
	svc := newEncryptService()
	plaintext := []byte("secret data")

	ciphertext, _, err := svc.encrypt(plaintext)
	assert.NilError(t, err)

	// Use a wrong-length nonce (5 bytes instead of 12).
	wrongNonce := []byte{1, 2, 3, 4, 5}
	_, err = svc.decrypt(ciphertext, wrongNonce)
	assert.Assert(t, err != nil, "decrypt with invalid nonce length should fail")
}
