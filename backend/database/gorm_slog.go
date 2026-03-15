package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

var (
	sensitiveValuePattern = regexp.MustCompile(`(?i)("(?:client_secret|refresh_token|access_token|token|secret)"\s*:\s*")[^"]*(")`)
	bearerTokenPattern    = regexp.MustCompile(`(?i)Bearer\s+[^\s'";,)]+`)
)

// GormSlogOptions configures the GORM slog adapter.
type GormSlogOptions struct {
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
	LogLevel                  gormlogger.LogLevel
}

// NewGormSlogLogger returns a GORM logger that emits records through slog.
func NewGormSlogLogger(base *slog.Logger, options GormSlogOptions) gormlogger.Interface {
	if base == nil {
		base = slog.Default()
	}

	log := base.With(
		slog.String("subsystem", "db"),
		slog.String("module", "gorm"),
	)

	if options.LogLevel == 0 {
		options.LogLevel = gormlogger.Info
	}
	if options.SlowThreshold == 0 {
		options.SlowThreshold = 200 * time.Millisecond
	}

	return &gormSlogLogger{
		logger:  log,
		options: options,
	}
}

type gormSlogLogger struct {
	logger  *slog.Logger
	options GormSlogOptions
}

func (l *gormSlogLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	copy := *l
	copy.options.LogLevel = level
	return &copy
}

func (l *gormSlogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.options.LogLevel < gormlogger.Info {
		return
	}

	l.logger.Log(ctx, slog.LevelInfo, fmt.Sprintf(msg, data...))
}

func (l *gormSlogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.options.LogLevel < gormlogger.Warn {
		return
	}

	l.logger.Log(ctx, slog.LevelWarn, fmt.Sprintf(msg, data...))
}

func (l *gormSlogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.options.LogLevel < gormlogger.Error {
		return
	}

	l.logger.Log(ctx, slog.LevelError, fmt.Sprintf(msg, data...))
}

func (l *gormSlogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.options.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	attrs := []any{
		slog.Duration("duration", elapsed),
	}

	sql, rows := fc()
	attrs = append(attrs, slog.Any("rows", formatRows(rows)), slog.String("sql", sanitizeSQL(sql)))

	switch {
	case err != nil && l.options.LogLevel >= gormlogger.Error && (!errors.Is(err, gormlogger.ErrRecordNotFound) || !l.options.IgnoreRecordNotFoundError):
		attrs = append(attrs, slog.Any("error", err))
		l.logger.Log(ctx, slog.LevelError, "sql_error", attrs...)
	case l.options.SlowThreshold > 0 && elapsed > l.options.SlowThreshold && l.options.LogLevel >= gormlogger.Warn:
		attrs = append(attrs, slog.Bool("slow", true), slog.Duration("slow_threshold", l.options.SlowThreshold))
		l.logger.Log(ctx, slog.LevelWarn, "slow_sql", attrs...)
	case l.options.LogLevel >= gormlogger.Info:
		l.logger.Log(ctx, slog.LevelInfo, "sql", attrs...)
	}
}

func formatRows(rows int64) any {
	if rows < 0 {
		return "-"
	}
	return rows
}

func sanitizeSQL(sql string) string {
	if sql == "" {
		return sql
	}

	masked := sensitiveValuePattern.ReplaceAllString(sql, `${1}[REDACTED]${2}`)
	masked = redactInsertValues(masked)
	masked = redactAssignments(masked)
	masked = bearerTokenPattern.ReplaceAllString(masked, "Bearer [REDACTED]")
	return masked
}

func redactInsertValues(sql string) string {
	upperSQL := strings.ToUpper(sql)
	insertPrefix := "INSERT INTO "
	valuesMarker := " VALUES ("

	if !strings.HasPrefix(upperSQL, insertPrefix) {
		return sql
	}

	valuesIndex := strings.Index(upperSQL, valuesMarker)
	if valuesIndex == -1 {
		return sql
	}

	columnsStart := strings.Index(sql, "(")
	columnsEnd := strings.Index(sql[columnsStart:], ")")
	if columnsStart == -1 || columnsEnd == -1 {
		return sql
	}
	columnsEnd += columnsStart

	valuesStart := valuesIndex + len(valuesMarker)
	valuesEnd := strings.LastIndex(sql, ")")
	if valuesEnd <= valuesStart {
		return sql
	}

	columns := splitSQLCSV(sql[columnsStart+1 : columnsEnd])
	values := splitSQLCSV(sql[valuesStart:valuesEnd])
	if len(columns) != len(values) {
		return sql
	}

	for i, column := range columns {
		if isSensitiveColumn(column) {
			values[i] = redactSQLValue(values[i])
		}
	}

	return sql[:valuesStart] + strings.Join(values, ", ") + sql[valuesEnd:]
}

func redactAssignments(sql string) string {
	upperSQL := strings.ToUpper(sql)
	setIndex := strings.Index(upperSQL, " SET ")
	if setIndex == -1 {
		return sql
	}

	assignmentsStart := setIndex + len(" SET ")
	assignmentsEnd := len(sql)
	for _, marker := range []string{" WHERE ", " RETURNING ", " ORDER BY ", " LIMIT "} {
		if idx := strings.Index(upperSQL[assignmentsStart:], marker); idx != -1 {
			candidateEnd := assignmentsStart + idx
			if candidateEnd < assignmentsEnd {
				assignmentsEnd = candidateEnd
			}
		}
	}

	assignments := splitSQLCSV(sql[assignmentsStart:assignmentsEnd])
	for i, assignment := range assignments {
		parts := strings.SplitN(assignment, "=", 2)
		if len(parts) != 2 {
			continue
		}
		column := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if isSensitiveColumn(column) {
			assignments[i] = column + " = " + redactSQLValue(value)
			continue
		}
		assignments[i] = column + " = " + value
	}

	return sql[:assignmentsStart] + strings.Join(assignments, ", ") + sql[assignmentsEnd:]
}

func splitSQLCSV(input string) []string {
	var (
		parts    []string
		current  strings.Builder
		inSingle bool
		inDouble bool
	)

	for i := 0; i < len(input); i++ {
		ch := input[i]

		switch ch {
		case '\'':
			if !inDouble {
				if inSingle && i+1 < len(input) && input[i+1] == '\'' {
					current.WriteByte(ch)
					i++
					current.WriteByte(input[i])
					continue
				}
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case ',':
			if !inSingle && !inDouble {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
		}

		current.WriteByte(ch)
	}

	parts = append(parts, strings.TrimSpace(current.String()))
	return parts
}

func isSensitiveColumn(column string) bool {
	normalized := strings.ToLower(strings.Trim(column, "`\" []"))
	switch normalized {
	case "client_secret", "refresh_token", "access_token", "token", "secret":
		return true
	default:
		return false
	}
}

func redactSQLValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}

	if strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'") {
		return "'[REDACTED]'"
	}

	return "[REDACTED]"
}
