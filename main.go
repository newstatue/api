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
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/newstatue/api/internal/auth"
	"github.com/newstatue/api/internal/community"
	"github.com/newstatue/api/internal/config"
	"github.com/newstatue/api/internal/database"
	"github.com/newstatue/api/internal/middleware"
	"github.com/newstatue/api/internal/session"
	"github.com/newstatue/api/internal/storage"
	"github.com/newstatue/api/internal/user"
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
		slog.Error("Postgres 连接失败", "error", err)
		os.Exit(1)
	}
	defer postgres.Close()
	slog.Info("Postgres 连接成功")

	if err := database.Migrate(ctx, postgres); err != nil {
		slog.Error("Postgres 迁移失败", "error", err)
		os.Exit(1)
	}
	slog.Info("Postgres 迁移成功")

	valkey, err := database.NewValkey(ctx, cfg.ValkeyURL)
	if err != nil {
		slog.Error("Valkey 连接失败", "error", err)
		os.Exit(1)
	}
	defer valkey.Close()
	slog.Info("Valkey 连接成功")

	bucket, err := storage.NewBucket(ctx, cfg.SeaweedFSURL, cfg.SeaweedFSPublicEndpoint)
	if err != nil {
		slog.Error("SeaweedFS Bucket 连接失败", "error", err)
		os.Exit(1)
	}
	defer bucket.Close()
	slog.Info("SeaweedFS Bucket 连接成功")

	err = vips.Startup(nil)
	if err != nil {
		slog.Error("Vips 连接失败", "error", err)
		os.Exit(1)
	}
	defer vips.Shutdown()
	slog.Info("Vips 连接成功")

	mux := http.NewServeMux()

	humaConfig := huma.DefaultConfig("Evorsio API", "1.0.0")
	humaConfig.DocsRenderer = huma.DocsRendererScalar
	humaConfig.DocsRendererConfig = map[string]any{
		"hideClientButton":     true,    // 隐藏独立 API Client 入口
		"hideModels":           true,    // 隐藏独立的 Models 区域
		"hideDarkModeToggle":   true,    // 隐藏深浅色切换
		"documentDownloadType": "none",  // 隐藏 OpenAPI 下载按钮
		"showDeveloperTools":   "never", // 隐藏开发工具栏和 SDK 生成入口
		"agent": map[string]any{ // 隐藏 Ask AI 功能
			"disabled": true,
		},
	}

	humaConfig.DocsPath = "/api/docs"
	humaConfig.OpenAPIPath = "/api/openapi"
	humaConfig.SchemasPath = "/api/schemas"
	api := humago.New(mux, humaConfig)

	sessionManager := &session.Manager{
		Valkey: valkey,
		Secure: cfg.SessionSecure,
		TTL:    cfg.SessionTTL,
	}

	authenticated := huma.NewGroup(api)
	authenticated.UseMiddleware(sessionManager.Middleware(api, cfg.SessionSecure, true))

	optional := huma.NewGroup(api)
	optional.UseMiddleware(sessionManager.Middleware(api, cfg.SessionSecure, false))

	// Handlers
	authHandler := auth.Handler{DB: postgres, Session: sessionManager, Bucket: bucket}
	communityHandler := community.Handler{DB: postgres, Session: sessionManager}
	userHandler := user.Handler{DB: postgres, Session: sessionManager, Bucket: bucket}

	// 公开接口
	huma.Register(api, huma.Operation{
		OperationID:   "post-auth-register",
		Method:        http.MethodPost,
		Path:          "/api/auth/register",
		Summary:       "注册用户",
		DefaultStatus: http.StatusOK,
	}, authHandler.RegisterUser)
	huma.Register(api, huma.Operation{
		OperationID:   "post-auth-login",
		Method:        http.MethodPost,
		Path:          "/api/auth/login",
		Summary:       "用户登录",
		DefaultStatus: http.StatusOK,
	}, authHandler.LoginUser)

	huma.Register(api, huma.Operation{
		OperationID:   "get-communities",
		Method:        http.MethodGet,
		Path:          "/api/communities",
		Summary:       "获取社区列表",
		DefaultStatus: http.StatusOK,
	}, communityHandler.List)
	huma.Register(api, huma.Operation{
		OperationID:   "get-communities-by-slug-posts",
		Method:        http.MethodGet,
		Path:          "/api/communities/{slug}/posts",
		Summary:       "获取社区帖子列表",
		DefaultStatus: http.StatusOK,
	}, communityHandler.ListPosts)
	huma.Register(api, huma.Operation{
		OperationID:   "get-communities-by-slug-posts-by-post-id",
		Method:        http.MethodGet,
		Path:          "/api/communities/{slug}/posts/{postId}",
		Summary:       "获取帖子详情",
		DefaultStatus: http.StatusOK,
	}, communityHandler.GetPost)

	// 可选认证接口
	huma.Register(optional, huma.Operation{
		OperationID:   "get-communities-by-slug",
		Method:        http.MethodGet,
		Path:          "/api/communities/{slug}",
		Summary:       "获取社区详情",
		DefaultStatus: http.StatusOK,
	}, communityHandler.Get)

	// 必须认证的接口
	huma.Register(authenticated, huma.Operation{
		OperationID:   "get-auth-me",
		Method:        http.MethodGet,
		Path:          "/api/auth/me",
		Summary:       "获取当前用户",
		DefaultStatus: http.StatusOK,
	}, authHandler.GetCurrentUser)

	huma.Register(authenticated, huma.Operation{
		OperationID:   "post-communities",
		Method:        http.MethodPost,
		Path:          "/api/communities",
		Summary:       "创建社区",
		DefaultStatus: http.StatusOK,
	}, communityHandler.Create)

	huma.Register(authenticated, huma.Operation{
		OperationID:   "put-communities-by-slug-members-me",
		Method:        http.MethodPut,
		Path:          "/api/communities/{slug}/members/me",
		Summary:       "设置当前用户的社区加入状态",
		DefaultStatus: http.StatusNoContent,
	}, communityHandler.PutMembership)

	huma.Register(authenticated, huma.Operation{
		OperationID:   "post-communities-by-slug-posts",
		Method:        http.MethodPost,
		Path:          "/api/communities/{slug}/posts",
		Summary:       "创建帖子",
		DefaultStatus: http.StatusOK,
	}, communityHandler.CreatePost)

	huma.Register(authenticated, huma.Operation{
		OperationID:   "put-communities-posts-by-post-id-vote",
		Method:        http.MethodPut,
		Path:          "/api/communities/posts/{postId}/vote",
		Summary:       "设置帖子投票",
		DefaultStatus: http.StatusNoContent,
	}, communityHandler.PutPostVote)
	huma.Register(authenticated, huma.Operation{
		OperationID:   "put-communities-posts-by-post-id-bookmark",
		Method:        http.MethodPut,
		Path:          "/api/communities/posts/{postId}/bookmark",
		Summary:       "设置帖子收藏",
		DefaultStatus: http.StatusNoContent,
	}, communityHandler.PutPostBookmark)

	huma.Register(authenticated, huma.Operation{
		OperationID:   "put-user-avatar",
		Method:        http.MethodPut,
		Path:          "/api/user/me/avatar",
		Summary:       "更新当前用户头像",
		MaxBodyBytes:  5 << 20,
		DefaultStatus: http.StatusNoContent,
	}, userHandler.PutAvatar)

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: middleware.RequestID(mux),
	}
	slog.Info("服务器运行中", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("服务器运行失败", "error", err)
		os.Exit(1)
	}
}
