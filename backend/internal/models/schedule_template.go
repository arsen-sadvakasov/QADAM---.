package models

import "time"

// LessonType — тип занятия (раздел 34.1 спецификации).
type LessonType string

const (
	LessonTypeLecture  LessonType = "lecture"
	LessonTypePractice LessonType = "practice"
	LessonTypeLab      LessonType = "lab"
	LessonTypeSeminar  LessonType = "seminar"
	LessonTypeOther    LessonType = "other"
)

// WeekParity — чётность недели, на которую распространяется занятие
// (раздел 20 спецификации: "чётная/нечётная неделя").
type WeekParity string

const (
	WeekParityAll  WeekParity = "all"
	WeekParityOdd  WeekParity = "odd"
	WeekParityEven WeekParity = "even"
)

// ScheduleTemplateStatus — статус шаблона занятия.
type ScheduleTemplateStatus string

const (
	ScheduleTemplateStatusActive    ScheduleTemplateStatus = "active"
	ScheduleTemplateStatusCancelled ScheduleTemplateStatus = "cancelled"
)

// ScheduleTemplate — регулярное занятие (таблица schedule_templates).
// Конкретные занятия на дни/недели/месяц вычисляются "на лету" из шаблона
// (см. LessonOccurrence), а не хранятся построчно на каждый день —
// решение из раздела 20 спецификации.
type ScheduleTemplate struct {
	ID         string
	GroupID    string
	SubjectID  string
	TeacherID  string
	RoomID     string
	DayOfWeek  int       // 1 (понедельник) .. 7 (воскресенье), ISO-8601
	StartTime  time.Time // используется только компонент времени (часы/минуты)
	EndTime    time.Time
	LessonType LessonType
	WeekParity WeekParity
	ValidFrom  time.Time
	ValidTo    *time.Time
	Status     ScheduleTemplateStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time

	// Поля, подгружаемые джойнами для удобного отображения карточки занятия
	// (FR-3 спецификации) без дополнительных запросов на клиенте.
	SubjectName string
	TeacherName string
	RoomNumber  string
	GroupName   string
}

// LessonOccurrence — конкретное занятие на определённую дату, вычисленное
// из ScheduleTemplate (раздел 20 спецификации: "вычислять на лету по
// шаблону"). В Phase 6 сюда будут накладываться поверх schedule_changes.
type LessonOccurrence struct {
	TemplateID  string
	Date        time.Time
	GroupID     string
	GroupName   string
	SubjectID   string
	SubjectName string
	TeacherID   string
	TeacherName string
	RoomID      string
	RoomNumber  string
	StartTime   time.Time
	EndTime     time.Time
	LessonType  LessonType
}
