package middleware

import (
	"context"
	"net/http"
	"uuid"
)

type requestIDKey struct{}

// GetRequestID 返回分配给这次请求的 ID，或者空字符串
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// RequestID 给每个请求分配一个 ID
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewV7().String()
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)

		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
