package models

import "time"

// Subject — учебная дисциплина (таблица subjects).
type Subject struct {
	ID        string
	CollegeID *string
	Name      string
	Code      *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
