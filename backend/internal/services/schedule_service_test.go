package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// fakeScheduleTemplateRepository — тестовая заглушка ScheduleTemplateRepository.
type fakeScheduleTemplateRepository struct {
	byID           map[string]*models.ScheduleTemplate
	byGroup        map[string][]*models.ScheduleTemplate
	byTeacher      map[string][]*models.ScheduleTemplate
	byRoom         map[string][]*models.ScheduleTemplate
	studentUserIDs []string // UserID активных студентов группы (Phase 8 уведомления)
}

func newFakeScheduleTemplateRepository() *fakeScheduleTemplateRepository {
	return &fakeScheduleTemplateRepository{
		byID:      make(map[string]*models.ScheduleTemplate),
		byGroup:   make(map[string][]*models.ScheduleTemplate),
		byTeacher: make(map[string][]*models.ScheduleTemplate),
		byRoom:    make(map[string][]*models.ScheduleTemplate),
	}
}

func (f *fakeScheduleTemplateRepository) add(t *models.ScheduleTemplate) {
	f.byID[t.ID] = t
	f.byGroup[t.GroupID] = append(f.byGroup[t.GroupID], t)
	f.byTeacher[t.TeacherID] = append(f.byTeacher[t.TeacherID], t)
	f.byRoom[t.RoomID] = append(f.byRoom[t.RoomID], t)
}

func (f *fakeScheduleTemplateRepository) FindByID(_ context.Context, id string) (*models.ScheduleTemplate, error) {
	t, ok := f.byID[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return t, nil
}

func (f *fakeScheduleTemplateRepository) ListByGroup(_ context.Context, groupID string) ([]*models.ScheduleTemplate, error) {
	return f.byGroup[groupID], nil
}

func (f *fakeScheduleTemplateRepository) ListByTeacher(_ context.Context, teacherID string) ([]*models.ScheduleTemplate, error) {
	return f.byTeacher[teacherID], nil
}

func (f *fakeScheduleTemplateRepository) ListByRoom(_ context.Context, roomID string) ([]*models.ScheduleTemplate, error) {
	return f.byRoom[roomID], nil
}

func (f *fakeScheduleTemplateRepository) Create(_ context.Context, t *models.ScheduleTemplate) (string, error) {
	if t.ID == "" {
		t.ID = fmt.Sprintf("tpl-generated-%d", len(f.byID)+1)
	}
	f.add(t)
	return t.ID, nil
}

func (f *fakeScheduleTemplateRepository) Update(_ context.Context, t *models.ScheduleTemplate) error {
	f.byID[t.ID] = t
	// Перестраиваем вторичные индексы, чтобы последующие проверки конфликтов
	// (ListByTeacher/ListByRoom) видели обновлённые данные шаблона.
	f.rebuildIndexes()
	return nil
}

func (f *fakeScheduleTemplateRepository) rebuildIndexes() {
	f.byGroup = make(map[string][]*models.ScheduleTemplate)
	f.byTeacher = make(map[string][]*models.ScheduleTemplate)
	f.byRoom = make(map[string][]*models.ScheduleTemplate)
	for _, t := range f.byID {
		f.byGroup[t.GroupID] = append(f.byGroup[t.GroupID], t)
		f.byTeacher[t.TeacherID] = append(f.byTeacher[t.TeacherID], t)
		f.byRoom[t.RoomID] = append(f.byRoom[t.RoomID], t)
	}
}

func (f *fakeScheduleTemplateRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

// ListActiveStudentUserIDsByGroup — заглушка для уведомлений (Phase 8):
// возвращает UserID активных студентов группы из заглушенных шаблонов.
func (f *fakeScheduleTemplateRepository) ListActiveStudentUserIDsByGroup(_ context.Context, groupID string) ([]string, error) {
	return f.studentUserIDs, nil
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("failed to parse date %q: %v", s, err)
	}
	return d
}

func mustParseClock(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse("15:04", s)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", s, err)
	}
	return tm
}

// baseTemplate returns a Monday, all-parity, active template valid for a wide range.
func baseTemplate(t *testing.T) *models.ScheduleTemplate {
	return &models.ScheduleTemplate{
		ID:          "tpl-1",
		GroupID:     "group-1",
		SubjectID:   "subject-1",
		TeacherID:   "teacher-1",
		RoomID:      "room-1",
		DayOfWeek:   1, // Monday
		StartTime:   mustParseClock(t, "09:00"),
		EndTime:     mustParseClock(t, "10:30"),
		LessonType:  models.LessonTypeLecture,
		WeekParity:  models.WeekParityAll,
		ValidFrom:   mustParseDate(t, "2024-01-01"),
		ValidTo:     nil,
		Status:      models.ScheduleTemplateStatusActive,
		SubjectName: "Databases",
		TeacherName: "Ivanov I.I.",
		RoomNumber:  "301",
		GroupName:   "PO-23",
	}
}

func TestScheduleService_GetGroupSchedule_MatchesDayOfWeek(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	repo.add(baseTemplate(t))
	svc := NewScheduleService(repo, nil)

	// 2024-06-03 is a Monday, 2024-06-04 is a Tuesday.
	from := mustParseDate(t, "2024-06-03")
	to := mustParseDate(t, "2024-06-09") // full week, one Monday inside

	occurrences, err := svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(occurrences) != 1 {
		t.Fatalf("expected exactly 1 occurrence in the week, got %d", len(occurrences))
	}
	if !occurrences[0].Date.Equal(mustParseDate(t, "2024-06-03")) {
		t.Errorf("expected occurrence on Monday 2024-06-03, got %v", occurrences[0].Date)
	}
	if occurrences[0].SubjectName != "Databases" {
		t.Errorf("expected joined subject name, got %q", occurrences[0].SubjectName)
	}
}

func TestScheduleService_GetGroupSchedule_RespectsValidToBoundary(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	tpl := baseTemplate(t)
	validTo := mustParseDate(t, "2024-06-03")
	tpl.ValidTo = &validTo
	repo.add(tpl)
	svc := NewScheduleService(repo, nil)

	// Query a range starting the day after valid_to.
	from := mustParseDate(t, "2024-06-04")
	to := mustParseDate(t, "2024-06-10")

	occurrences, err := svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(occurrences) != 0 {
		t.Errorf("expected no occurrences past valid_to, got %d", len(occurrences))
	}
}

func TestScheduleService_GetGroupSchedule_RespectsWeekParity(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	tpl := baseTemplate(t)
	tpl.WeekParity = models.WeekParityOdd
	repo.add(tpl)
	svc := NewScheduleService(repo, nil)

	// ISO week of 2024-06-03 is week 23 (odd) -> should match.
	// ISO week of 2024-06-10 is week 24 (even) -> should not match.
	from := mustParseDate(t, "2024-06-03")
	to := mustParseDate(t, "2024-06-10")

	occurrences, err := svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(occurrences) != 1 {
		t.Fatalf("expected exactly 1 occurrence (odd week only), got %d", len(occurrences))
	}
	if !occurrences[0].Date.Equal(mustParseDate(t, "2024-06-03")) {
		t.Errorf("expected occurrence on odd-week Monday 2024-06-03, got %v", occurrences[0].Date)
	}
}

func TestScheduleService_GetGroupSchedule_SkipsCancelledTemplate(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	tpl := baseTemplate(t)
	tpl.Status = models.ScheduleTemplateStatusCancelled
	repo.add(tpl)
	svc := NewScheduleService(repo, nil)

	from := mustParseDate(t, "2024-06-03")
	to := mustParseDate(t, "2024-06-09")

	occurrences, err := svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(occurrences) != 0 {
		t.Errorf("expected no occurrences for a cancelled template, got %d", len(occurrences))
	}
}

func TestScheduleService_GetGroupSchedule_InvalidRange(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleService(repo, nil)

	from := mustParseDate(t, "2024-06-10")
	to := mustParseDate(t, "2024-06-03") // before from

	_, err := svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != ErrInvalidDateRange {
		t.Errorf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestScheduleService_GetTeacherSchedule(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	repo.add(baseTemplate(t))
	svc := NewScheduleService(repo, nil)

	from := mustParseDate(t, "2024-06-03")
	to := mustParseDate(t, "2024-06-09")

	occurrences, err := svc.GetTeacherSchedule(context.Background(), "teacher-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(occurrences) != 1 {
		t.Fatalf("expected exactly 1 occurrence, got %d", len(occurrences))
	}
}

func TestScheduleService_GetLessonDetails_Success(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	repo.add(baseTemplate(t))
	svc := NewScheduleService(repo, nil)

	date := mustParseDate(t, "2024-06-03") // Monday, matches template

	occurrence, err := svc.GetLessonDetails(context.Background(), "tpl-1", date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if occurrence.TemplateID != "tpl-1" {
		t.Errorf("expected template ID tpl-1, got %s", occurrence.TemplateID)
	}
	if occurrence.RoomNumber != "301" {
		t.Errorf("expected joined room number 301, got %q", occurrence.RoomNumber)
	}
}

func TestScheduleService_GetLessonDetails_WrongDayReturnsNotFound(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	repo.add(baseTemplate(t)) // Monday template
	svc := NewScheduleService(repo, nil)

	date := mustParseDate(t, "2024-06-04") // Tuesday, does not match

	_, err := svc.GetLessonDetails(context.Background(), "tpl-1", date)
	if err != repositories.ErrNotFound {
		t.Errorf("expected ErrNotFound for a date the template doesn't apply to, got %v", err)
	}
}

func TestScheduleService_GetLessonDetails_UnknownTemplate(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleService(repo, nil)

	_, err := svc.GetLessonDetails(context.Background(), "does-not-exist", mustParseDate(t, "2024-06-03"))
	if err != repositories.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
