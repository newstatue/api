package middleware

import (
	"net/http"

	"golang.org/x/time/rate"
)

func RateLimit(limiter *rate.Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "请求过于频繁", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
