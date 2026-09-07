package models

import "time"

// Group — учебная группа (таблица groups), например "ПО-23".
type Group struct {
	ID          string
	CollegeID   *string
	SpecialtyID string
	CourseID    string
	CuratorID   *string
	Name        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
