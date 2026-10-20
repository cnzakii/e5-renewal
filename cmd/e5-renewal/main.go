package main

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/cnzakii/e5-renewal/internal/app"
	"github.com/cnzakii/e5-renewal/web"
)

var version = "dev"

func main() {
	slog.SetDefault(newLogger("text", slog.LevelInfo, os.Stdout))
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	reset := flag.Bool(
		"reset-password",
		false,
		"Generate a new administrator password while the server is stopped",
	)
	showVersion := flag.Bool("version", false, "Print version")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return nil
	}

	config, err := app.LoadConfig()
	if err != nil {
		return err
	}

	if *reset {
		// Recovery output belongs to this command's console.
		config.LogFile = ""
	}

	output, logFile, err := newLogOutput(config, os.Stdout)
	if err != nil {
		return err
	}

	if logFile != nil {
		defer func() {
			if err := logFile.Close(); err != nil {
				newLogger(config.LogFormat, slog.LevelWarn, os.Stderr).Warn("log file close failed", "error", err)
			}
		}()
	}

	slog.SetDefault(newLogger(config.LogFormat, config.LogLevel, output))
	assets, err := fs.Sub(web.Files, "dist")
	if err != nil {
		return err
	}

	if !*reset {
		if _, err := fs.Stat(assets, "index.html"); err != nil {
			return fmt.Errorf("build the frontend before compiling: %w", err)
		}
	}

	db, err := app.OpenDB(config)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Warn("database close failed", "error_type", fmt.Sprintf("%T", err))
		}
	}()
	if *reset {
		password := rand.Text()
		if err := app.ResetPassword(db, password); err != nil {
			return err
		}

		// Credential notices go only to the console, never the optional application log file.
		newLogger(config.LogFormat, slog.LevelInfo, os.Stdout).
			Info("administrator password reset; start the server to use it", "administrator_password", password)

		return nil
	}

	password, err := app.Initialize(db, config.InitialPassword)
	if err != nil {
		return err
	}

	if password != "" {
		// Keep this console-only notice visible even when ordinary INFO records are disabled.
		newLogger(config.LogFormat, slog.LevelInfo, os.Stdout).
			Info("administrator initialized; change the password after signing in", "administrator_password", password)
	}

	microsoft := app.NewMicrosoft()
	defer microsoft.Client.CloseIdleConnections()
	runner, err := app.NewRunner(db, microsoft)
	if err != nil {
		return err
	}
	defer runner.Close()
	api, err := app.NewServer(config, db, runner, microsoft, assets)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              config.Address,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	if err := runner.Start(); err != nil {
		return errors.Join(err, listener.Close())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	slog.Info("server started", "version", version, "address", config.Address, "public_url", config.PublicURL)
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			return err
		}
	}

	return nil
}
