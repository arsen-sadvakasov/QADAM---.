package models

// Role — роль пользователя в системе RBAC (таблица roles, раздел 34.1
// спецификации). RoleKey (student/teacher/curator/admin) уже определён в
// user.go — здесь описана полная строка таблицы roles, используемая при
// создании/редактировании пользователей в Admin Panel (Phase 5).
type Role struct {
	ID   string
	Key  RoleKey
	Name string
}
