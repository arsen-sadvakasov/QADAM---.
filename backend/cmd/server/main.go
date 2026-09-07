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
	roleRepo := repositories.NewRoleRepository(pool)
	tokenService := services.NewTokenService(cfg.JWTAccessSecret)
	authService := services.NewAuthService(userRepo, refreshTokenRepo, tokenService)

	// Репозитории учебной структуры (Phase 3 — Database Core).
	specialtyRepo := repositories.NewSpecialtyRepository(pool)
	courseRepo := repositories.NewCourseRepository(pool)
	groupRepo := repositories.NewGroupRepository(pool)
	subjectRepo := repositories.NewSubjectRepository(pool)
	roomRepo := repositories.NewRoomRepository(pool)
	teacherRepo := repositories.NewTeacherRepository(pool)
	_ = repositories.NewStudentRepository(pool) // студенческий CRUD — Phase 9 (Curator Module)
	_ = specialtyRepo                           // CRUD специальностей/курсов не входит в API Plan Phase 5 (только справочные данные)
	_ = courseRepo

	// Phase 4 — Schedule: вычисление расписания на день/неделю/месяц из
	// schedule_templates + карточка занятия (FR-2, FR-3 спецификации).
	scheduleTemplateRepo := repositories.NewScheduleTemplateRepository(pool)
	scheduleService := services.NewScheduleService(scheduleTemplateRepo)

	// Phase 5 — Admin Panel (Core CRUD): пользователи, группы, кабинеты,
	// предметы, преподаватели, кураторы, управление расписанием.
	userAdminService := services.NewUserAdminService(userRepo, roleRepo)
	teacherAdminService := services.NewTeacherAdminService(userAdminService, teacherRepo)
	scheduleAdminService := services.NewScheduleAdminService(scheduleTemplateRepo)

	authHandler := handlers.NewAuthHandler(authService)
	usersHandler := handlers.NewUsersHandler(userRepo, userAdminService)
	schedulesHandler := handlers.NewSchedulesHandler(scheduleService, scheduleAdminService)
	groupsHandler := handlers.NewGroupsHandler(groupRepo)
	roomsHandler := handlers.NewRoomsHandler(roomRepo)
	subjectsHandler := handlers.NewSubjectsHandler(subjectRepo)
	teachersHandler := handlers.NewTeachersHandler(teacherAdminService)
	curatorsHandler := handlers.NewCuratorsHandler(userAdminService)

	loginRateLimiter := middleware.NewRateLimiter(10, time.Minute)
	authMiddleware := middleware.Auth(tokenService)
	adminOnly := middleware.RequireRole(models.RoleAdmin)
	adminOrCurator := middleware.RequireRole(models.RoleAdmin, models.RoleCurator)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlers.Health)

	// --- Auth (Phase 2) ---
	mux.Handle("POST /api/v1/auth/login", loginRateLimiter.Middleware(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.Handle("POST /api/v1/auth/logout", authMiddleware(http.HandlerFunc(authHandler.Logout)))

	// --- Users (Phase 2 + Phase 5) ---
	mux.Handle("GET /api/v1/users/me", authMiddleware(http.HandlerFunc(usersHandler.Me)))
	mux.Handle("GET /api/v1/users", authMiddleware(adminOnly(http.HandlerFunc(usersHandler.List))))
	mux.Handle("GET /api/v1/users/{id}", authMiddleware(adminOnly(http.HandlerFunc(usersHandler.Get))))
	mux.Handle("POST /api/v1/users", authMiddleware(adminOnly(http.HandlerFunc(usersHandler.Create))))
	mux.Handle("PATCH /api/v1/users/{id}", authMiddleware(adminOnly(http.HandlerFunc(usersHandler.Update))))
	mux.Handle("DELETE /api/v1/users/{id}", authMiddleware(adminOnly(http.HandlerFunc(usersHandler.Block))))

	// --- Schedules (Phase 4 read + Phase 5 admin CRUD) ---
	mux.Handle("GET /api/v1/schedules", authMiddleware(http.HandlerFunc(schedulesHandler.GetSchedule)))
	mux.Handle("GET /api/v1/schedules/lesson/{id}", authMiddleware(http.HandlerFunc(schedulesHandler.GetLessonDetails)))
	mux.Handle("POST /api/v1/schedules", authMiddleware(adminOnly(http.HandlerFunc(schedulesHandler.CreateTemplate))))
	mux.Handle("PATCH /api/v1/schedules/{id}", authMiddleware(adminOnly(http.HandlerFunc(schedulesHandler.UpdateTemplate))))
	mux.Handle("DELETE /api/v1/schedules/{id}", authMiddleware(adminOnly(http.HandlerFunc(schedulesHandler.DeleteTemplate))))

	// --- Groups (Phase 5) ---
	mux.Handle("GET /api/v1/groups", authMiddleware(http.HandlerFunc(groupsHandler.List)))
	mux.Handle("GET /api/v1/groups/{id}", authMiddleware(http.HandlerFunc(groupsHandler.Get)))
	mux.Handle("POST /api/v1/groups", authMiddleware(adminOnly(http.HandlerFunc(groupsHandler.Create))))
	mux.Handle("PATCH /api/v1/groups/{id}", authMiddleware(adminOnly(http.HandlerFunc(groupsHandler.Update))))
	mux.Handle("DELETE /api/v1/groups/{id}", authMiddleware(adminOnly(http.HandlerFunc(groupsHandler.Delete))))

	// --- Subjects (Phase 5) ---
	mux.Handle("GET /api/v1/subjects", authMiddleware(http.HandlerFunc(subjectsHandler.List)))
	mux.Handle("GET /api/v1/subjects/{id}", authMiddleware(http.HandlerFunc(subjectsHandler.Get)))
	mux.Handle("POST /api/v1/subjects", authMiddleware(adminOnly(http.HandlerFunc(subjectsHandler.Create))))
	mux.Handle("PATCH /api/v1/subjects/{id}", authMiddleware(adminOnly(http.HandlerFunc(subjectsHandler.Update))))

	// --- Rooms (Phase 5) ---
	mux.Handle("GET /api/v1/rooms", authMiddleware(http.HandlerFunc(roomsHandler.List)))
	mux.Handle("GET /api/v1/rooms/{id}", authMiddleware(http.HandlerFunc(roomsHandler.Get)))
	mux.Handle("POST /api/v1/rooms", authMiddleware(adminOnly(http.HandlerFunc(roomsHandler.Create))))
	mux.Handle("PATCH /api/v1/rooms/{id}", authMiddleware(adminOnly(http.HandlerFunc(roomsHandler.Update))))

	// --- Teachers (Phase 5) ---
	// GET /teachers (список): Admin, Curator (раздел 35 API Plan). GET /teachers/{id}
	// (детали): все авторизованные, но с ограниченным набором полей для student
	// (см. roleAllowsTeacherContacts в TeachersHandler).
	mux.Handle("GET /api/v1/teachers", authMiddleware(adminOrCurator(http.HandlerFunc(teachersHandler.List))))
	mux.Handle("GET /api/v1/teachers/{id}", authMiddleware(http.HandlerFunc(teachersHandler.Get)))
	mux.Handle("POST /api/v1/teachers", authMiddleware(adminOnly(http.HandlerFunc(teachersHandler.Create))))
	mux.Handle("PATCH /api/v1/teachers/{id}/subjects", authMiddleware(adminOnly(http.HandlerFunc(teachersHandler.AssignSubject))))

	// --- Curators (Phase 5) ---
	mux.Handle("GET /api/v1/curators", authMiddleware(adminOnly(http.HandlerFunc(curatorsHandler.List))))
	mux.Handle("POST /api/v1/curators", authMiddleware(adminOnly(http.HandlerFunc(curatorsHandler.Create))))
	mux.Handle("PATCH /api/v1/curators/{id}", authMiddleware(adminOnly(http.HandlerFunc(curatorsHandler.Update))))

	var h http.Handler = mux
	h = middleware.Logging(h)

	addr := ":" + cfg.Port
	log.Printf("QADAM backend starting (env=%s) on %s", cfg.AppEnv, addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
