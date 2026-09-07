package models

import "time"

// Teacher — профиль преподавателя (таблица teachers), 1:1 с users.
type Teacher struct {
	ID        string
	UserID    string
	CollegeID *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time

	// Поля, подгружаемые джойном с users (для удобного возврата в API) —
	// заполняются репозиторием, не хранятся в самой таблице teachers.
	FullName  string
	Email     *string
	Phone     *string
	AvatarURL *string
}

// TeacherSubject — назначение преподавателя на предмет в конкретной группе
// (таблица teacher_subjects, решает edge case "один предмет — разные
// преподаватели у разных групп" — раздел 34.1 спецификации).
type TeacherSubject struct {
	TeacherID string
	SubjectID string
	GroupID   string
}
