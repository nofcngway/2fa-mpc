package crypto

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestZeroize(t *testing.T) {
	t.Run("zeroes all bytes", func(t *testing.T) {
		data := []byte{0xFF, 0xAB, 0x01, 0x99, 0x42}
		Zeroize(data)
		for i, b := range data {
			assert.Equal(t, byte(0), b, "byte at index %d should be zero", i)
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		data := []byte{}
		Zeroize(data) // should not panic
	})

	t.Run("handles nil slice", func(t *testing.T) {
		var data []byte
		Zeroize(data) // should not panic
	})

	t.Run("preserves slice length", func(t *testing.T) {
		data := make([]byte, 20)
		for i := range data {
			data[i] = byte(i + 1)
		}
		Zeroize(data)
		assert.Equal(t, 20, len(data))
		for _, b := range data {
			assert.Equal(t, byte(0), b)
		}
	})
}
