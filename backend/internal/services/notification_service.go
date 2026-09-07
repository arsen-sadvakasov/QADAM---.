package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// ErrNotificationValidation возвращается при нарушении правил валидации
// уведомления — HTTP 400.
var ErrNotificationValidation = errors.New("invalid notification input")

// ErrNotificationForbidden возвращается, когда пользователь пытается
// работать с чужим уведомлением — HTTP 403.
var ErrNotificationForbidden = errors.New("not allowed to access this notification")

// NotificationService реализует бизнес-логику уведомлений (Phase 8):
// создание с fan-out на получателей, список своих, отметка прочтения,
// автоматические уведомления при заменах расписания и новых материалах.
type NotificationService struct {
	notifications repositories.NotificationRepository
}

// NewNotificationService создаёт NotificationService с внедрённым репозиторием.
func NewNotificationService(notifications repositories.NotificationRepository) *NotificationService {
	return &NotificationService{notifications: notifications}
}

// NotificationInput — входные данные для создания уведомления.
type NotificationInput struct {
	Type              models.NotificationType
	Title             string
	Body              string
	Priority          models.NotificationPriority
	RelatedEntityType *string
	RelatedEntityID   *string
}

// Create создаёт уведомление и разворачивает его на получателей
// (POST /api/v1/notifications — Admin, а также внутренние вызовы из
// сервисов замен/материалов).
func (s *NotificationService) Create(ctx context.Context, createdBy *string, in NotificationInput, recipientUserIDs []string) (string, error) {
	if !in.Type.IsValid() {
		return "", fmt.Errorf("%w: unknown notification type %q", ErrNotificationValidation, in.Type)
	}
	if in.Title == "" || in.Body == "" {
		return "", fmt.Errorf("%w: title and body are required", ErrNotificationValidation)
	}
	if in.Priority == "" {
		in.Priority = models.NotificationPriorityNormal
	}
	if !in.Priority.IsValid() {
		return "", fmt.Errorf("%w: unknown priority %q", ErrNotificationValidation, in.Priority)
	}
	if len(recipientUserIDs) == 0 {
		return "", fmt.Errorf("%w: at least one recipient is required", ErrNotificationValidation)
	}

	n := &models.Notification{
		Type:              in.Type,
		Title:             in.Title,
		Body:              in.Body,
		Priority:          in.Priority,
		RelatedEntityType: in.RelatedEntityType,
		RelatedEntityID:   in.RelatedEntityID,
		CreatedBy:         createdBy,
	}
	return s.notifications.Create(ctx, n, recipientUserIDs)
}

// ListForUser возвращает уведомления текущего пользователя
// (GET /api/v1/notifications?is_read=).
func (s *NotificationService) ListForUser(ctx context.Context, userID string, isRead *bool, limit int) ([]*models.UserNotification, error) {
	return s.notifications.ListForUser(ctx, userID, isRead, limit)
}

// CountUnread возвращает число непрочитанных уведомлений пользователя.
func (s *NotificationService) CountUnread(ctx context.Context, userID string) (int, error) {
	return s.notifications.CountUnread(ctx, userID)
}

// MarkRead отмечает уведомление прочитанным. Пользователь может отметить
// только своё уведомление (проверка через персональную запись получателя).
func (s *NotificationService) MarkRead(ctx context.Context, userID, notificationID string) error {
	_, err := s.notifications.FindRecipient(ctx, notificationID, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrNotificationForbidden
		}
		return err
	}
	return s.notifications.MarkRead(ctx, notificationID, userID, time.Now())
}

// NotifyScheduleChange создаёт уведомление о замене расписания для
// пользователей группы (вызывается из ScheduleChangeService.Create).
// studentUserIDs — ID пользователей группы; передаются вызывающей стороной,
// чтобы сервис уведомлений не зависел от репозиториев учебной структуры.
func (s *NotificationService) NotifyScheduleChange(ctx context.Context, changeType models.ScheduleChangeType, changeID, groupName, subjectName, changeDate string, studentUserIDs []string) error {
	var (
		notifType models.NotificationType
		title     string
	)
	switch changeType {
	case models.ScheduleChangeCancel:
		notifType = models.NotificationCancellation
		title = "Отмена занятия"
	case models.ScheduleChangeReplaceTeacher, models.ScheduleChangeReplaceRoom, models.ScheduleChangeRescheduleTime:
		notifType = models.NotificationReplacement
		title = "Замена по расписанию"
	case models.ScheduleChangeMove:
		notifType = models.NotificationScheduleChange
		title = "Перенос занятия"
	default:
		return fmt.Errorf("%w: unknown schedule change type %q", ErrNotificationValidation, changeType)
	}

	body := fmt.Sprintf("Группа %s, предмет %q, дата %s. Подробности в расписании.", groupName, subjectName, changeDate)
	entityType := "schedule_change"
	_, err := s.Create(ctx, nil, NotificationInput{
		Type:              notifType,
		Title:             title,
		Body:              body,
		Priority:          models.NotificationPriorityHigh,
		RelatedEntityType: &entityType,
		RelatedEntityID:   &changeID,
	}, studentUserIDs)
	return err
}

// NotifyNewMaterial создаёт уведомление о новом материале (вызывается из
// MaterialService.Create). recipientUserIDs — пользователи, которым читается
// предмет (студенты группы, куратор).
func (s *NotificationService) NotifyNewMaterial(ctx context.Context, materialID, subjectName, materialTitle string, recipientUserIDs []string) error {
	title := "Новый учебный материал"
	body := fmt.Sprintf("По предмету %q опубликован материал: %q.", subjectName, materialTitle)
	entityType := "material"
	_, err := s.Create(ctx, nil, NotificationInput{
		Type:              models.NotificationNewMaterial,
		Title:             title,
		Body:              body,
		Priority:          models.NotificationPriorityNormal,
		RelatedEntityType: &entityType,
		RelatedEntityID:   &materialID,
	}, recipientUserIDs)
	return err
}
