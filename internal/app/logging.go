package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

func databaseAttributes(err error) []slog.Attr {
	attrs := []slog.Attr{slog.String("error_type", fmt.Sprintf("%T", err))}

	var state interface{ SQLState() string }
	if errors.As(err, &state) {
		attrs = append(attrs, slog.String("sqlstate", state.SQLState()))
	}

	var mysqlError *mysqlDriver.MySQLError
	if errors.As(err, &mysqlError) {
		attrs = append(attrs, slog.Int("database_code", int(mysqlError.Number)))
	}

	var sqliteError interface{ Code() int }
	if errors.As(err, &sqliteError) {
		attrs = append(attrs, slog.Int("database_code", sqliteError.Code()))
	}

	return attrs
}

// Driver messages can contain SQL values, so diagnostics retain codes rather than raw errors.
func logDatabaseError(ctx context.Context, message string, err error, attrs ...slog.Attr) {
	attrs = append(attrs, databaseAttributes(err)...)
	attrs = append(attrs, slog.String("request_id", middleware.GetReqID(ctx)))
	slog.LogAttrs(ctx, slog.LevelError, message, attrs...)
}

func logPanic(ctx context.Context, message string, recovered any, attrs ...slog.Attr) {
	attrs = append(
		attrs,
		slog.String("panic_type", fmt.Sprintf("%T", recovered)),
		slog.String("stack", string(debug.Stack())),
	)
	slog.LogAttrs(ctx, slog.LevelError, message, attrs...)
}

func observeHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), middleware.RequestIDKey, rand.Text()))
		started := time.Now()
		response := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		requestID := middleware.GetReqID(r.Context())
		response.Header().Set("X-Request-ID", requestID)
		defer func() {
			if recovered := recover(); recovered != nil {
				if aborted, ok := recovered.(error); ok && errors.Is(aborted, http.ErrAbortHandler) {
					panic(http.ErrAbortHandler)
				}

				logPanic(r.Context(), "request panicked", recovered, slog.String("request_id", requestID))
				if response.Status() != 0 {
					panic(http.ErrAbortHandler)
				}

				fail(
					response,
					http.StatusInternalServerError,
					newError("internal_error", "The server encountered an unexpected error."),
				)
			}

			status := response.Status()
			if status == 0 {
				status = http.StatusOK
			}

			level := slog.LevelDebug
			switch {
			case status >= 500:
				level = slog.LevelError
			case status >= 400:
				level = slog.LevelWarn
			case r.Method != http.MethodGet && r.Method != http.MethodHead:
				level = slog.LevelInfo
			}

			// Query strings include OAuth codes; bodies and authorization headers are never logged.
			slog.LogAttrs(
				r.Context(),
				level,
				"http request",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int("bytes", response.BytesWritten()),
				slog.Int64("duration_ms", time.Since(started).Milliseconds()),
			)
		}()
		next.ServeHTTP(response, r)
	})
}
