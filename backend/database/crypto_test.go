package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustInitEncryption_ValidKey(t *testing.T) {
	assert.NotPanics(t, func() {
		MustInitEncryption("this-is-a-valid-key-1234")
	})
	assert.Len(t, encryptionKey, 32)
}

func TestMustInitEncryption_EmptyKey(t *testing.T) {
	assert.PanicsWithValue(t, "E5_ENCRYPTION_KEY must be set — auth data encryption is required", func() {
		MustInitEncryption("")
	})
}

func TestMustInitEncryption_ShortKey(t *testing.T) {
	assert.PanicsWithValue(t, "E5_ENCRYPTION_KEY must be at least 16 characters for adequate security", func() {
		MustInitEncryption("short")
	})
}

func TestEncryptDecryptAuthInfo_RoundTrip(t *testing.T) {
	MustInitEncryption("test-encryption-key-1234")

	plaintext := `{"client_id":"abc","client_secret":"xyz"}`
	encrypted, err := encryptAuthInfo(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := decryptAuthInfo(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptAuthInfo_ProducesDifferentCiphertexts(t *testing.T) {
	MustInitEncryption("test-encryption-key-1234")

	plaintext := "same-input"
	enc1, err := encryptAuthInfo(plaintext)
	require.NoError(t, err)
	enc2, err := encryptAuthInfo(plaintext)
	require.NoError(t, err)

	// AES-GCM with random nonce should produce different ciphertexts.
	assert.NotEqual(t, enc1, enc2)
}

func TestDecryptAuthInfo_InvalidData(t *testing.T) {
	MustInitEncryption("test-encryption-key-1234")

	_, err := decryptAuthInfo("not-valid-base64!!!")
	assert.Error(t, err)
}

func TestEncryptDecryptAuthInfo_EmptyString(t *testing.T) {
	MustInitEncryption("test-encryption-key-1234")

	encrypted, err := encryptAuthInfo("")
	require.NoError(t, err)

	decrypted, err := decryptAuthInfo(encrypted)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}
