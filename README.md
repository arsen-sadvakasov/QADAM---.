# QADAM

Цифровая платформа управления учебным процессом колледжа (Academic Management System).

Полная спецификация проекта — [QADAM_Project_Specification.md](./QADAM_Project_Specification.md).

## Структура репозитория

```
QADAM/
  backend/     — Go REST API (см. backend/internal для слоёв handlers/services/repositories)
  frontend/    — React + TypeScript + Tailwind CSS (Vite)
  docker-compose.yml — локальное окружение: postgres, minio, redis, backend, frontend
  .env.example — пример переменных окружения
```

## Стек

- **Backend:** Go, PostgreSQL, MinIO (S3-совместимое хранилище файлов), Redis.
- **Frontend:** React, TypeScript, Tailwind CSS, React Router, TanStack Query, Zod, React Hook Form, Radix UI, i18next.

## Локальный запуск

### Через Docker Compose (полный стек)

```sh
cp .env.example .env
docker compose up --build
```

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080 (health-check: `/health`)
- MinIO Console: http://localhost:9001
- PostgreSQL: `localhost:5432`

### Backend отдельно (без Docker)

```sh
cd backend
go run ./cmd/server
```

### Frontend отдельно (без Docker)

```sh
cd frontend
npm install
npm run dev
```

## Статус разработки

Разработка ведётся строго поэтапно (16 фаз), согласно разделу 36 спецификации.
Текущий этап: **Phase 3 — Database Core** ✅.

| Фаза | Статус |
|---|---|
| Phase 1 — Project Setup | ✅ Готово |
| Phase 2 — Authentication | ✅ Готово |
| Phase 3 — Database Core | ✅ Готово |
| Phase 4 — Schedule | ⏳ Следующий |
| Phase 5–16 | Ожидают |
