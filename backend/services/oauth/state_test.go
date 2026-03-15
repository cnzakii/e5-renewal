package oauth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/services/oauth"
)

func TestStateStoreBasic(t *testing.T) {
	store := &oauth.StateStore{}
	store.Reset()

	data := oauth.OAuthState{ClientID: "cid", TenantID: "tid"}
	state := store.NewState(data)
	assert.NotEmpty(t, state)

	got, ok := store.Consume(state)
	assert.True(t, ok)
	assert.Equal(t, "cid", got.ClientID)

	// Second consume should fail (single-use)
	_, ok = store.Consume(state)
	assert.False(t, ok)
}

func TestStateStoreInvalidKey(t *testing.T) {
	store := &oauth.StateStore{}
	_, ok := store.Consume("nonexistent")
	assert.False(t, ok)
}

func TestStateStorePreservesPerStateTTL(t *testing.T) {
	store := &oauth.StateStore{}
	store.Reset()
	state := store.NewState(oauth.OAuthState{
		ClientID:    "cid",
		RedirectURI: "http://localhost:3000/api/oauth/callback",
		TTL:         15 * time.Minute,
	})
	got, ok := store.Consume(state)
	require.True(t, ok)
	assert.Equal(t, 15*time.Minute, got.TTL)
}
