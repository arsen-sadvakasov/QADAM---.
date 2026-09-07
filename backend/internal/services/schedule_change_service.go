package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// ErrScheduleChangeValidation возвращается при нарушении правил валидации
// данных замены (неверный тип, отсутствуют обязательные поля под тип) —
// всегда должна отвечать HTTP 400, а не 500.
var ErrScheduleChangeValidation = errors.New("invalid schedule change input")

// ErrScheduleChangeConflict возвращается, когда для шаблона на выбранную
// дату уже существует замена (unique-индекс uniq_schedule_change_per_template_date).
var ErrScheduleChangeConflict = errors.New("schedule change already exists for this template and date")

// ScheduleChangeInput — входные данные для создания/редактирования замены
// (POST/PATCH /api/v1/schedule-changes, раздел 35 API Plan).
type ScheduleChangeInput struct {
	ScheduleTemplateID string
	ChangeDate         time.Time
	ChangeType         models.ScheduleChangeType
	NewTeacherID       *string
	NewRoomID          *string
	NewStartTime       *time.Time
	NewEndTime         *time.Time
	NewDate            *time.Time
	Reason             *string
}

// ScheduleChangeService реализует бизнес-логику замен расписания
// (Phase 6 спецификации): валидация по типу замены, проверка конфликта
// "одна замена на шаблон+дата", CRUD, авто-уведомления затронутым
// пользователям (Phase 8).
type ScheduleChangeService struct {
	changes   repositories.ScheduleChangeRepository
	templates repositories.ScheduleTemplateRepository
	notifier  ScheduleChangeNotifier
}

// ScheduleChangeNotifier — интерфейс отправки уведомлений о заменах.
// Отделяет сервис замен от сервиса уведомлений (упрощает тестирование;
// реализуется *NotificationService).
type ScheduleChangeNotifier interface {
	NotifyScheduleChange(ctx context.Context, changeType models.ScheduleChangeType,
		changeID, groupName, subjectName, changeDate string, studentUserIDs []string) error
}

// NewScheduleChangeService создаёт ScheduleChangeService с внедрёнными
// репозиториями. notifier может быть nil (тогда уведомления не отправляются).
func NewScheduleChangeService(changes repositories.ScheduleChangeRepository, templates repositories.ScheduleTemplateRepository, notifier ScheduleChangeNotifier) *ScheduleChangeService {
	return &ScheduleChangeService{changes: changes, templates: templates, notifier: notifier}
}

// Create создаёт замену, проверяя существование шаблона и отсутствие
// замены на ту же дату.
func (s *ScheduleChangeService) Create(ctx context.Context, createdBy string, in ScheduleChangeInput) (*models.ScheduleChange, error) {
	if err := validateScheduleChangeInput(in); err != nil {
		return nil, err
	}

	template, err := s.templates.FindByID(ctx, in.ScheduleTemplateID)
	if err != nil {
		return nil, err
	}

	existing, err := s.changes.FindByTemplateAndDate(ctx, in.ScheduleTemplateID, in.ChangeDate)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrScheduleChangeConflict
	}

	change := &models.ScheduleChange{
		ScheduleTemplateID: in.ScheduleTemplateID,
		ChangeDate:         in.ChangeDate,
		ChangeType:         in.ChangeType,
		NewTeacherID:       in.NewTeacherID,
		NewRoomID:          in.NewRoomID,
		NewStartTime:       in.NewStartTime,
		NewEndTime:         in.NewEndTime,
		NewDate:            in.NewDate,
		Reason:             in.Reason,
		CreatedBy:          createdBy,
	}

	id, err := s.changes.Create(ctx, change)
	if err != nil {
		return nil, err
	}
	change.ID = id

	s.notifyScheduleChange(ctx, change, template)
	return change, nil
}

// notifyScheduleChange отправляет уведомление студентам группы о замене.
// Ошибка уведомления не откатывает создание замены — замену логируем.
func (s *ScheduleChangeService) notifyScheduleChange(ctx context.Context, change *models.ScheduleChange, template *models.ScheduleTemplate) {
	if s.notifier == nil {
		return
	}
	studentUserIDs, err := s.templates.ListActiveStudentUserIDsByGroup(ctx, template.GroupID)
	if err != nil || len(studentUserIDs) == 0 {
		return
	}
	_ = s.notifier.NotifyScheduleChange(ctx, change.ChangeType, change.ID,
		template.GroupName, template.SubjectName,
		change.ChangeDate.Format("2006-01-02"), studentUserIDs)
}

// Get возвращает замену по ID.
func (s *ScheduleChangeService) Get(ctx context.Context, id string) (*models.ScheduleChange, error) {
	return s.changes.FindByID(ctx, id)
}

// List возвращает замены в диапазоне дат с фильтрами по группе/преподавателю.
// Ровно один из groupID/teacherID может быть задан (или оба пусты — все замены).
func (s *ScheduleChangeService) List(ctx context.Context, from, to time.Time, groupID, teacherID string) ([]*models.ScheduleChangeWithNames, error) {
	from = normalizeDate(from)
	to = normalizeDate(to)
	if to.Before(from) {
		return nil, ErrInvalidDateRange
	}

	switch {
	case groupID != "" && teacherID != "":
		return nil, fmt.Errorf("%w: provide only one of group_id or teacher_id", ErrScheduleChangeValidation)
	case groupID != "":
		return s.changes.ListByGroupAndDateRange(ctx, groupID, from, to)
	case teacherID != "":
		return s.changes.ListByTeacherAndDateRange(ctx, teacherID, from, to)
	default:
		return s.changes.ListByDateRange(ctx, from, to)
	}
}

// Update редактирует существующую замену.
func (s *ScheduleChangeService) Update(ctx context.Context, id string, in ScheduleChangeInput) (*models.ScheduleChange, error) {
	existing, err := s.changes.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := validateScheduleChangeInput(in); err != nil {
		return nil, err
	}

	// Если меняется дата замены — проверяем уникальность "шаблон+дата".
	if !normalizeDate(in.ChangeDate).Equal(normalizeDate(existing.ChangeDate)) {
		conflict, err := s.changes.FindByTemplateAndDate(ctx, existing.ScheduleTemplateID, in.ChangeDate)
		if err != nil && !errors.Is(err, repositories.ErrNotFound) {
			return nil, err
		}
		if conflict != nil && conflict.ID != existing.ID {
			return nil, ErrScheduleChangeConflict
		}
	}

	existing.ChangeType = in.ChangeType
	existing.NewTeacherID = in.NewTeacherID
	existing.NewRoomID = in.NewRoomID
	existing.NewStartTime = in.NewStartTime
	existing.NewEndTime = in.NewEndTime
	existing.NewDate = in.NewDate
	existing.Reason = in.Reason

	if err := s.changes.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// Delete удаляет замену.
func (s *ScheduleChangeService) Delete(ctx context.Context, id string) error {
	return s.changes.Delete(ctx, id)
}

// validateScheduleChangeInput проверяет соответствие полей типу замены:
// replace_teacher требует new_teacher_id, replace_room — new_room_id,
// reschedule_time — новый интервал времени (end > start), move — new_date.
// cancel дополнительных полей не требует.
func validateScheduleChangeInput(in ScheduleChangeInput) error {
	switch in.ChangeType {
	case models.ScheduleChangeReplaceTeacher:
		if in.NewTeacherID == nil || *in.NewTeacherID == "" {
			return fmt.Errorf("%w: new_teacher_id is required for replace_teacher", ErrScheduleChangeValidation)
		}
	case models.ScheduleChangeReplaceRoom:
		if in.NewRoomID == nil || *in.NewRoomID == "" {
			return fmt.Errorf("%w: new_room_id is required for replace_room", ErrScheduleChangeValidation)
		}
	case models.ScheduleChangeRescheduleTime:
		if in.NewStartTime == nil || in.NewEndTime == nil {
			return fmt.Errorf("%w: new_start_time and new_end_time are required for reschedule_time", ErrScheduleChangeValidation)
		}
		if !in.NewEndTime.After(*in.NewStartTime) {
			return fmt.Errorf("%w: new_end_time must be after new_start_time", ErrScheduleChangeValidation)
		}
	case models.ScheduleChangeMove:
		if in.NewDate == nil {
			return fmt.Errorf("%w: new_date is required for move", ErrScheduleChangeValidation)
		}
		if normalizeDate(*in.NewDate).Equal(normalizeDate(in.ChangeDate)) {
			return fmt.Errorf("%w: new_date must differ from change_date", ErrScheduleChangeValidation)
		}
	case models.ScheduleChangeCancel:
		// дополнительных полей не требуется
	default:
		return fmt.Errorf("%w: unknown change_type %q", ErrScheduleChangeValidation, in.ChangeType)
	}
	return nil
}
