package models

import "time"

// ScheduleChangeType — тип точечного изменения расписания (таблица
// schedule_changes, раздел 34.1 спецификации).
type ScheduleChangeType string

const (
	ScheduleChangeReplaceTeacher ScheduleChangeType = "replace_teacher"
	ScheduleChangeReplaceRoom    ScheduleChangeType = "replace_room"
	ScheduleChangeRescheduleTime ScheduleChangeType = "reschedule_time"
	ScheduleChangeCancel         ScheduleChangeType = "cancel"
	ScheduleChangeMove           ScheduleChangeType = "move"
)

// ScheduleChange — точечная замена/отмена/перенос занятия на конкретную дату
// (раздел 34.1 спецификации). Накладывается поверх шаблона при отображении
// расписания (раздел 20: "вычислять на лету по шаблону + применять поверх
// точечные schedule_changes").
type ScheduleChange struct {
	ID                 string
	ScheduleTemplateID string
	ChangeDate         time.Time // дата, на которую распространяется изменение
	ChangeType         ScheduleChangeType

	// Новые значения (nullable, интерпретация зависит от ChangeType)
	NewTeacherID *string
	NewRoomID    *string
	NewStartTime *time.Time // только компонент времени
	NewEndTime   *time.Time
	NewDate      *time.Time // для переноса занятия на другой день

	Reason    *string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ScheduleChangeWithNames — замена с подгруженными джойнами именами
// ("было → стало" для UI замен, раздел 36 Phase 6).
type ScheduleChangeWithNames struct {
	ScheduleChange

	// Оригинальные данные занятия (из schedule_templates + связанные таблицы)
	OriginalTeacherName string
	OriginalRoomNumber  string
	OriginalStartTime   time.Time
	OriginalEndTime     time.Time
	SubjectName         string
	GroupName           string

	// Новые имена (если соответствующие ID заданы)
	NewTeacherName *string
	NewRoomNumber  *string
}
