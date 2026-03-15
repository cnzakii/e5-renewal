package login_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/database"
	"e5-renewal/backend/services/login"
	"e5-renewal/backend/services/security"
)

func initTestDB(t *testing.T) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	err := database.Init(dsn)
	require.NoError(t, err)
	database.MustInitEncryption("test-encryption-key-minimum16chars")
}

func TestMustInit_WithConfiguredKey(t *testing.T) {
	initTestDB(t)

	login.MustInit("my-configured-key")
	// Key() now returns bcrypt hash; verify it matches the configured plaintext
	assert.True(t, security.VerifyPassword(login.Key(), "my-configured-key"))
}

func TestMustInit_GeneratesKeyWhenEmpty(t *testing.T) {
	initTestDB(t)

	login.MustInit("")
	key := login.Key()
	assert.NotEmpty(t, key)
	// Generated key is stored as bcrypt hash, which is always 60 chars
	assert.GreaterOrEqual(t, len(key), 16)
}

func TestMustInit_ReusesExistingKey(t *testing.T) {
	initTestDB(t)

	// First init generates a key
	login.MustInit("")
	firstKey := login.Key()
	assert.NotEmpty(t, firstKey)

	// Second init with configured key overrides
	login.MustInit("explicit-key")
	assert.True(t, security.VerifyPassword(login.Key(), "explicit-key"))

	// Re-init with empty should find the stored hash from the previous upsert
	login.MustInit("")
	storedKey := login.Key()
	assert.NotEmpty(t, storedKey)
	// The stored hash should still verify against "explicit-key"
	assert.True(t, security.VerifyPassword(storedKey, "explicit-key"))
}

func TestMustInit_ConfiguredKeyOverridesExisting(t *testing.T) {
	initTestDB(t)

	login.MustInit("key-1")
	assert.True(t, security.VerifyPassword(login.Key(), "key-1"))

	login.MustInit("key-2")
	assert.True(t, security.VerifyPassword(login.Key(), "key-2"))
}

func TestKey_ReturnsCurrentValue(t *testing.T) {
	initTestDB(t)

	login.MustInit("test-key-value")
	assert.True(t, security.VerifyPassword(login.Key(), "test-key-value"))
}
