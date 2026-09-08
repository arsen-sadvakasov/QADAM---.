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
	"github.com/qadam/backend/internal/storage"
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
	_ = specialtyRepo                           // CRUD специальностей/курсов не входит в API Plan Phase 5 (только справочные данные)
	_ = courseRepo

	// Phase 4 — Schedule: вычисление расписания на день/неделю/месяц из
	// schedule_templates + карточка занятия (FR-2, FR-3 спецификации).
	// Phase 6 — замены накладываются поверх шаблона при вычислении.
	scheduleTemplateRepo := repositories.NewScheduleTemplateRepository(pool)
	scheduleChangeRepo := repositories.NewScheduleChangeRepository(pool)
	scheduleService := services.NewScheduleService(scheduleTemplateRepo, scheduleChangeRepo)

	// Phase 14 — Security Hardening: журнал аудита мутаций администраторов
	// (раздел 29 спецификации). best-effort запись, чтение — Admin.
	auditRepo := repositories.NewAuditLogRepository(pool)
	auditService := services.NewAuditService(auditRepo)

	// Phase 5 — Admin Panel (Core CRUD): пользователи, группы, кабинеты,
	// предметы, преподаватели, кураторы, управление расписанием.
	userAdminService := services.NewUserAdminService(userRepo, roleRepo)
	userAdminService.SetAudit(auditService)
	teacherAdminService := services.NewTeacherAdminService(userAdminService, teacherRepo)
	scheduleAdminService := services.NewScheduleAdminService(scheduleTemplateRepo)
	scheduleAdminService.SetAudit(auditService)

	// Phase 8 — Notifications: уведомления о заменах расписания и новых
	// материалах (раздел 19 спецификации), системные уведомления от Admin.
	notificationRepo := repositories.NewNotificationRepository(pool)
	notificationService := services.NewNotificationService(notificationRepo)

	// Phase 9 — Curator Module: управление студентами группы с проверкой
	// прав "куратор — своя группа" (groups.curator_id), admin — все
	// (разделы 17, 22 спецификации).
	studentRepo := repositories.NewStudentRepository(pool)
	curatorService := services.NewCuratorService(studentRepo, groupRepo)

	// Phase 11 — Search: глобальный поиск по ключевым полям с учётом прав
	// (ILIKE/pg_trgm, раздел 25 спецификации).
	searchRepo := repositories.NewSearchRepository(pool)
	searchService := services.NewSearchService(searchRepo)

	// Phase 6 — Schedule Changes (Замены): точечные замены/отмены/переносы
	// занятий на конкретную дату, накладываемые поверх шаблона (раздел 20).
	scheduleChangeService := services.NewScheduleChangeService(scheduleChangeRepo, scheduleTemplateRepo, notificationService)
	scheduleChangeService.SetAudit(auditService)

	// Phase 7 — Materials: учебные материалы. Файловое хранилище — через
	// абстракцию FileStorage (раздел 14): локальная ФС в разработке;
	// MinIO (S3) подключается заменой реализации без изменения остального кода.
	fileStorage, err := storage.NewLocalFileStorage(cfg.MaterialsDir)
	if err != nil {
		log.Fatalf("failed to init file storage: %v", err)
	}
	materialRepo := repositories.NewMaterialRepository(pool)
	materialService := services.NewMaterialService(materialRepo, fileStorage, notificationService)

	authHandler := handlers.NewAuthHandler(authService)
	usersHandler := handlers.NewUsersHandler(userRepo, userAdminService)
	schedulesHandler := handlers.NewSchedulesHandler(scheduleService, scheduleAdminService)
	groupsHandler := handlers.NewGroupsHandler(groupRepo)
	roomsHandler := handlers.NewRoomsHandler(roomRepo)
	subjectsHandler := handlers.NewSubjectsHandler(subjectRepo)
	teachersHandler := handlers.NewTeachersHandler(teacherAdminService)
	curatorsHandler := handlers.NewCuratorsHandler(userAdminService)
	studentsHandler := handlers.NewStudentsHandler(curatorService)
	searchHandler := handlers.NewSearchHandler(searchService)
	localesHandler := handlers.NewLocalesHandler(userAdminService, userRepo)
	auditLogsHandler := handlers.NewAuditLogsHandler(auditService)
	scheduleChangesHandler := handlers.NewScheduleChangesHandler(scheduleChangeService)
	materialsHandler := handlers.NewMaterialsHandler(materialService)
	notificationsHandler := handlers.NewNotificationsHandler(notificationService)

	loginRateLimiter := middleware.NewRateLimiter(10, time.Minute)
	authMiddleware := middleware.Auth(tokenService)
	adminOnly := middleware.RequireRole(models.RoleAdmin)
	adminOrCurator := middleware.RequireRole(models.RoleAdmin, models.RoleCurator)
	adminOrTeacher := middleware.RequireRole(models.RoleAdmin, models.RoleTeacher)

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

	// --- Admin: audit logs (Phase 14, раздел 29) ---
	// Только чтение, с фильтрами. Admin only.
	mux.Handle("GET /api/v1/admin/audit-logs", authMiddleware(adminOnly(http.HandlerFunc(auditLogsHandler.List))))

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

	// --- Schedule Changes (Phase 6) ---
	// Замены расписания: создание, просмотр, обновление, удаление.
	// Доступ к изменению — только Admin. Просмотр — Admin и затронутые пользователи
	// (реализация фильтрации по ролям будет доработана в Phase 9 для студентов/кураторов).
	mux.Handle("GET /api/v1/schedule-changes", authMiddleware(http.HandlerFunc(scheduleChangesHandler.List)))
	mux.Handle("GET /api/v1/schedule-changes/{id}", authMiddleware(http.HandlerFunc(scheduleChangesHandler.Get)))
	mux.Handle("POST /api/v1/schedule-changes", authMiddleware(adminOnly(http.HandlerFunc(scheduleChangesHandler.Create))))
	mux.Handle("PATCH /api/v1/schedule-changes/{id}", authMiddleware(adminOnly(http.HandlerFunc(scheduleChangesHandler.Update))))
	mux.Handle("DELETE /api/v1/schedule-changes/{id}", authMiddleware(adminOnly(http.HandlerFunc(scheduleChangesHandler.Delete))))

	// --- Materials (Phase 7) ---
	// Учебные материалы: просмотр — все авторизованные; создание/загрузка/удаление
	// — Teacher (свои материалы) и Admin (раздел 21 спецификации).
	mux.Handle("GET /api/v1/materials", authMiddleware(http.HandlerFunc(materialsHandler.List)))
	mux.Handle("GET /api/v1/materials/{id}", authMiddleware(http.HandlerFunc(materialsHandler.Get)))
	mux.Handle("POST /api/v1/materials", authMiddleware(adminOrTeacher(http.HandlerFunc(materialsHandler.Create))))
	mux.Handle("PATCH /api/v1/materials/{id}", authMiddleware(adminOrTeacher(http.HandlerFunc(materialsHandler.Update))))
	mux.Handle("DELETE /api/v1/materials/{id}", authMiddleware(adminOrTeacher(http.HandlerFunc(materialsHandler.Delete))))
	mux.Handle("POST /api/v1/materials/{id}/files", authMiddleware(adminOrTeacher(http.HandlerFunc(materialsHandler.UploadFile))))
	mux.Handle("POST /api/v1/materials/{id}/links", authMiddleware(adminOrTeacher(http.HandlerFunc(materialsHandler.AddLink))))
	mux.Handle("GET /api/v1/materials/files/{fileID}/download", authMiddleware(http.HandlerFunc(materialsHandler.DownloadFile)))
	mux.Handle("DELETE /api/v1/materials/files/{fileID}", authMiddleware(adminOrTeacher(http.HandlerFunc(materialsHandler.DeleteFile))))

	// --- Notifications (Phase 8) ---
	// Список своих уведомлений и отметка прочтения — все авторизованные;
	// создание системного уведомления — Admin (раздел 35 API Plan).
	mux.Handle("GET /api/v1/notifications", authMiddleware(http.HandlerFunc(notificationsHandler.List)))
	mux.Handle("PATCH /api/v1/notifications/{id}/read", authMiddleware(http.HandlerFunc(notificationsHandler.MarkRead)))
	mux.Handle("POST /api/v1/notifications", authMiddleware(adminOnly(http.HandlerFunc(notificationsHandler.Create))))

	// --- Students (Phase 9 — Curator Module) ---
	// Доступ: Curator (своя группа — проверка groups.curator_id в сервисе)
	// и Admin. Преподаватели/студенты права управления не имеют.
	mux.Handle("GET /api/v1/students", authMiddleware(adminOrCurator(http.HandlerFunc(studentsHandler.List))))
	mux.Handle("GET /api/v1/students/{id}", authMiddleware(adminOrCurator(http.HandlerFunc(studentsHandler.Get))))
	mux.Handle("POST /api/v1/students", authMiddleware(adminOrCurator(http.HandlerFunc(studentsHandler.Create))))
	mux.Handle("PATCH /api/v1/students/{id}", authMiddleware(adminOrCurator(http.HandlerFunc(studentsHandler.Update))))
	mux.Handle("DELETE /api/v1/students/{id}", authMiddleware(adminOrCurator(http.HandlerFunc(studentsHandler.Delete))))

	// --- Search (Phase 11) ---
	// Глобальный поиск: все авторизованные; набор типов зависит от роли
	// (студент — только группы/предметы/кабинеты, без людей).
	mux.Handle("GET /api/v1/search", authMiddleware(http.HandlerFunc(searchHandler.Search)))

	// --- Locales (Phase 12) ---
	// Словарь переводов — по любому языку (fallback на русский);
	// смена языка — самим пользователем (раздел 26 спецификации).
	mux.Handle("GET /api/v1/locales/{lang}", authMiddleware(http.HandlerFunc(localesHandler.Get)))
	mux.Handle("GET /api/v1/users/me/language", authMiddleware(http.HandlerFunc(localesHandler.MyLanguage)))
	mux.Handle("PATCH /api/v1/users/me/language", authMiddleware(http.HandlerFunc(localesHandler.ChangeMyLanguage)))

	var h http.Handler = mux
	h = middleware.Logging(h)
	// Phase 14 — Security Hardening: защитные заголовки; HSTS только в
	// production (HTTPS). В dev HTTP без TLS, HSTS не добавляется.
	h = middleware.SecurityHeaders(cfg.AppEnv == "production")(h)
	// CORS для development: Vite (:5173) ходит на API (:8080) с credentials.
	// В production фронтенд отдаётся nginx'ом с того же origin — CORS не нужен.
	if cfg.AppEnv == "development" {
		h = middleware.CORS("http://localhost:5173", "http://127.0.0.1:5173")(h)
	}

	addr := ":" + cfg.Port
	log.Printf("QADAM backend starting (env=%s) on %s", cfg.AppEnv, addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
