package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsKey contextKey = "claims"

// JWTMiddleware 验证 Authorization: Bearer <token>
func JWTMiddleware(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			http.Error(w, "缺少 Authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Authorization 格式错误，应为 Bearer <token>", http.StatusUnauthorized)
			return
		}

		claims, err := ParseToken(parts[1], secret)
		if err != nil {
			http.Error(w, "token 无效: "+err.Error(), http.StatusUnauthorized)
			return
		}

		// 把 claims 注入 context，handler 里可以取出
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// GetClaims 从 context 取出 claims
func GetClaims(r *http.Request) *Claims {
	claims, _ := r.Context().Value(claimsKey).(*Claims)
	return claims
}
