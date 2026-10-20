package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cnzakii/e5-renewal/internal/app"
	"gopkg.in/natefinch/lumberjack.v2"
)

func TestLogFormatsPreserveFieldsAndLevels(t *testing.T) {
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			var output bytes.Buffer
			logger := newLogger(format, slog.LevelInfo, &output)
			logger.Debug("hidden diagnostic")
			logger.Info("task completed", "run_id", 42)
			if strings.Contains(output.String(), "hidden diagnostic") {
				t.Fatal("log level did not filter DEBUG records")
			}

			if format == "json" {
				var record map[string]any
				if err := json.Unmarshal(output.Bytes(), &record); err != nil {
					t.Fatal(err)
				}

				if record["run_id"] != float64(42) || record["msg"] != "task completed" {
					t.Fatal("JSON output lost structured fields")
				}
			} else if !strings.Contains(output.String(), `msg="task completed" run_id=42`) {
				t.Fatal("text output lost readable structured fields")
			}
		})
	}
}

func TestOptionalLogFile(t *testing.T) {
	var console bytes.Buffer
	output, file, err := newLogOutput(app.Config{}, &console)
	if err != nil || file != nil || output != &console {
		t.Fatal("default logging created a file destination")
	}

	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "logs", "server.log")
			config := app.Config{LogFile: filename, LogFormat: format}
			for _, message := range []string{"before restart", "after restart"} {
				console.Reset()
				output, file, err := newLogOutput(config, &console)
				if err != nil {
					t.Fatal(err)
				}

				newLogger(format, slog.LevelInfo, output).Info(message, "run_id", 42)
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}

				data, err := os.ReadFile(filename)
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.HasSuffix(data, console.Bytes()) || !bytes.Contains(data, []byte("before restart")) {
					t.Fatal("file and console records differ or old records were overwritten")
				}
			}

			info, err := os.Stat(filename)
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Fatal("log file is not owner-only")
			}
		})
	}

	if _, _, err := newLogOutput(app.Config{LogFile: t.TempDir()}, &console); err == nil {
		t.Fatal("unwritable log destination was accepted")
	}
}

func TestFileWriteFailureKeepsConsoleRecord(t *testing.T) {
	var console, diagnostics bytes.Buffer
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	file := &lumberjack.Logger{Filename: filepath.Join(blocked, "server.log")}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	output := io.MultiWriter(&console, logFileWriter{file, newLogger("text", slog.LevelError, &diagnostics)})
	newLogger("text", slog.LevelInfo, output).Info("visible despite file failure")
	if !strings.Contains(console.String(), "visible despite file failure") ||
		!strings.Contains(diagnostics.String(), "log file write failed") {
		t.Fatal("file failure hid the console record or its diagnostic")
	}
}

func TestLogFileRotatesAtSizeLimit(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "server.log")
	previous, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}

	if err := previous.Truncate(10 << 20); err != nil {
		t.Fatal(err)
	}

	if err := previous.Close(); err != nil {
		t.Fatal(err)
	}

	var console bytes.Buffer
	output, file, err := newLogOutput(app.Config{LogFile: filename}, &console)
	if err != nil {
		t.Fatal(err)
	}

	newLogger("text", slog.LevelInfo, output).Info("after rotation")
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(filepath.Dir(filename), "server*.log"))
	if err != nil || len(files) != 2 {
		t.Fatal("full log file was not retained as a rotated backup")
	}

	current, err := os.ReadFile(filename)
	if err != nil || !bytes.Equal(current, console.Bytes()) {
		t.Fatal("new records did not move to a fresh current log")
	}
}
