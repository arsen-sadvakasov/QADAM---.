package models

import "time"

// AuditLog — запись журнала аудита (таблица audit_logs, раздел 29
// спецификации): кто, что, над какой сущностью, описание "было → стало".
type AuditLog struct {
	ID          string
	ActorID     string
	Action      string // create/update/delete/block/restore
	EntityType  string // user/group/schedule/schedule_change/material/...
	EntityID    *string
	Description string
	CreatedAt   time.Time

	// Джойн-поля для отображения (заполняются репозиторием).
	ActorName string
}
