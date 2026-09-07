package models

import "time"

// NotificationType — тип уведомления (раздел 19 спецификации).
type NotificationType string

const (
	NotificationScheduleChange NotificationType = "schedule_change"
	NotificationReplacement    NotificationType = "replacement"
	NotificationCancellation   NotificationType = "cancellation"
	NotificationNewMaterial    NotificationType = "new_material"
	NotificationSystem         NotificationType = "system"
	NotificationSession        NotificationType = "session"
)

// NotificationPriority — приоритет уведомления (раздел 19).
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "low"
	NotificationPriorityNormal   NotificationPriority = "normal"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// IsValid проверяет корректность типа уведомления.
func (t NotificationType) IsValid() bool {
	switch t {
	case NotificationScheduleChange, NotificationReplacement, NotificationCancellation,
		NotificationNewMaterial, NotificationSystem, NotificationSession:
		return true
	}
	return false
}

// IsValid проверяет корректность приоритета уведомления.
func (p NotificationPriority) IsValid() bool {
	switch p {
	case NotificationPriorityLow, NotificationPriorityNormal,
		NotificationPriorityHigh, NotificationPriorityCritical:
		return true
	}
	return false
}

// Notification — уведомление (таблица notifications), разворачиваемое на
// конкретных получателей через notification_recipients (раздел 19).
type Notification struct {
	ID                string
	Type              NotificationType
	Title             string
	Body              string
	Priority          NotificationPriority
	RelatedEntityType *string
	RelatedEntityID   *string
	CreatedBy         *string
	CreatedAt         time.Time
}

// NotificationRecipient — персональная запись уведомления для пользователя
// (таблица notification_recipients): индивидуальный статус прочтения.
type NotificationRecipient struct {
	ID             string
	NotificationID string
	UserID         string
	IsRead         bool
	ReadAt         *time.Time
	CreatedAt      time.Time
}

// UserNotification — уведомление конкретного пользователя: содержимое
// уведомления + его персональный статус прочтения (для API GET /notifications).
type UserNotification struct {
	Notification
	RecipientID string
	IsRead      bool
	ReadAt      *time.Time
}
