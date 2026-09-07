# QADAM — Состояние проекта (PROJECT_STATE.md)

> Документ сгенерирован анализом реального кода, git-истории и файловой структуры репозитория на момент проверки. Ничего не выдумано — каждый пункт подтверждён чтением конкретных файлов, выполнением `go build`/`go vet`/`go test`, просмотром git-веток и содержимого миграций.

**Дата анализа:** 2026-09-07
**Текущая git-ветка:** `phase-4-schedule`
**Спецификация:** [`QADAM_Project_Specification.md`](./QADAM_Project_Specification.md), раздел 36 (Development Roadmap)

---

## 0. Git — фактическое состояние веток

```
main                      = e16534a (только "first commit", пустой каркас)
dev                       = e16534a (совпадает с main, ничего не влито)
phase-1-project-setup     = e4dfd25 (запушено в origin)
phase-2-authentication    = 4c677f5 (запушено в origin)
phase-3-database-core     = caf606d (запушено в origin)
phase-4-schedule (HEAD)   = caf606d + незакоммиченные изменения (см. ниже)
```

⚠️ **Важно:** ветки `dev` и `main` до сих пор находятся на первом коммите. Ни один из готовых этапов (1–3) в них не влит. Слияние в `dev`, судя по договорённости с пользователем, должно происходить **после завершения всех этапов**, поэтому это не ошибка, а зафиксированное текущее состояние, которое нужно иметь в виду.

⚠️ **Ветка `phase-4-schedule` содержит незакоммиченные изменения** (рабочая копия расходится с последним коммитом `caf606d`, который на самом деле относится к Phase 3). Файлы Phase 4 созданы на диске, но:
- не добавлены в git commit,
- не запушены в `origin/phase-4-schedule`.

Незакоммиченные/непушенные файлы прямо сейчас:
```
M  README.md
M  backend/cmd/server/main.go
?? backend/internal/handlers/schedules.go
?? backend/internal/handlers/schedules_test.go
?? backend/internal/models/schedule_template.go
?? backend/internal/repositories/schedule_template_repository.go
?? backend/internal/services/schedule_service.go
?? backend/internal/services/schedule_service_test.go
?? backend/migrations/000003_schedule_templates.down.sql
?? backend/migrations/000003_schedule_templates.up.sql
?? PROJECT_STATE.md (этот файл)
```

Это несоответствие также отражено в `README.md`, который **уже** помечает Phase 4 как "✅ Готово", хотя коммит и push для неё ещё не выполнены — статус в README на текущий момент забегает вперёд по сравнению с git.

---

## 1. Полный список этапов разработки (16 фаз, раздел 36 спецификации)

| № | Название этапа | Git-статус | Код-статус |
|---|---|---|---|
| 1 | Project Setup | ✅ закоммичен, запушен (`phase-1-project-setup`) | ✅ реализован |
| 2 | Authentication | ✅ закоммичен, запушен (`phase-2-authentication`) | ✅ реализован |
| 3 | Database Core | ✅ закоммичен, запушен (`phase-3-database-core`) | ✅ реализован |
| 4 | Schedule | 🟡 код написан, **не закоммичен и не запушен** | 🟡 реализован, ждёт коммита |
| 5 | Admin Panel (Core CRUD) | ⬜ не начат | ⬜ нет кода |
| 6 | Schedule Changes (Замены) | ⬜ не начат | ⬜ нет кода |
| 7 | Materials | ⬜ не начат | ⬜ нет кода |
| 8 | Notifications | ⬜ не начат | ⬜ нет кода |
| 9 | Curator Module | ⬜ не начат | ⬜ нет кода |
| 10 | Session (Сессия) | ⬜ не начат | ⬜ нет кода |
| 11 | Search | ⬜ не начат | ⬜ нет кода |
| 12 | Localization | ⬜ не начат (только пустые папки `locales/{en,kz,ru}`) | ⬜ нет кода |
| 13 | Mobile Polish / PWA | ⬜ не начат | ⬜ нет кода |
| 14 | Security Hardening | ⬜ не начат | ⬜ нет кода |
| 15 | Testing (полное покрытие) | ⬜ не начат как отдельный этап (частичное покрытие тестами уже есть внутри Phase 2 и 4, см. ниже) | 🟡 частично, только auth + schedule |
| 16 | Deployment | ⬜ не начат | ⬜ нет кода |

**Текущий этап: Phase 4 — Schedule**, статус 🟡 **в процессе** (код реализован и протестирован локально, но не закоммичен/не запушен — формально этап не закрыт по установленному рабочему процессу "коммит → push → новая ветка").

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

1. Дождаться решения пользователя, что делать с незакоммиченными изменениями Phase 4:
   - Если пользователь подтверждает — выполнить `git add -A && git commit` с сообщением по Phase 4, `git push` в `phase-4-schedule`, затем создать и запушить ветку `phase-5-admin-panel`.
   - Актуализировать `README.md`, если нужно (сейчас он уже отражает Phase 4 как готовую, что станет верным только после коммита).
2. После коммита Phase 4 — приступить к Phase 5 (Admin Panel — Core CRUD): HTTP-хендлеры и роуты для пользователей, групп, кабинетов, предметов, преподавателей, кураторов + CRUD для `schedule_templates` (создание/редактирование занятий), согласно разделу 35 (API Plan) и разделу 36 (Roadmap) спецификации.
3. Рекомендуется отдельно (не блокируя Phase 5) когда появится доступ к Docker — прогнать все три миграции (`000001`–`000003`) на реальной PostgreSQL хотя бы один раз, чтобы исключить риск из проблемы №4.
4. Рекомендуется решить проблему №7 (удалить неиспользуемое `JWTRefreshSecret` из конфига, либо когда/если в будущем refresh-токены станут подписанными JWT — задокументировать план использования).

**Согласно вашей инструкции, на этом анализ и документация остановлены — код не менялся, коммит не выполнялся, следующий этап не начат.**
