package session

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const CookieName = "session"

type contextKey struct{}

func FromContext(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(contextKey{}).(*Session)
	return session, ok && session != nil
}

// Middleware 要求登录
func (m *Manager) Middleware(api huma.API, secure bool) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		cookie, err := huma.ReadCookie(ctx, CookieName)
		if err != nil || cookie.Value == "" {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "请先登录")
			return
		}

		session, err := m.Get(ctx.Context(), cookie.Value)
		if errors.Is(err, ErrNotFound) {
			appendCookie(ctx, "", -1, secure)
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "会话已过期，请重新登录")
			return
		}
		if err != nil || session == nil {
			slog.ErrorContext(ctx.Context(), "读取会话失败", "error", err)
			_ = huma.WriteErr(
				api, ctx,
				http.StatusServiceUnavailable,
				"会话服务暂不可用",
			)
			return
		}

		// Get 已续期 Redis，这里同步续期浏览器 Cookie。
		appendCookie(ctx, cookie.Value, int(m.TTL.Seconds()), secure)

		next(huma.WithValue(ctx, contextKey{}, session))
	}
}

func appendCookie(ctx huma.Context, sessionID string, maxAge int, secure bool) {
	cookie := NewCookie(sessionID, maxAge, secure)
	ctx.AppendHeader("Set-Cookie", cookie.String())
}

func NewCookie(sessionID string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}
