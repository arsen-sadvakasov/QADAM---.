package models

import "time"

// Course — курс обучения в рамках специальности, например "1 курс"
// (таблица courses).
type Course struct {
	ID          string
	SpecialtyID string
	YearNumber  int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
