package login

import (
	"context"
	"fmt"

	"e5-renewal/backend/database"
	"e5-renewal/backend/models"
	"e5-renewal/backend/services/security"

	"github.com/google/uuid"
)

var globalLoginKey string

// MustInit resolves the login key and stores it as a singleton. Panics on error.
func MustInit(configuredKey string) {
	hash, plaintext, err := initKey(configuredKey)
	if err != nil {
		panic(fmt.Sprintf("init login key: %v", err))
	}
	globalLoginKey = hash
	if plaintext != "" {
		fmt.Printf("=== Login key generated ===\n")
		fmt.Printf("  Key: %s\n", plaintext)
		fmt.Printf("===========================\n")
	}
}

// Key returns the resolved login key.
func Key() string {
	return globalLoginKey
}

// initKey returns (bcryptHash, plaintextIfGenerated, error).
// The hash is always stored in globalLoginKey for bcrypt verification.
func initKey(configuredKey string) (hash string, plaintext string, err error) {
	ctx := context.Background()
	if configuredKey != "" {
		h, err := security.HashPassword(configuredKey)
		if err != nil {
			return "", "", err
		}
		if err := upsertLoginKey(ctx, h); err != nil {
			return "", "", err
		}
		return h, "", nil
	}

	existing, err := loadLoginKey(ctx)
	if err != nil {
		return "", "", err
	}
	if existing != "" {
		return existing, "", nil
	}

	key := uuid.NewString()
	h, err := security.HashPassword(key)
	if err != nil {
		return "", "", err
	}
	if err := upsertLoginKey(ctx, h); err != nil {
		return "", "", err
	}
	return h, key, nil
}

func loadLoginKey(ctx context.Context) (string, error) {
	val, err := database.Settings.Get(ctx, models.SettingKeyLoginKey)
	if err != nil {
		return "", err
	}
	return val, nil
}

func upsertLoginKey(ctx context.Context, value string) error {
	return database.Settings.Upsert(ctx, models.SettingKeyLoginKey, value)
}
