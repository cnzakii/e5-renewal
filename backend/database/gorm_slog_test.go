package database

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSlogLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	}))
}

func TestGormSlogLoggerTrace_LogsSQLWithSubsystemMetadata(t *testing.T) {
	var buf bytes.Buffer
	logger := NewGormSlogLogger(newTestSlogLogger(&buf), GormSlogOptions{
		SlowThreshold: 200 * time.Millisecond,
	})

	logger.Trace(context.Background(), time.Now().Add(-50*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM settings WHERE key = 'ui_theme'", 1
	}, nil)

	output := buf.String()
	require.NotEmpty(t, output)
	assert.Contains(t, output, "level=INFO")
	assert.Contains(t, output, "msg=sql")
	assert.Contains(t, output, "subsystem=db")
	assert.Contains(t, output, "rows=1")
	assert.Contains(t, output, "sql=\"SELECT * FROM settings WHERE key = 'ui_theme'\"")
}

func TestGormSlogLoggerTrace_ElevatesSlowQueries(t *testing.T) {
	var buf bytes.Buffer
	logger := NewGormSlogLogger(newTestSlogLogger(&buf), GormSlogOptions{
		SlowThreshold: 100 * time.Millisecond,
	})

	logger.Trace(context.Background(), time.Now().Add(-250*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM task_logs", 5
	}, nil)

	output := buf.String()
	require.NotEmpty(t, output)
	assert.Contains(t, output, "level=WARN")
	assert.Contains(t, output, "msg=slow_sql")
	assert.Contains(t, output, "subsystem=db")
	assert.Contains(t, output, "slow=true")
	assert.Contains(t, output, "sql=\"SELECT * FROM task_logs\"")
}

func TestGormSlogLoggerTrace_LogsErrorsWithoutLeakingSecrets(t *testing.T) {
	var buf bytes.Buffer
	logger := NewGormSlogLogger(newTestSlogLogger(&buf), GormSlogOptions{
		SlowThreshold: time.Second,
	})

	logger.Trace(context.Background(), time.Now().Add(-20*time.Millisecond), func() (string, int64) {
		return "INSERT INTO accounts (name, auth_info) VALUES ('primary', '{\"client_secret\":\"super-secret\",\"refresh_token\":\"refresh-token-value\"}')", 0
	}, errors.New("constraint failed"))

	output := buf.String()
	require.NotEmpty(t, output)
	assert.Contains(t, output, "level=ERROR")
	assert.Contains(t, output, "msg=sql_error")
	assert.Contains(t, output, "subsystem=db")
	assert.Contains(t, output, "error=\"constraint failed\"")
	assert.Contains(t, output, "sql=\"")
	assert.Contains(t, output, "INSERT INTO accounts")
	assert.NotContains(t, output, "super-secret")
	assert.NotContains(t, output, "refresh-token-value")
}

func TestGormSlogLoggerTrace_RedactsPlainSQLSensitiveColumnValues(t *testing.T) {
	var buf bytes.Buffer
	logger := NewGormSlogLogger(newTestSlogLogger(&buf), GormSlogOptions{
		SlowThreshold: time.Second,
	})

	logger.Trace(context.Background(), time.Now().Add(-20*time.Millisecond), func() (string, int64) {
		return "INSERT INTO accounts (client_secret, refresh_token, tenant_id) VALUES ('plain-secret', 'plain-refresh-token', 'tenant-1')", 1
	}, nil)

	output := buf.String()
	require.NotEmpty(t, output)
	assert.Contains(t, output, "INSERT INTO accounts")
	assert.Contains(t, output, "[REDACTED]")
	assert.Contains(t, output, "tenant-1")
	assert.NotContains(t, output, "plain-secret")
	assert.NotContains(t, output, "plain-refresh-token")
}

func TestGormSlogLoggerTrace_RedactsBearerTokensFully(t *testing.T) {
	var buf bytes.Buffer
	logger := NewGormSlogLogger(newTestSlogLogger(&buf), GormSlogOptions{
		SlowThreshold: time.Second,
	})

	logger.Trace(context.Background(), time.Now().Add(-20*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM endpoint_logs WHERE response_body = 'Authorization: Bearer token-value-123'", 1
	}, nil)

	output := buf.String()
	require.NotEmpty(t, output)
	assert.Contains(t, output, "Authorization: Bearer [REDACTED]")
	assert.NotContains(t, output, "token-value-123")
}

func TestGormSlogLoggerTrace_RedactsSensitiveInsertValuesWithQuotedCommas(t *testing.T) {
	var buf bytes.Buffer
	logger := NewGormSlogLogger(newTestSlogLogger(&buf), GormSlogOptions{
		SlowThreshold: time.Second,
	})

	logger.Trace(context.Background(), time.Now().Add(-20*time.Millisecond), func() (string, int64) {
		return "INSERT INTO accounts (client_secret, auth_info, tenant_id) VALUES ('plain,secret', '{\"note\":\"kept,visible\"}', 'tenant-1')", 1
	}, nil)

	output := buf.String()
	require.NotEmpty(t, output)
	assert.Contains(t, output, "INSERT INTO accounts")
	assert.Contains(t, output, "'[REDACTED]'", output)
	assert.Contains(t, output, "kept,visible")
	assert.Contains(t, output, "tenant-1")
	assert.NotContains(t, output, "plain,secret")
}

func TestGormSlogLoggerTrace_RedactsSensitiveAssignments(t *testing.T) {
	var buf bytes.Buffer
	logger := NewGormSlogLogger(newTestSlogLogger(&buf), GormSlogOptions{
		SlowThreshold: time.Second,
	})

	logger.Trace(context.Background(), time.Now().Add(-20*time.Millisecond), func() (string, int64) {
		return "UPDATE endpoint_logs SET response_body = 'Authorization: Bearer token-value-456', secret = 'plain-secret' WHERE id = 1", 1
	}, nil)

	output := buf.String()
	require.NotEmpty(t, output)
	assert.Contains(t, output, "response_body = 'Authorization: Bearer [REDACTED]'")
	assert.Contains(t, output, "secret = '[REDACTED]'")
	assert.NotContains(t, output, "token-value-456")
	assert.NotContains(t, output, "plain-secret")
}

func TestInit_UsesGormSlogLogger(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(newTestSlogLogger(&buf))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	require.NoError(t, Init(":memory:"))
	require.NotNil(t, globalDB)
	_, ok := globalDB.Logger.(*gormSlogLogger)
	assert.True(t, ok)
}
