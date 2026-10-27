package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
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
		log.Fatal("加载配置失败", err)
	}
	log.Print("加载配置成功")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	postgres, err := database.NewPostgres(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatal("Postgres连接失败", err)
	}
	defer postgres.Close()
	log.Print("Postgres连接成功")

	if err := database.Migrate(ctx, postgres); err != nil {
		log.Fatal("Postgres迁移失败", err)
	}
	log.Print("Postgres迁移成功")

	valkey, err := database.NewValkey(ctx, cfg.ValkeyURL)
	if err != nil {
		log.Fatal("Valkey连接失败", err)
	}
	defer valkey.Close()
	log.Print("Valkey连接成功")

	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("示例API", "0.0.1"))

	sessionManager := &session.Manager{Valkey: valkey}

	authHandler := auth.Handler{DB: postgres, Session: sessionManager}
	authHandler.RegisterRoutes(api)

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	log.Printf("服务器运行于: %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
