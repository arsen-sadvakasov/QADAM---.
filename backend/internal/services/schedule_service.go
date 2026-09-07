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
// на лету по шаблону + применять поверх точечные schedule_changes".
type ScheduleService struct {
	templates repositories.ScheduleTemplateRepository
	changes   repositories.ScheduleChangeRepository
}

// NewScheduleService создаёт ScheduleService с внедрёнными репозиториями.
// changes может быть nil (используется в тестах Phase 4, замены тогда не
// применяются), но в продакшене всегда передаётся.
func NewScheduleService(templates repositories.ScheduleTemplateRepository, changes repositories.ScheduleChangeRepository) *ScheduleService {
	return &ScheduleService{templates: templates, changes: changes}
}

// GetGroupSchedule возвращает занятия группы на диапазон [from, to]
// включительно, с применёнными поверх шаблона заменами.
func (s *ScheduleService) GetGroupSchedule(ctx context.Context, groupID string, from, to time.Time) ([]models.LessonOccurrence, error) {
	templates, err := s.templates.ListByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	changes, err := s.loadChanges(ctx, from, to)
	if err != nil {
		return nil, err
	}
	return expandTemplates(templates, from, to, changes)
}

// GetTeacherSchedule возвращает занятия преподавателя на диапазон [from, to]
// включительно, с применёнными поверх шаблона заменами.
func (s *ScheduleService) GetTeacherSchedule(ctx context.Context, teacherID string, from, to time.Time) ([]models.LessonOccurrence, error) {
	templates, err := s.templates.ListByTeacher(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	changes, err := s.loadChanges(ctx, from, to)
	if err != nil {
		return nil, err
	}
	return expandTemplates(templates, from, to, changes)
}

// loadChanges загружает замены в диапазоне дат (если репозиторий замен
// подключён) и индексирует их по "шаблон+дата".
func (s *ScheduleService) loadChanges(ctx context.Context, from, to time.Time) (map[changeKey]*models.ScheduleChange, error) {
	if s.changes == nil {
		return nil, nil
	}
	listed, err := s.changes.ListByDateRange(ctx, from, to)
	if err != nil {
		return nil, err
	}
	index := make(map[changeKey]*models.ScheduleChange, len(listed))
	for _, c := range listed {
		index[changeKey{templateID: c.ScheduleTemplateID, date: normalizeDate(c.ChangeDate)}] = &c.ScheduleChange
	}
	return index, nil
}

// changeKey — ключ индекса замен: шаблон + конкретная дата.
type changeKey struct {
	templateID string
	date       time.Time
}

// GetLessonDetails возвращает детальную карточку занятия (FR-3 спецификации):
// шаблон + конкретная дата, на которую запрошены детали, с применённой заменой.
func (s *ScheduleService) GetLessonDetails(ctx context.Context, templateID string, date time.Time) (*models.LessonOccurrence, error) {
	template, err := s.templates.FindByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if !templateAppliesOn(template, date) {
		return nil, repositories.ErrNotFound
	}

	var change *models.ScheduleChange
	if s.changes != nil {
		change, err = s.changes.FindByTemplateAndDate(ctx, templateID, normalizeDate(date))
		if err != nil && !errors.Is(err, repositories.ErrNotFound) {
			return nil, err
		}
	}
	occurrence := occurrenceFromTemplate(template, date)
	applyChangeToOccurrence(&occurrence, change)
	return &occurrence, nil
}

// expandTemplates разворачивает набор шаблонов в список конкретных занятий
// на каждый день диапазона [from, to], которому соответствует хотя бы один
// шаблон (день недели + чётность недели + период действия), применяя поверх
// замены: отменённые/перенесённые занятия скрываются, заменённые поля
// (преподаватель/кабинет/время) подменяются.
func expandTemplates(templates []*models.ScheduleTemplate, from, to time.Time, changes map[changeKey]*models.ScheduleChange) ([]models.LessonOccurrence, error) {
	from = normalizeDate(from)
	to = normalizeDate(to)
	if to.Before(from) {
		return nil, ErrInvalidDateRange
	}

	var result []models.LessonOccurrence
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		for _, t := range templates {
			if !templateAppliesOn(t, d) {
				continue
			}
			occurrence := occurrenceFromTemplate(t, d)
			change := changes[changeKey{templateID: t.ID, date: d}]
			// Отменённое или перенесённое на другой день занятие не показывается
			// в этот день (перенесённое появится на new_date за счёт замены,
			// добавляемой в список ниже).
			if change != nil && (change.ChangeType == models.ScheduleChangeCancel || change.ChangeType == models.ScheduleChangeMove) {
				continue
			}
			applyChangeToOccurrence(&occurrence, change)
			result = append(result, occurrence)
		}
	}

	// Занятия, перенесённые НА даты внутри диапазона (change_type=move,
	// new_date в [from, to]): показываются на новой дате с исходным
	// предметом/группой и времене́м исходного шаблона.
	for key, change := range changes {
		if change.ChangeType != models.ScheduleChangeMove || change.NewDate == nil {
			continue
		}
		newDate := normalizeDate(*change.NewDate)
		if newDate.Before(from) || newDate.After(to) {
			continue
		}
		template, ok := templateByID(templates, key.templateID)
		if !ok {
			continue
		}
		occurrence := occurrenceFromTemplate(template, newDate)
		result = append(result, occurrence)
	}
	return result, nil
}

// applyChangeToOccurrence накладывает замену на конкретное занятие:
// подменяет преподавателя, кабинет или время (cancel/move обрабатываются
// вызывающей стороной — такие занятия скрываются).
func applyChangeToOccurrence(o *models.LessonOccurrence, change *models.ScheduleChange) {
	if change == nil {
		return
	}
	switch change.ChangeType {
	case models.ScheduleChangeReplaceTeacher:
		if change.NewTeacherID != nil {
			o.TeacherID = *change.NewTeacherID
		}
	case models.ScheduleChangeReplaceRoom:
		if change.NewRoomID != nil {
			o.RoomID = *change.NewRoomID
		}
	case models.ScheduleChangeRescheduleTime:
		if change.NewStartTime != nil {
			o.StartTime = *change.NewStartTime
		}
		if change.NewEndTime != nil {
			o.EndTime = *change.NewEndTime
		}
	}
}

// templateByID ищет шаблон в уже загруженном списке по ID.
func templateByID(templates []*models.ScheduleTemplate, id string) (*models.ScheduleTemplate, bool) {
	for _, t := range templates {
		if t.ID == id {
			return t, true
		}
	}
	return nil, false
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
