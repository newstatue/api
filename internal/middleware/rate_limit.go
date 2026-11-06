package middleware

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/time/rate"
)

func RateLimit(api huma.API, limiter *rate.Limiter) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if !limiter.Allow() {
			_ = huma.WriteErr(api, ctx, http.StatusTooManyRequests, "请求过于频繁")
			return
		}

		next(ctx)
	}
}
