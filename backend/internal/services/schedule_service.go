package services

import (
	"context"
	"errors"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// ErrInvalidDateRange возвращается, когда конечная дата раньше начальной.
var ErrInvalidDateRange = errors.New("end date must not be before start date")

// MaxScheduleRangeDays — максимальный диапазон запроса расписания (месяц
// с запасом), чтобы защититься от чрезмерно широких запросов (NFR-1:
// производительность).
const MaxScheduleRangeDays = 62

// ScheduleService вычисляет конкретные занятия ("occurrences") на диапазон
// дат из шаблонов schedule_templates — раздел 20 спецификации: "вычислять
// на лету по шаблону + применять поверх точечные schedule_changes" (замены
// добавляются в Phase 6, здесь — только шаблонная часть).
type ScheduleService struct {
	templates repositories.ScheduleTemplateRepository
}

// NewScheduleService создаёт ScheduleService с внедрённым репозиторием.
func NewScheduleService(templates repositories.ScheduleTemplateRepository) *ScheduleService {
	return &ScheduleService{templates: templates}
}

// GetGroupSchedule возвращает занятия группы на диапазон [from, to] включительно.
func (s *ScheduleService) GetGroupSchedule(ctx context.Context, groupID string, from, to time.Time) ([]models.LessonOccurrence, error) {
	templates, err := s.templates.ListByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return expandTemplates(templates, from, to)
}

// GetTeacherSchedule возвращает занятия преподавателя на диапазон [from, to] включительно.
func (s *ScheduleService) GetTeacherSchedule(ctx context.Context, teacherID string, from, to time.Time) ([]models.LessonOccurrence, error) {
	templates, err := s.templates.ListByTeacher(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	return expandTemplates(templates, from, to)
}

// GetLessonDetails возвращает детальную карточку занятия (FR-3 спецификации):
// шаблон + конкретная дата, на которую запрошены детали.
func (s *ScheduleService) GetLessonDetails(ctx context.Context, templateID string, date time.Time) (*models.LessonOccurrence, error) {
	template, err := s.templates.FindByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if !templateAppliesOn(template, date) {
		return nil, repositories.ErrNotFound
	}
	occurrence := occurrenceFromTemplate(template, date)
	return &occurrence, nil
}

// expandTemplates разворачивает набор шаблонов в список конкретных занятий
// на каждый день диапазона [from, to], которому соответствует хотя бы один
// шаблон (день недели + чётность недели + период действия).
func expandTemplates(templates []*models.ScheduleTemplate, from, to time.Time) ([]models.LessonOccurrence, error) {
	from = normalizeDate(from)
	to = normalizeDate(to)
	if to.Before(from) {
		return nil, ErrInvalidDateRange
	}

	var result []models.LessonOccurrence
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		for _, t := range templates {
			if templateAppliesOn(t, d) {
				result = append(result, occurrenceFromTemplate(t, d))
			}
		}
	}
	return result, nil
}

// templateAppliesOn проверяет, распространяется ли шаблон на конкретную
// дату: день недели совпадает, дата в пределах [valid_from, valid_to],
// чётность недели совпадает (если задана), и шаблон активен.
func templateAppliesOn(t *models.ScheduleTemplate, date time.Time) bool {
	if t.Status != models.ScheduleTemplateStatusActive {
		return false
	}
	if isoWeekday(date) != t.DayOfWeek {
		return false
	}
	if date.Before(normalizeDate(t.ValidFrom)) {
		return false
	}
	if t.ValidTo != nil && date.After(normalizeDate(*t.ValidTo)) {
		return false
	}
	if t.WeekParity != models.WeekParityAll && weekParityOf(date) != t.WeekParity {
		return false
	}
	return true
}

// occurrenceFromTemplate собирает конкретное занятие на дату из шаблона.
func occurrenceFromTemplate(t *models.ScheduleTemplate, date time.Time) models.LessonOccurrence {
	return models.LessonOccurrence{
		TemplateID:  t.ID,
		Date:        date,
		GroupID:     t.GroupID,
		GroupName:   t.GroupName,
		SubjectID:   t.SubjectID,
		SubjectName: t.SubjectName,
		TeacherID:   t.TeacherID,
		TeacherName: t.TeacherName,
		RoomID:      t.RoomID,
		RoomNumber:  t.RoomNumber,
		StartTime:   t.StartTime,
		EndTime:     t.EndTime,
		LessonType:  t.LessonType,
	}
}

// isoWeekday возвращает день недели по ISO-8601: 1 = понедельник, 7 = воскресенье
// (Go's time.Weekday использует 0 = воскресенье, поэтому требуется пересчёт).
func isoWeekday(date time.Time) int {
	weekday := int(date.Weekday())
	if weekday == 0 {
		return 7
	}
	return weekday
}

// weekParityOf определяет чётность недели по номеру ISO-недели года
// (раздел 20 спецификации: "чётная/нечётная неделя"). Это разумное
// умолчание при отсутствии отдельного справочника учебного календаря.
func weekParityOf(date time.Time) models.WeekParity {
	_, week := date.ISOWeek()
	if week%2 == 0 {
		return models.WeekParityEven
	}
	return models.WeekParityOdd
}

// normalizeDate обрезает время до полуночи UTC, чтобы сравнения дат были
// корректны независимо от часового пояса входных данных.
func normalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
