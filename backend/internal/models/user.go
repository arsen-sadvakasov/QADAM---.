// Package models содержит доменные структуры QADAM, отражающие таблицы БД
// (см. раздел 34 спецификации — Database ER Model).
package models

import "time"

// RoleKey — ключ роли пользователя в системе RBAC (раздел 4, 17 спецификации).
type RoleKey string

const (
	RoleStudent RoleKey = "student"
	RoleTeacher RoleKey = "teacher"
	RoleCurator RoleKey = "curator"
	RoleAdmin   RoleKey = "admin"
)

// User — доменная модель пользователя (таблица users).
type User struct {
	ID              string
	CollegeID       *string
	Email           *string
	Phone           *string
	Username        string
	PasswordHash    string
	FullName        string
	AvatarURL       *string
	RoleID          string
	RoleKey         RoleKey
	Language        string
	ThemePreference string
	IsActive        bool
	LastLoginAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}
