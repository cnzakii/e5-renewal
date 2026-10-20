package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/cnzakii/e5-renewal/internal/app"
	"gopkg.in/natefinch/lumberjack.v2"
)

func newLogger(format string, level slog.Level, output io.Writer) *slog.Logger {
	options := &slog.HandlerOptions{Level: level}

	var handler slog.Handler = slog.NewTextHandler(output, options)
	if format == "json" {
		handler = slog.NewJSONHandler(output, options)
	}

	return slog.New(handler)
}

func newLogOutput(config app.Config, console io.Writer) (io.Writer, *lumberjack.Logger, error) {
	if config.LogFile == "" {
		return console, nil, nil
	}

	if err := os.MkdirAll(filepath.Dir(config.LogFile), 0o700); err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}

	// Validate the destination before the rotating writer can rename an existing path.
	probe, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}

	if err := errors.Join(probe.Chmod(0o600), probe.Close()); err != nil {
		return nil, nil, err
	}

	file := &lumberjack.Logger{Filename: config.LogFile, MaxSize: 10, MaxBackups: 3}
	errorLog := newLogger(config.LogFormat, slog.LevelError, os.Stderr)

	return io.MultiWriter(console, logFileWriter{file, errorLog}), file, nil
}

type logFileWriter struct {
	file   *lumberjack.Logger
	errors *slog.Logger
}

func (w logFileWriter) Write(data []byte) (int, error) {
	n, err := w.file.Write(data)
	if err != nil {
		// slog discards write errors; report them through a separate stderr handler.
		w.errors.Error("log file write failed", "error", err)
	}

	return n, err
}
