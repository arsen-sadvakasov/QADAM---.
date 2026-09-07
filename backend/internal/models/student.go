package models

import "time"

// StudentStatus — статус студента (раздел 34.1 спецификации).
type StudentStatus string

const (
	StudentStatusActive        StudentStatus = "active"
	StudentStatusExpelled      StudentStatus = "expelled"
	StudentStatusAcademicLeave StudentStatus = "academic_leave"
	StudentStatusGraduated     StudentStatus = "graduated"
)

// Student — профиль студента (таблица students).
type Student struct {
	ID                string
	UserID            *string
	GroupID           string
	Status            StudentStatus
	AcademicStatusID  *string
	ScholarshipStatus *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time

	// Поля, подгружаемые джойном с users, когда у студента есть аккаунт —
	// заполняются репозиторием, не хранятся в самой таблице students.
	FullName  *string
	Email     *string
	Phone     *string
	AvatarURL *string
}
