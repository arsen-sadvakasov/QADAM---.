package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/services"
)

type contextKey string

const (
	userIDContextKey contextKey = "userID"
	userRoleContextKey contextKey = "userRole"
)

// Auth проверяет JWT access-токен из заголовка Authorization: Bearer <token>
// и, при успехе, помещает userID и роль пользователя в контекст запроса
// (раздел 16 спецификации: каждый запрос проходит middleware авторизации).
func Auth(tokens *services.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				writeUnauthorized(w, "missing or malformed Authorization header")
				return
			}

			claims, err := tokens.ParseAccessToken(parts[1])
			if err != nil {
				writeUnauthorized(w, "invalid or expired access token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, userRoleContextKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole ограничивает доступ к хендлеру только перечисленным ролям
// (базовый RBAC по 4 ролям — раздел 4, 17: "в MVP использовать простой
// RBAC по ролям, без гибких точечных permissions").
func RequireRole(roles ...models.RoleKey) func(http.Handler) http.Handler {
	allowed := make(map[models.RoleKey]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := RoleFromContext(r.Context())
			if !ok || !allowed[role] {
				writeForbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromContext извлекает ID авторизованного пользователя, помещённый
// middleware Auth.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDContextKey).(string)
	return v, ok
}

// RoleFromContext извлекает роль авторизованного пользователя, помещённую
// middleware Auth.
func RoleFromContext(ctx context.Context) (models.RoleKey, bool) {
	v, ok := ctx.Value(userRoleContextKey).(models.RoleKey)
	return v, ok
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func writeForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "insufficient permissions"})
}
