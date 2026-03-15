package main

import (
	"context"
	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/handlers"
	"e5-renewal/backend/middleware"
	"e5-renewal/backend/services/executor"
	"e5-renewal/backend/services/login"
	"e5-renewal/backend/services/oauth"
	"e5-renewal/backend/services/scheduler"
	"e5-renewal/backend/spa"
	"embed"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed all:static/dist
var frontendFS embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	config.MustInit()

	cfg := config.Get()
	if err := database.Init(cfg.Database.Path); err != nil {
		slog.Error("init database failed", "error", err)
		os.Exit(1)
	}

	database.MustInitEncryption(cfg.Security.EncryptionKey)

	login.MustInit(cfg.Security.LoginKey)

	oauthSvc := oauth.NewService(nil)
	execRng := rand.New(rand.NewSource(time.Now().UnixNano()))
	schedRng := rand.New(rand.NewSource(time.Now().UnixNano() + 1))
	exec := executor.New(oauthSvc, execRng)
	sched := scheduler.New(exec, schedRng)

	r := gin.New()
	r.Use(middleware.SlogLogger(), middleware.SlogRecovery())
	_ = r.SetTrustedProxies(nil)
	handlers.RegisterHealthRoutes(r)
	handlers.RegisterAuthRoutes(r)
	handlers.RegisterAccountRoutes(r, sched)
	handlers.RegisterSettingRoutes(r)
	handlers.RegisterDashboardRoutes(r)
	handlers.RegisterLogRoutes(r)
	handlers.RegisterOAuthRoutes(r)
	spa.RegisterSPA(r, cfg.Server.PathPrefix, frontendFS, cfg.Server.PathPrefix)

	go sched.Start(context.Background())

	addr := cfg.Addr()
	slog.Info("server starting", "addr", addr)

	if cfg.Server.TLSCert != "" && cfg.Server.TLSKey != "" {
		slog.Info("TLS enabled")
		if err := r.RunTLS(addr, cfg.Server.TLSCert, cfg.Server.TLSKey); err != nil {
			slog.Error("run server (TLS) failed", "error", err)
			os.Exit(1)
		}
	} else {
		if err := r.Run(addr); err != nil {
			slog.Error("run server failed", "error", err)
			os.Exit(1)
		}
	}
}
