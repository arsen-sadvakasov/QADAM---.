package services

import (
	"context"
	"testing"

	"github.com/qadam/backend/internal/config"
	"github.com/qadam/backend/internal/repositories"
)

// TestDbgLogin воспроизводит login на реальной БД для диагностики 500
// (временный, удаляется после Phase demo-data).
func TestDbgLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	cfg := config.Load()
	ctx := context.Background()
	pool, err := config.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skipf("no db: %v", err)
	}
	defer pool.Close()

	userRepo := repositories.NewUserRepository(pool)
	authService := NewAuthService(userRepo, repositories.NewRefreshTokenRepository(pool), NewTokenService(cfg.JWTAccessSecret))

	result, err := authService.Login(ctx, "student01", "demo12345")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	t.Logf("login OK: %s (%s)", result.User.FullName, result.User.RoleKey)
}
