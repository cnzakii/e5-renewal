package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingRepo_Get_NotFound(t *testing.T) {
	ctx := setupTestDB(t)

	val, err := Settings.Get(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestSettingRepo_Upsert_Create(t *testing.T) {
	ctx := setupTestDB(t)

	require.NoError(t, Settings.Upsert(ctx, "theme", "dark"))

	val, err := Settings.Get(ctx, "theme")
	require.NoError(t, err)
	assert.Equal(t, "dark", val)
}

func TestSettingRepo_Upsert_Update(t *testing.T) {
	ctx := setupTestDB(t)

	require.NoError(t, Settings.Upsert(ctx, "theme", "dark"))
	require.NoError(t, Settings.Upsert(ctx, "theme", "light"))

	val, err := Settings.Get(ctx, "theme")
	require.NoError(t, err)
	assert.Equal(t, "light", val)
}

func TestSettingRepo_MultipleKeys(t *testing.T) {
	ctx := setupTestDB(t)

	require.NoError(t, Settings.Upsert(ctx, "key1", "val1"))
	require.NoError(t, Settings.Upsert(ctx, "key2", "val2"))

	v1, err := Settings.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, "val1", v1)

	v2, err := Settings.Get(ctx, "key2")
	require.NoError(t, err)
	assert.Equal(t, "val2", v2)
}

func TestSettingRepo_Upsert_EmptyValue(t *testing.T) {
	ctx := setupTestDB(t)

	require.NoError(t, Settings.Upsert(ctx, "empty", ""))

	val, err := Settings.Get(ctx, "empty")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestSettingRepo_Upsert_JSONValue(t *testing.T) {
	ctx := setupTestDB(t)

	jsonVal := `{"url":"https://example.com","on_task_all_failed":true}`
	require.NoError(t, Settings.Upsert(ctx, "notification", jsonVal))

	val, err := Settings.Get(ctx, "notification")
	require.NoError(t, err)
	assert.Equal(t, jsonVal, val)
}
