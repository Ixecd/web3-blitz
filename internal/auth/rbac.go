package auth

import (
	"context"
	"net/http"

	"github.com/Ixecd/web3-blitz/internal/db"
)

// HasPermission 检查用户是否拥有指定权限
func HasPermission(ctx context.Context, queries *db.Queries, userID int64, permission string) (bool, error) {
	perms, err := queries.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}

// RBACMiddleware 权限中间件，需要先经过 JWTMiddleware
func RBACMiddleware(queries *db.Queries, permission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ok, err := HasPermission(r.Context(), queries, claims.UserID, permission)
		if err != nil {
			http.Error(w, "权限查询失败", http.StatusInternalServerError)
			return
		}
		if !ok {
			http.Error(w, "权限不足", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}