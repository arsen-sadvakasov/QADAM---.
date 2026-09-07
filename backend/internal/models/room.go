package models

import "time"

// RoomType — тип учебного кабинета.
type RoomType string

const (
	RoomTypeLecture  RoomType = "lecture"
	RoomTypeLab      RoomType = "lab"
	RoomTypeComputer RoomType = "computer"
	RoomTypeOther    RoomType = "other"
)

// Room — учебный кабинет/аудитория (таблица rooms).
type Room struct {
	ID        string
	CollegeID *string
	Number    string
	Name      *string
	Type      RoomType
	Floor     *int
	Building  *string
	Notes     *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
