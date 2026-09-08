package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// ErrScheduleConflict возвращается при попытке создать/отредактировать
// занятие, которое пересекается по времени с другим активным занятием того
// же преподавателя или того же кабинета (раздел 35 спецификации: "409 —
// конфликт, например кабинет уже занят в это время").
var ErrScheduleConflict = errors.New("schedule conflict: teacher or room is already booked at this time")

// ErrScheduleTemplateValidation возвращается при нарушении базовых
// правил валидации вводимых данных шаблона (невалидный day_of_week,
// end_time раньше start_time) — всегда должна отвечать HTTP 400, а не 500.
var ErrScheduleTemplateValidation = errors.New("invalid schedule template input")

// ScheduleTemplateInput — входные данные для создания/редактирования
// шаблона занятия администратором (POST/PATCH /api/v1/schedules).
type ScheduleTemplateInput struct {
	GroupID    string
	SubjectID  string
	TeacherID  string
	RoomID     string
	DayOfWeek  int
	StartTime  time.Time
	EndTime    time.Time
	LessonType models.LessonType
	WeekParity models.WeekParity
	ValidFrom  time.Time
	ValidTo    *time.Time
	Status     models.ScheduleTemplateStatus
}

// ScheduleAdminService реализует бизнес-логику управления расписанием для
// Admin Panel (Phase 5 спецификации): создание/редактирование/удаление
// шаблонов занятий с проверкой конфликтов преподавателя/кабинета.
// Phase 14: мутирующие операции журналируются в audit_logs (best-effort).
type ScheduleAdminService struct {
	templates repositories.ScheduleTemplateRepository
	audit     *AuditService
}

// NewScheduleAdminService создаёт ScheduleAdminService с внедрённым репозиторием.
func NewScheduleAdminService(templates repositories.ScheduleTemplateRepository) *ScheduleAdminService {
	return &ScheduleAdminService{templates: templates}
}

// SetAudit подключает журнал аудита (опционально; вызывается из main.go).
func (s *ScheduleAdminService) SetAudit(a *AuditService) {
	s.audit = a
}

// Create создаёt новый шаблон занятия, предварительно проверив отсутствие
// конфликтов по преподавателю и кабинету.
func (s *ScheduleAdminService) Create(ctx context.Context, actorID string, in ScheduleTemplateInput) (*models.ScheduleTemplate, error) {
	if err := validateScheduleTemplateInput(in); err != nil {
		return nil, err
	}

	if err := s.checkConflicts(ctx, in, ""); err != nil {
		return nil, err
	}

	template := &models.ScheduleTemplate{
		GroupID:    in.GroupID,
		SubjectID:  in.SubjectID,
		TeacherID:  in.TeacherID,
		RoomID:     in.RoomID,
		DayOfWeek:  in.DayOfWeek,
		StartTime:  in.StartTime,
		EndTime:    in.EndTime,
		LessonType: in.LessonType,
		WeekParity: in.WeekParity,
		ValidFrom:  in.ValidFrom,
		ValidTo:    in.ValidTo,
		Status:     in.Status,
	}

	id, err := s.templates.Create(ctx, template)
	if err != nil {
		return nil, err
	}
	template.ID = id

	// Audit: создание занятия (best-effort, раздел 29).
	if s.audit != nil {
		s.audit.Record(ctx, actorID, "create", "schedule", &id,
			"создано занятие (группа "+in.GroupID+", "+in.StartTime.Format("15:04")+"-"+in.EndTime.Format("15:04")+")")
	}
	return template, nil
}

// Update редактирует существующий шаблон занятия, проверяя конфликты среди
// прочих шаблонов (исключая сам редактируемый).
func (s *ScheduleAdminService) Update(ctx context.Context, actorID, id string, in ScheduleTemplateInput) (*models.ScheduleTemplate, error) {
	if err := validateScheduleTemplateInput(in); err != nil {
		return nil, err
	}

	existing, err := s.templates.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.checkConflicts(ctx, in, id); err != nil {
		return nil, err
	}

	existing.GroupID = in.GroupID
	existing.SubjectID = in.SubjectID
	existing.TeacherID = in.TeacherID
	existing.RoomID = in.RoomID
	existing.DayOfWeek = in.DayOfWeek
	existing.StartTime = in.StartTime
	existing.EndTime = in.EndTime
	existing.LessonType = in.LessonType
	existing.WeekParity = in.WeekParity
	existing.ValidFrom = in.ValidFrom
	existing.ValidTo = in.ValidTo
	existing.Status = in.Status

	if err := s.templates.Update(ctx, existing); err != nil {
		return nil, err
	}

	// Audit: редактирование занятия (best-effort, раздел 29).
	if s.audit != nil {
		s.audit.Record(ctx, actorID, "update", "schedule", &id,
			"изменено занятие (группа "+in.GroupID+", "+in.StartTime.Format("15:04")+"-"+in.EndTime.Format("15:04")+")")
	}
	return existing, nil
}

// Delete удаляет (soft delete) шаблон занятия.
func (s *ScheduleAdminService) Delete(ctx context.Context, actorID, id string) error {
	if err := s.templates.SoftDelete(ctx, id); err != nil {
		return err
	}
	// Audit: удаление занятия (best-effort, раздел 29).
	if s.audit != nil {
		s.audit.Record(ctx, actorID, "delete", "schedule", &id, "занятие удалено")
	}
	return nil
}

// checkConflicts проверяет, не пересекается ли новое/редактируемое занятие
// по времени с другими активными занятиями того же преподавателя или
// кабинета. excludeID — ID редактируемого шаблона (пропускается при
// сравнении), пустая строка при создании нового шаблона.
func (s *ScheduleAdminService) checkConflicts(ctx context.Context, in ScheduleTemplateInput, excludeID string) error {
	teacherTemplates, err := s.templates.ListByTeacher(ctx, in.TeacherID)
	if err != nil {
		return err
	}
	roomTemplates, err := s.templates.ListByRoom(ctx, in.RoomID)
	if err != nil {
		return err
	}

	candidates := make([]*models.ScheduleTemplate, 0, len(teacherTemplates)+len(roomTemplates))
	candidates = append(candidates, teacherTemplates...)
	candidates = append(candidates, roomTemplates...)

	for _, existing := range candidates {
		if existing.ID == excludeID {
			continue
		}
		if templatesOverlap(existing, in) {
			return ErrScheduleConflict
		}
	}
	return nil
}

// templatesOverlap проверяет пересечение существующего шаблона с новыми
// входными данными: одинаковый день недели, пересекающиеся периоды действия,
// пересекающаяся чётность недели и пересекающийся временной интервал.
func templatesOverlap(existing *models.ScheduleTemplate, in ScheduleTemplateInput) bool {
	if existing.Status != models.ScheduleTemplateStatusActive {
		return false
	}
	if existing.DayOfWeek != in.DayOfWeek {
		return false
	}
	if !dateRangesOverlap(existing.ValidFrom, existing.ValidTo, in.ValidFrom, in.ValidTo) {
		return false
	}
	if !weekParitiesOverlap(existing.WeekParity, in.WeekParity) {
		return false
	}
	return timeRangesOverlap(existing.StartTime, existing.EndTime, in.StartTime, in.EndTime)
}

func dateRangesOverlap(aFrom time.Time, aTo *time.Time, bFrom time.Time, bTo *time.Time) bool {
	if aTo != nil && bFrom.After(*aTo) {
		return false
	}
	if bTo != nil && aFrom.After(*bTo) {
		return false
	}
	return true
}

func weekParitiesOverlap(a, b models.WeekParity) bool {
	if a == models.WeekParityAll || b == models.WeekParityAll {
		return true
	}
	return a == b
}

func timeRangesOverlap(aStart, aEnd, bStart, bEnd time.Time) bool {
	aStartMin, aEndMin := timeOfDayMinutes(aStart), timeOfDayMinutes(aEnd)
	bStartMin, bEndMin := timeOfDayMinutes(bStart), timeOfDayMinutes(bEnd)
	return aStartMin < bEndMin && bStartMin < aEndMin
}

func timeOfDayMinutes(t time.Time) int {
	return t.Hour()*60 + t.Minute()
}

func validateScheduleTemplateInput(in ScheduleTemplateInput) error {
	if in.DayOfWeek < 1 || in.DayOfWeek > 7 {
		return fmt.Errorf("%w: day_of_week must be between 1 and 7", ErrScheduleTemplateValidation)
	}
	if !in.EndTime.After(in.StartTime) {
		return fmt.Errorf("%w: end_time must be after start_time", ErrScheduleTemplateValidation)
	}
	if in.ValidTo != nil && in.ValidTo.Before(in.ValidFrom) {
		return ErrInvalidDateRange
	}
	return nil
}
