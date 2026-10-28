package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/newstatue/api/internal/auth"
	"github.com/newstatue/api/internal/config"
	"github.com/newstatue/api/internal/database"
	"github.com/newstatue/api/internal/session"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("加载配置失败", "error", err)
		os.Exit(1)
	}
	slog.Info("加载配置成功")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	postgres, err := database.NewPostgres(ctx, cfg.PostgresURL)
	if err != nil {
		slog.Error("Postgres连接失败", "error", err)
		os.Exit(1)
	}
	defer postgres.Close()
	slog.Info("Postgres连接成功")

	if err := database.Migrate(ctx, postgres); err != nil {
		slog.Error("Postgres迁移失败", "error", err)
		os.Exit(1)
	}
	slog.Info("Postgres迁移成功")

	valkey, err := database.NewValkey(ctx, cfg.ValkeyURL)
	if err != nil {
		slog.Error("Valkey连接失败", "error", err)
		os.Exit(1)
	}
	defer valkey.Close()
	slog.Info("Valkey连接成功")

	mux := http.NewServeMux()

	humaConfig := huma.DefaultConfig("Evorsio API", "1.0.0")
	humaConfig.DocsPath = "/api/docs"
	humaConfig.OpenAPIPath = "/api/openapi.json"
	humaConfig.SchemasPath = "/api/schemas"
	api := humago.New(mux, humaConfig)

	sessionManager := &session.Manager{
		Valkey: valkey,
		Secure: cfg.SessionSecure,
		TTL:    cfg.SessionTTL,
	}

	authenticated := huma.NewGroup(api)
	authenticated.UseMiddleware(sessionManager.Middleware(api, cfg.SessionSecure))

	authHandler := auth.Handler{DB: postgres, Session: sessionManager}
	huma.Post(api, "/api/auth/register", authHandler.RegisterUser)
	huma.Post(api, "/api/auth/login", authHandler.LoginUser)

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	slog.Info("服务器运行中", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("服务器运行失败", "error", err)
		os.Exit(1)
	}
}
