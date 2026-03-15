package security_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/services/security"
)

func TestHashPassword_ReturnsHash(t *testing.T) {
	hash, err := security.HashPassword("mypassword")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "mypassword", hash)
}

func TestHashPassword_DifferentHashesForSamePassword(t *testing.T) {
	h1, err := security.HashPassword("samepassword")
	require.NoError(t, err)

	h2, err := security.HashPassword("samepassword")
	require.NoError(t, err)

	// bcrypt uses random salt, so hashes should differ
	assert.NotEqual(t, h1, h2)
}

func TestVerifyPassword_CorrectPassword(t *testing.T) {
	hash, err := security.HashPassword("correctpassword")
	require.NoError(t, err)

	assert.True(t, security.VerifyPassword(hash, "correctpassword"))
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := security.HashPassword("correctpassword")
	require.NoError(t, err)

	assert.False(t, security.VerifyPassword(hash, "wrongpassword"))
}

func TestVerifyPassword_EmptyPassword(t *testing.T) {
	hash, err := security.HashPassword("notempty")
	require.NoError(t, err)

	assert.False(t, security.VerifyPassword(hash, ""))
}

func TestHashPassword_EmptyString(t *testing.T) {
	hash, err := security.HashPassword("")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	assert.True(t, security.VerifyPassword(hash, ""))
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	assert.False(t, security.VerifyPassword("not-a-bcrypt-hash", "password"))
}

func TestHashPassword_TooLongPassword(t *testing.T) {
	// bcrypt rejects passwords exceeding 72 bytes
	longPwd := string(make([]byte, 200))
	_, err := security.HashPassword(longPwd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "72 bytes")
}

func TestVerifyPassword_EmptyHash(t *testing.T) {
	assert.False(t, security.VerifyPassword("", "password"))
}
