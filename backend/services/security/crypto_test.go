package security_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/services/security"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plain := `{"client_id":"test","client_secret":"secret123"}`
	cipher, err := security.EncryptString(key, plain)
	require.NoError(t, err)
	assert.NotEqual(t, plain, cipher)

	decrypted, err := security.DecryptString(key, cipher)
	require.NoError(t, err)
	assert.Equal(t, plain, decrypted)
}

func TestEncryptDifferentEachTime(t *testing.T) {
	key := make([]byte, 32)
	c1, _ := security.EncryptString(key, "hello")
	c2, _ := security.EncryptString(key, "hello")
	assert.NotEqual(t, c1, c2) // nonce is random, so each encryption produces a different result
}

func TestDecryptWrongKey(t *testing.T) {
	key1, key2 := make([]byte, 32), make([]byte, 32)
	key2[0] = 1
	cipher, _ := security.EncryptString(key1, "secret")
	_, err := security.DecryptString(key2, cipher)
	assert.Error(t, err)
}
