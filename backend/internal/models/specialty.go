package models

import "time"

// Specialty — специальность обучения (таблица specialties).
type Specialty struct {
	ID        string
	CollegeID *string
	Name      string
	Code      *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
