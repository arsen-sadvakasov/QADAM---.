// Command server запускает HTTP API сервиса QADAM.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/qadam/backend/internal/config"
	"github.com/qadam/backend/internal/handlers"
	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

func main() {
	cfg := config.Load()

	if err := config.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	ctx := context.Background()
	pool, err := config.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	userRepo := repositories.NewUserRepository(pool)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(pool)
	tokenService := services.NewTokenService(cfg.JWTAccessSecret)
	authService := services.NewAuthService(userRepo, refreshTokenRepo, tokenService)

	authHandler := handlers.NewAuthHandler(authService)
	usersHandler := handlers.NewUsersHandler(userRepo)

	loginRateLimiter := middleware.NewRateLimiter(10, time.Minute)
	authMiddleware := middleware.Auth(tokenService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlers.Health)

	mux.Handle("POST /api/v1/auth/login", loginRateLimiter.Middleware(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.Handle("POST /api/v1/auth/logout", authMiddleware(http.HandlerFunc(authHandler.Logout)))

	mux.Handle("GET /api/v1/users/me", authMiddleware(http.HandlerFunc(usersHandler.Me)))

	// Пример защищённого admin-only маршрута — демонстрирует использование
	// RequireRole; полноценный CRUD появится в Phase 5 (Admin Panel).
	mux.Handle("GET /api/v1/admin/ping", authMiddleware(middleware.RequireRole(models.RoleAdmin)(http.HandlerFunc(handlers.Health))))

	var h http.Handler = mux
	h = middleware.Logging(h)

	addr := ":" + cfg.Port
	log.Printf("QADAM backend starting (env=%s) on %s", cfg.AppEnv, addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
