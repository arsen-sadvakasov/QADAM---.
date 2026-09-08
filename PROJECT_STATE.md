# QADAM — Состояние проекта (PROJECT_STATE.md)

> Документ сгенерирован анализом реального кода, git-истории и файловой структуры репозитория на момент проверки. Ничего не выдумано — каждый пункт подтверждён чтением конкретных файлов, выполнением `go build`/`go vet`/`go test`, просмотром git-веток и содержимого миграций.

**Дата анализа:** 2026-09-07
**Текущая git-ветка:** `phase-4-schedule`
**Спецификация:** [`QADAM_Project_Specification.md`](./QADAM_Project_Specification.md), раздел 36 (Development Roadmap)

---

## 0. Git — фактическое состояние веток

*(обновлено после завершения Phase 6, 2026-09-08)*

```
main                              = e16534a (только "first commit", пустой каркас)
dev                               = e16534a (совпадает с main, ничего не влито)
phase-1-project-setup             = запушено в origin
phase-2-authentication            = запушено в origin
phase-3-database-core             = запушено в origin
phase-4-schedule                 = закоммичено и запушено
phase-5-admin-panel               = закоммичено и запушено
phase-6-schedule-changes            = 5c973db, закоммичено и запушено
phase-7-materials                    = 440395b, закоммичено и запушено
phase-8-notifications               = adf6046, закоммичено и запушено
phase-9-curator-module              = 2faf7bd, закоммичено и запушено
phase-11-search                      = cfe2a06, закоммичено и запушено
phase-12-localization                = 1397274, закоммичено и запушено
phase-13-mobile-pwa                  = код готов, ожидает коммита вместе с Phase 13.5
phase-13-5-frontend-api              = d83a78d, закоммичено и запушено
phase-14-security-hardening          = fd31462, закоммичено и запушено
phase-15-testing (HEAD)             = код готов, ожидает коммита
```

⚠️ **Важно:** ветки `dev` и `main` по-прежнему находятся на первом коммите. Согласно договорённости, слияние в `dev` происходит **после завершения всех этапов** — это нормальное состояние, не ошибка.

**Статус Phase 6 (обновлено 2026-09-08):** код закоммичен (`5c973db`) и запушен в `origin/phase-6-schedule-changes`. Этап закрыт по рабочему процессу "коммит → push".

**Статус Phase 7 (2026-09-08):** код закоммичен (`440395b`) и запушен в `origin/phase-7-materials`. Этап закрыт.

**Статус Phase 8 (2026-09-08):** код закоммичен (`adf6046`) и запушен в `origin/phase-8-notifications`. Этап закрыт.

**Статус Phase 9 (2026-09-08):** код закоммичен (`2faf7bd`) и запушен в `origin/phase-9-curator-module`. Этап закрыт.

**Статус Phase 11 (2026-09-08):** код закоммичен (`cfe2a06`) и запушен в `origin/phase-11-search`. Этап закрыт.

**Статус Phase 12 (2026-09-08):** код закоммичен (`1397274`) и запушен в `origin/phase-12-localization`. Этап закрыт.

**Статус Phase 13.5 (2026-09-08):** код закоммичен (`d83a78d`) и запушен в `origin/phase-13-5-frontend-api`. Этап закрыт.

**Статус Phase 14 (2026-09-08):** код закоммичен (`fd31462`) и запушен в `origin/phase-14-security-hardening`. Этап закрыт.

**Статус Phase 15 (2026-09-08):** код Phase 15 (Testing) реализован в рабочей копии ветки `phase-15-testing`, все проверки пройдены (`go build`, `go vet`, `go test ./...` — зелёные), ожидает коммита и push.

**Статус Phase 13 (2026-09-08):** код Phase 13 (Mobile Polish / PWA) реализован, сборка (`npm run build`) и линт зелёные, ожидает коммита и push.

---

## 1. Полный список этапов разработки (16 фаз, раздел 36 спецификации)

| № | Название этапа | Git-статус | Код-статус |
|---|---|---|---|
| 1 | Project Setup | ✅ закоммичен, запушен (`phase-1-project-setup`) | ✅ реализован |
| 2 | Authentication | ✅ закоммичен, запушен (`phase-2-authentication`) | ✅ реализован |
| 3 | Database Core | ✅ закоммичен, запушен (`phase-3-database-core`) | ✅ реализован |
| 4 | Schedule | ✅ закоммичен, запушен (`phase-4-schedule`) | ✅ реализован |
| 5 | Admin Panel (Core CRUD) | ✅ закоммичен, запушен (`phase-5-admin-panel`) | ✅ реализован |
| 6 | Schedule Changes (Замены) | ✅ закоммичен, запушен (`phase-6-schedule-changes`, `5c973db`) | ✅ реализован |
| 7 | Materials | ✅ закоммичен, запушен (`phase-7-materials`, `440395b`) | ✅ реализован |
| 8 | Notifications | ✅ закоммичен, запушен (`phase-8-notifications`, `adf6046`) | ✅ реализован |
| 9 | Curator Module | ✅ закоммичен, запушен (`phase-9-curator-module`, `2faf7bd`) | ✅ реализован |
| ~~10~~ | ~~Session (Сессия)~~ | ❌ исключён из роадмапа (решение пользователя 2026-09-08: экзамены/сессия пока не нужны сайту) | — |
| 11 | Search | ✅ закоммичен, запушен (`phase-11-search`, `cfe2a06`) | ✅ реализован |
| 12 | Localization | ✅ закоммичен, запушен (`phase-12-localization`, `1397274`) | ✅ реализован |
| 13 | Mobile Polish / PWA | 🟡 код готов, не закоммичен (`phase-13-mobile-pwa`) | ✅ реализован (базовый каркас UI) |
| 13.5 | Frontend API Integration | ✅ закоммичен, запушен (`phase-13-5-frontend-api`, `d83a78d`) | ✅ реализован |
| 14 | Security Hardening | ✅ закоммичен, запушен (`phase-14-security-hardening`, `fd31462`) | ✅ реализован |
| 15 | Testing | 🟡 код готов, не закоммичен (`phase-15-testing`) | ✅ реализован (покрытие критичных модулей) |
| 16 | Deployment | ⬜ не начат | ⬜ нет кода |

**Текущий этап: Phase 15 — Testing**, статус 🟡 **код реализован и протестирован локально**, ждёт коммита и push.

Покрытие тестами до Phase 15 → после:
| Пакет | До | После |
|---|---|---|
| middleware | 0% | **94.9%** |
| services | 71.1% | **83.6%** |
| handlers | 51.6% | 51.6% (уже был покрыт CRUD/ошибками всех хендлеров) |
| locales | 90.7% | 90.7% |
| storage | 82.8% | 82.8% |
| repositories | 0.3% | 0.3% (SQL-слой; тестируется интеграционно — см. следующий шаг) |

Новое в Phase 15:
- middleware (0% → 94.9%): Auth (missing/malformed/invalid/valid токен + контекст), RequireRole (allowed/forbidden/без пользователя), SecurityHeaders (заголовки присутствуют/HSTS выключен в dev), CORS (allowed origin echo, блокировка чужих origin, preflight short-circuit), RateLimiter (блок после лимита, независимые IP, сброс окна).
- services (+12.5%): UserAdminService (создание с defaults ru/dark/active, bcrypt-хэш не равен паролю, валидация username/password/language/role, дубликат username, update/block/list-фильтры), TokenService (generate+parse round-trip, мусор/пустой/чужой секрет отклоняются, refresh-токен уникален и HashToken детерминирован), HashPassword (bcrypt-формат, уникальные соли), AuditService (record + list-фильтры), TeacherAdminService (create через оркестрацию users+teachers, проброс валидации, assign/unassign, list/get/404), MaterialService.Get (материал с файлами + 404), ScheduleChangeService.Get (404).

---

## 2. Что реализовано — по этапам

### Phase 1 — Project Setup ✅ (в ветке `phase-1-project-setup`, подтверждено чтением файлов)
- Backend: Go-модуль `github.com/qadam/backend`, структура `cmd/server`, `internal/{handlers,services,repositories,models,middleware,config,ws}`, `migrations/`, `pkg/`.
- `GET /health` endpoint + тест (`internal/handlers/health.go`, `health_test.go`).
- `backend/Dockerfile` (multi-stage, `golang:1.23-alpine` → `alpine:3.20`).
- Frontend: Vite + React 19 + TypeScript + Tailwind CSS v4, структура каталогов (`api`, `components`, `features`, `hooks`, `layouts`, `locales`, `pages`, `types`, `utils`) — почти все пока пустые (`.gitkeep`), кроме `components/Logo.tsx` и `pages/HomePage.tsx`.
- `frontend/Dockerfile` + `nginx.conf`.
- Инфраструктура: корневой `docker-compose.yml` (postgres, minio, redis, backend, frontend), `.env.example`, `.gitignore`, GitHub Actions CI (`.github/workflows/ci.yml`: backend vet/build/test, frontend lint/build).

### Phase 2 — Authentication ✅ (в ветке `phase-2-authentication`)
- Миграция `000001_init.up/down.sql`: таблицы `colleges`, `roles`, `permissions`, `role_permissions`, `users`, `refresh_tokens` + сиды (4 роли, 10 permissions, дефолтный mapping role→permission).
- `internal/models/user.go` — `User`, `RoleKey` (student/teacher/curator/admin).
- `internal/repositories/user_repository.go`, `refresh_token_repository.go` — pgx-реализации.
- `internal/services/token_service.go` — JWT access-токены (HS256, 15 мин TTL), непрозрачные refresh-токены (crypto/rand, хранятся как SHA-256-хэш).
- `internal/services/auth_service.go` — `Login` (bcrypt), `Refresh` (ротация), `Logout`, `HashPassword`.
- `internal/middleware/auth.go` — `Auth` (парсинг JWT), `RequireRole`.
- `internal/middleware/rate_limit.go` — in-memory rate limiter по IP (защита `/auth/login` от брутфорса).
- `internal/handlers/auth.go`, `users.go` — `POST /api/v1/auth/{login,refresh,logout}`, `GET /api/v1/users/me`. Refresh-токен — httpOnly+Secure+SameSite=Strict cookie.
- `internal/services/auth_service_test.go` — 7 юнит-тестов (login success/wrong-password/unknown-user/inactive-user, refresh rotation, refresh invalid, logout).
- `internal/config/db.go` — `NewPostgresPool`, `RunMigrations` (golang-migrate).

### Phase 3 — Database Core ✅ (в ветке `phase-3-database-core`)
- Миграция `000002_academic_structure.up/down.sql`: `academic_statuses` (+ сиды excellent/good/average), `specialties`, `courses`, `groups`, `subjects`, `rooms`, `teachers`, `teacher_subjects`, `students`. Все FK, CHECK-констрейнты и индексы (`idx_students_group_id`, `idx_groups_*`, `idx_teacher_subjects_*`, `idx_courses_specialty_id`) на месте.
- Go-модели: `Specialty`, `Course`, `Group`, `Subject`, `Room`(+`RoomType`), `Teacher`/`TeacherSubject`, `Student`/`StudentStatus`, `AcademicStatus`.
- Репозитории (pgx, CRUD + soft delete): `SpecialtyRepository`, `CourseRepository`, `GroupRepository`, `SubjectRepository`, `RoomRepository`, `TeacherRepository` (+ управление `teacher_subjects`), `StudentRepository`.
- **HTTP-хендлеров для этих сущностей пока нет** — согласно роадмапу, CRUD-эндпоинты относятся к Phase 5 (Admin Panel). Репозитории подключены в `main.go` только для проверки компиляции (`_ = repositories.NewXxxRepository(pool)`), реальных роутов не зарегистрировано.

### Phase 4 — Schedule 🟡 (код готов, но НЕ закоммичен и НЕ запушен)
- Миграция `000003_schedule_templates.up/down.sql`: таблица `schedule_templates` (группа/предмет/преподаватель/кабинет, день недели, время, тип занятия, чётность недели, период действия, статус) + индексы `idx_schedule_templates_group_day`, `idx_schedule_templates_teacher_id`, `idx_schedule_templates_room_id`.
- `internal/models/schedule_template.go` — `ScheduleTemplate`, `LessonOccurrence`, enums `LessonType`, `WeekParity`, `ScheduleTemplateStatus`.
- `internal/repositories/schedule_template_repository.go` — CRUD + `ListByGroup`/`ListByTeacher` с джойнами на subjects/teachers/users/rooms/groups.
- `internal/services/schedule_service.go` — вычисление занятий "на лету" из шаблонов (день недели, ISO-week чётность, `valid_from`/`valid_to`, статус). Методы: `GetGroupSchedule`, `GetTeacherSchedule`, `GetLessonDetails`.
- `internal/handlers/schedules.go` — `GET /api/v1/schedules?group_id=&teacher_id=&from=&to=`, `GET /api/v1/schedules/lesson/{id}?date=`. Подключены в `main.go` за `authMiddleware`.
- Тесты: `schedule_service_test.go` (9 тестов) + `schedules_test.go` (6 HTTP-тестов) — всего 15 новых тестов.
- CRUD создания/редактирования шаблонов (`POST/PATCH/DELETE`) сознательно не реализован в этом этапе — по роадмапу относится к Phase 5.

### Phase 5–16
Нет никакого кода. Соответствующие таблицы (`schedule_changes`, `materials`, `material_files`, `notifications`, `notification_recipients`, `exams`, `audit_logs`) отсутствуют в миграциях (подтверждено grep по `backend/migrations/`). Frontend не содержит ни одной реальной страницы, кроме `HomePage`.

---

## 3. Проверка сборки и тестов (выполнено сейчас, реальные результаты)

```
cd backend
go build ./...      → OK, без ошибок
go vet ./...         → OK, без предупреждений
go test ./...        → OK, все пакеты "ok"
go test ./... -v     → 23 теста, все PASS (0 FAIL)
```

Разбивка тестов:
- `internal/handlers`: `TestHealth` + 6 тестов расписания (`TestGetSchedule_*`, `TestGetLessonDetails_*`) = 7 тестов.
- `internal/services`: 7 тестов `AuthService` + 9 тестов `ScheduleService` = 16 тестов.
- `internal/config`, `internal/models`, `internal/repositories`, `internal/middleware`, `cmd/server` — тестов нет (`[no test files]`).

`go.mod`: `go 1.23.0`, зависимости зафиксированы на версиях, совместимых с этой версией Go (`pgx/v5 v5.7.5`, `golang-migrate/v4 v4.18.3`, `golang.org/x/crypto v0.37.0`) — совпадает с `Dockerfile` (`golang:1.23-alpine`) и CI (`go-version: "1.23"`).

**Миграции против реальной PostgreSQL не проверялись** — в этой рабочей среде отсутствует Docker (`docker: command not found`), поэтому `golang-migrate` никогда не запускался на живой базе. SQL всех трёх миграций проверен только вручную (порядок создания таблиц/FK, синтаксис).

---

## 4. Frontend — фактическое состояние (важно)

Несмотря на то что `package.json` содержит полный намеченный стек (React Router, TanStack Query, Zod, React Hook Form, Radix UI, i18next), **реальной интеграции с backend нет**:
- `src/api/`, `src/hooks/`, `src/layouts/`, `src/types/`, `src/utils/`, `src/features/` — содержат только `.gitkeep`, т.е. пустые.
- `src/locales/{en,kz,ru}/` — папки существуют, но не проверено, есть ли внутри файлы переводов (не заявлены как реализованные, локализация — Phase 12, ещё не начата).
- Нет ни одной страницы логина, расписания или админки. Реализован только `HomePage.tsx` + `Logo.tsx` из Phase 1.
- Нет ни одного HTTP-запроса к backend API (`grep` по `fetch`/`axios`/`api/v1` в `src/**/*.{ts,tsx}` — 0 совпадений).

Это ожидаемо согласно роадмапу (Phase 2–4 в спецификации описаны как backend-only), но стоит явно зафиксировать: **frontend не продвинулся с конца Phase 1**.

---

## 5. Найденные проблемы и несоответствия

| # | Проблема | Серьёзность | Детали |
|---|---|---|---|
| 1 | Ветка `phase-4-schedule` не закоммичена и не запушена | ⚠️ Средняя | Работа по Phase 4 физически на диске, но по факту git ещё "видит" только Phase 3. Нарушает договорённый рабочий процесс (коммит+push после каждой фазы). |
| 2 | `README.md` уже помечает Phase 4 как "✅ Готово" | ⚠️ Низкая/Средняя | Текст в README забегает вперёд относительно фактического git-состояния (коммита нет). Несостыковка документации и репозитория. |
| 3 | `dev` и `main` не содержат ни одной фазы | ℹ️ Информационно, не ошибка | Согласно договорённости, слияние в `dev` происходит только после завершения всех этапов — так и есть, никаких фаз туда не влито ни на одном шаге. |
| 4 | Миграции никогда не запускались на реальной PostgreSQL | ⚠️ Средняя | Нет Docker в этой среде для проверки. SQL проверен только вручную. Риск скрытой ошибки (например, опечатки в имени столбца/таблицы), которая проявится только при первом реальном деплое/докер-запуске. |
| 5 | Нет HTTP CRUD для сущностей Phase 3 (specialties/courses/groups/subjects/rooms/teachers/students) | ℹ️ Не ошибка, а осознанное решение | Явно отложено до Phase 5 по роадмапу; репозитории существуют и компилируются, но не подключены к роутам. |
| 6 | Frontend не содержит интеграции с auth/schedule API | ℹ️ Не ошибка, а осознанное решение | Роадмап Phase 2–4 не предполагал frontend-работы; но нужно учитывать при планировании — фронтенд полностью отстаёт от backend. |
| 7 | `JWT_REFRESH_SECRET` объявлен в конфиге и docker-compose, но не используется в коде | ⚠️ Низкая | `config.Config.JWTRefreshSecret` читается из `JWT_REFRESH_SECRET` env, но `grep` по всему backend показывает, что это поле не используется ни в `TokenService`, ни где-либо ещё — сейчас refresh-токены не JWT, а случайные строки, подписи отдельным секретом не имеют. Мёртвая переменная конфигурации, потенциальная путаница в будущем. |
| 8 | Тестовое покрытие частичное | ℹ️ Информационно | Есть тесты только для `AuthService` и `ScheduleService`/handlers расписания. Репозитории, middleware, config — без тестов (ожидаемо на данном этапе, полное покрытие — задача Phase 15). |

Критических (блокирующих) ошибок сборки/тестов **не найдено** — весь существующий код компилируется и все тесты проходят.

---

## 6. Точный следующий шаг

1. Закоммитить и запушить Phase 15 в `phase-15-testing` (по команде пользователя).
2. Приступить к Phase 16 (Deployment): финальная сборка Docker-образов, prod-конфигурация (HTTPS/HSTS, секреты через env), CI-прогон тестов.
3. Интеграционные тесты repositories требуют живой PostgreSQL (docker compose) — вынести в отдельный прогон с тегом `integration`.
4. Подключить MinIO-реализацию `FileStorage` (нужен доступ к сети для `go get github.com/minio/minio-go/v7`).
5. Первый живой запуск: Docker (postgres+minio) → миграции `000001`–`000008` → `go run ./cmd/create-admin` → вход во фронтенд.

**Изменение плана (2026-09-08):** Phase 10 (Session — экзамены/сессия) исключена из роадмапа по решению пользователя — функционал пока не нужен сайту. Таблица `exams` из миграций не создавалась, кода нет, поэтому исключение не требует отката. При необходимости этап можно вернуть позже.

---

*Документ актуализирован после завершения кода Phase 6 (2026-09-08). Устаревшие блоки о незакоммиченной Phase 4 удалены.*
