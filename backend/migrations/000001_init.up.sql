-- Phase 2 (Authentication) + Phase 3 (Database Core) foundational schema.
-- Only entities required for auth/RBAC are created here in full; other
-- entities from the ER model (section 34 of the spec) are introduced in
-- their respective phases to keep migrations aligned with delivered features.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE colleges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(100) NOT NULL UNIQUE,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    college_id UUID REFERENCES colleges(id),
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(30),
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(500),
    role_id UUID NOT NULL REFERENCES roles(id),
    language VARCHAR(10) NOT NULL DEFAULT 'ru',
    theme_preference VARCHAR(10) NOT NULL DEFAULT 'dark',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_users_username ON users (username);
CREATE UNIQUE INDEX idx_users_email ON users (email) WHERE email IS NOT NULL;

-- Refresh tokens are stored server-side (hashed) so they can be revoked
-- individually (logout, password change, admin block) — see spec section 15.
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

INSERT INTO roles (key, name) VALUES
    ('student', 'Студент'),
    ('teacher', 'Преподаватель'),
    ('curator', 'Куратор'),
    ('admin', 'Администратор');

INSERT INTO permissions (key, description) VALUES
    ('schedule.view.own', 'Просмотр собственного расписания'),
    ('schedule.view.group', 'Просмотр расписания своей группы'),
    ('schedule.manage', 'Создание/изменение/удаление занятий и замен'),
    ('students.view.group', 'Просмотр списка студентов своей группы'),
    ('students.manage', 'Редактирование данных студентов'),
    ('materials.upload', 'Загрузка учебных материалов'),
    ('materials.view', 'Просмотр материалов'),
    ('notifications.send', 'Отправка системных уведомлений'),
    ('users.manage', 'Управление пользователями и ролями'),
    ('audit.view', 'Просмотр журнала аудита');

-- Default role -> permission mapping per spec section 17.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE (r.key = 'student' AND p.key IN ('schedule.view.own', 'materials.view'))
   OR (r.key = 'teacher' AND p.key IN ('schedule.view.own', 'students.view.group', 'materials.upload', 'materials.view'))
   OR (r.key = 'curator' AND p.key IN ('schedule.view.own', 'schedule.view.group', 'students.view.group', 'students.manage', 'materials.view'))
   OR (r.key = 'admin' AND p.key IN ('schedule.view.own', 'schedule.view.group', 'schedule.manage', 'students.view.group', 'students.manage', 'materials.upload', 'materials.view', 'notifications.send', 'users.manage', 'audit.view'));
