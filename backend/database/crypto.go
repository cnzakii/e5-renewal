package database

import (
	"crypto/sha256"
	"fmt"

	"e5-renewal/backend/services/security"
)

// encryptionKey is the 32-byte AES key derived from config.
var encryptionKey []byte

// MustInitEncryption derives a 32-byte AES key from keyStr.
// Panics if keyStr is empty — encryption is mandatory.
func MustInitEncryption(keyStr string) {
	if keyStr == "" {
		panic("E5_ENCRYPTION_KEY must be set — auth data encryption is required")
	}
	if len(keyStr) < 16 {
		panic("E5_ENCRYPTION_KEY must be at least 16 characters for adequate security")
	}
	hash := sha256.Sum256([]byte(keyStr))
	encryptionKey = hash[:]
}

func encryptAuthInfo(plain string) (string, error) {
	return security.EncryptString(encryptionKey, plain)
}

func decryptAuthInfo(stored string) (string, error) {
	result, err := security.DecryptString(encryptionKey, stored)
	if err != nil {
		return "", fmt.Errorf("decrypt auth_info: %w", err)
	}
	return result, nil
}
