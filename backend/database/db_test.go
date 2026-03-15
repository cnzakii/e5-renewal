package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB initializes an in-memory SQLite database and encryption for tests.
func setupTestDB(t *testing.T) context.Context {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	require.NoError(t, Init(dsn))
	MustInitEncryption("test-encryption-key-1234")
	return context.Background()
}

func TestInit_InMemory(t *testing.T) {
	err := Init(":memory:")
	require.NoError(t, err)
	assert.NotNil(t, globalDB)
}

func TestGetDB_ReturnsNonNil(t *testing.T) {
	require.NoError(t, Init(":memory:"))
	db := GetDB(context.Background())
	require.NotNil(t, db)
}

func TestInit_WithDirectoryPath(t *testing.T) {
	dir := t.TempDir()
	err := Init(dir + "/sub/test.db")
	require.NoError(t, err)
}

func TestSingletonRepos_NotNil(t *testing.T) {
	assert.NotNil(t, Accounts)
	assert.NotNil(t, TaskLogs)
	assert.NotNil(t, Schedules)
	assert.NotNil(t, Settings)
	assert.NotNil(t, EndpointLogs)
}
