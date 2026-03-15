package security_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/services/security"
)

func TestSignJWT_ReturnsValidToken(t *testing.T) {
	secret := []byte("test-secret-key-1234567890")
	token, err := security.SignJWT(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestParseJWT_ValidToken(t *testing.T) {
	secret := []byte("test-secret-key-1234567890")
	token, err := security.SignJWT(secret)
	require.NoError(t, err)

	claims, err := security.ParseJWT(secret, token)
	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "e5-renewal", claims.Issuer)
	assert.WithinDuration(t, time.Now(), claims.IssuedAt.Time, 5*time.Second)
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), claims.ExpiresAt.Time, 5*time.Second)
}

func TestParseJWT_WrongSecret(t *testing.T) {
	secret1 := []byte("correct-secret-key-12345")
	secret2 := []byte("wrong-secret-key-123456")

	token, err := security.SignJWT(secret1)
	require.NoError(t, err)

	_, err = security.ParseJWT(secret2, token)
	assert.Error(t, err)
}

func TestParseJWT_ExpiredToken(t *testing.T) {
	secret := []byte("test-secret-key-1234567890")

	// Manually create an expired token
	claims := security.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "e5-renewal",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-48 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(secret)
	require.NoError(t, err)

	_, err = security.ParseJWT(secret, tokenStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestParseJWT_MalformedToken(t *testing.T) {
	secret := []byte("test-secret-key-1234567890")

	_, err := security.ParseJWT(secret, "not-a-valid-jwt-token")
	assert.Error(t, err)
}

func TestParseJWT_EmptyToken(t *testing.T) {
	secret := []byte("test-secret-key-1234567890")

	_, err := security.ParseJWT(secret, "")
	assert.Error(t, err)
}

func TestSignJWT_DifferentSecretsDifferentTokens(t *testing.T) {
	secret1 := []byte("secret-one-1234567890abc")
	secret2 := []byte("secret-two-1234567890abc")

	token1, err := security.SignJWT(secret1)
	require.NoError(t, err)

	token2, err := security.SignJWT(secret2)
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2)
}

func TestParseJWT_RejectsNoneAlgorithm(t *testing.T) {
	secret := []byte("test-secret-key-1234567890")

	// Create a token with "none" signing method
	claims := security.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "e5-renewal",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = security.ParseJWT(secret, tokenStr)
	assert.Error(t, err)
}
