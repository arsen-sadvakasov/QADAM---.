package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// fakeScheduleChangeRepository — in-memory реализация
// repositories.ScheduleChangeRepository для unit-тестов сервиса.
type fakeScheduleChangeRepository struct {
	byID       map[string]*models.ScheduleChange
	nextID     int
	lastUpdate *models.ScheduleChange
	deletedIDs []string
}

func newFakeScheduleChangeRepository() *fakeScheduleChangeRepository {
	return &fakeScheduleChangeRepository{byID: make(map[string]*models.ScheduleChange)}
}

func (f *fakeScheduleChangeRepository) FindByID(_ context.Context, id string) (*models.ScheduleChange, error) {
	if c, ok := f.byID[id]; ok {
		return c, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeScheduleChangeRepository) FindByTemplateAndDate(_ context.Context, templateID string, date time.Time) (*models.ScheduleChange, error) {
	for _, c := range f.byID {
		if c.ScheduleTemplateID == templateID && normalizeDate(c.ChangeDate).Equal(normalizeDate(date)) {
			return c, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeScheduleChangeRepository) ListByDateRange(_ context.Context, from, to time.Time) ([]*models.ScheduleChangeWithNames, error) {
	var result []*models.ScheduleChangeWithNames
	for _, c := range f.byID {
		d := normalizeDate(c.ChangeDate)
		if !d.Before(normalizeDate(from)) && !d.After(normalizeDate(to)) {
			result = append(result, &models.ScheduleChangeWithNames{ScheduleChange: *c})
		}
	}
	return result, nil
}

func (f *fakeScheduleChangeRepository) ListByGroupAndDateRange(_ context.Context, _ string, from, to time.Time) ([]*models.ScheduleChangeWithNames, error) {
	return f.ListByDateRange(context.Background(), from, to)
}

func (f *fakeScheduleChangeRepository) ListByTeacherAndDateRange(_ context.Context, _ string, from, to time.Time) ([]*models.ScheduleChangeWithNames, error) {
	return f.ListByDateRange(context.Background(), from, to)
}

func (f *fakeScheduleChangeRepository) Create(_ context.Context, c *models.ScheduleChange) (string, error) {
	f.nextID++
	c.ID = string(rune('a' + f.nextID - 1))
	clone := *c
	f.byID[c.ID] = &clone
	return c.ID, nil
}

func (f *fakeScheduleChangeRepository) Update(_ context.Context, c *models.ScheduleChange) error {
	if _, ok := f.byID[c.ID]; !ok {
		return repositories.ErrNotFound
	}
	clone := *c
	f.byID[c.ID] = &clone
	f.lastUpdate = &clone
	return nil
}

func (f *fakeScheduleChangeRepository) Delete(_ context.Context, id string) error {
	if _, ok := f.byID[id]; !ok {
		return repositories.ErrNotFound
	}
	delete(f.byID, id)
	f.deletedIDs = append(f.deletedIDs, id)
	return nil
}

func baseScheduleChangeInput(t *testing.T) ScheduleChangeInput {
	return ScheduleChangeInput{
		ScheduleTemplateID: "template-1",
		ChangeDate:         mustParseDate(t, "2024-03-11"),
		ChangeType:         models.ScheduleChangeCancel,
	}
}

func TestScheduleChangeService_Create_Success(t *testing.T) {
	templates := newFakeScheduleTemplateRepository()
	templates.byID["template-1"] = &models.ScheduleTemplate{ID: "template-1", Status: models.ScheduleTemplateStatusActive}
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), templates, nil)

	created, err := svc.Create(context.Background(), "admin-1", baseScheduleChangeInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID == "" {
		t.Error("expected non-empty change ID")
	}
	if created.CreatedBy != "admin-1" {
		t.Errorf("expected created_by=admin-1, got %q", created.CreatedBy)
	}
}

func TestScheduleChangeService_Create_TemplateNotFound(t *testing.T) {
	templates := newFakeScheduleTemplateRepository()
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), templates, nil)

	_, err := svc.Create(context.Background(), "admin-1", baseScheduleChangeInput(t))
	if !errors.Is(err, repositories.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestScheduleChangeService_Create_DuplicateForSameDate(t *testing.T) {
	templates := newFakeScheduleTemplateRepository()
	templates.byID["template-1"] = &models.ScheduleTemplate{ID: "template-1", Status: models.ScheduleTemplateStatusActive}
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), templates, nil)

	if _, err := svc.Create(context.Background(), "admin-1", baseScheduleChangeInput(t)); err != nil {
		t.Fatalf("unexpected error creating first change: %v", err)
	}

	_, err := svc.Create(context.Background(), "admin-1", baseScheduleChangeInput(t))
	if !errors.Is(err, ErrScheduleChangeConflict) {
		t.Fatalf("expected ErrScheduleChangeConflict, got %v", err)
	}
}

func TestScheduleChangeService_Create_ValidationByType(t *testing.T) {
	templates := newFakeScheduleTemplateRepository()
	templates.byID["template-1"] = &models.ScheduleTemplate{ID: "template-1", Status: models.ScheduleTemplateStatusActive}
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), templates, nil)
	ctx := context.Background()

	// replace_teacher без new_teacher_id
	in := baseScheduleChangeInput(t)
	in.ChangeType = models.ScheduleChangeReplaceTeacher
	if _, err := svc.Create(ctx, "admin-1", in); !errors.Is(err, ErrScheduleChangeValidation) {
		t.Errorf("expected validation error for replace_teacher without teacher, got %v", err)
	}

	// replace_room без new_room_id
	in.ChangeType = models.ScheduleChangeReplaceRoom
	if _, err := svc.Create(ctx, "admin-1", in); !errors.Is(err, ErrScheduleChangeValidation) {
		t.Errorf("expected validation error for replace_room without room, got %v", err)
	}

	// reschedule_time без времён
	in.ChangeType = models.ScheduleChangeRescheduleTime
	if _, err := svc.Create(ctx, "admin-1", in); !errors.Is(err, ErrScheduleChangeValidation) {
		t.Errorf("expected validation error for reschedule_time without times, got %v", err)
	}

	// reschedule_time с end <= start
	start := mustParseClock(t, "10:00")
	end := mustParseClock(t, "09:00")
	in.NewStartTime = &start
	in.NewEndTime = &end
	if _, err := svc.Create(ctx, "admin-1", in); !errors.Is(err, ErrScheduleChangeValidation) {
		t.Errorf("expected validation error for end before start, got %v", err)
	}

	// move без new_date
	in.ChangeType = models.ScheduleChangeMove
	in.NewStartTime, in.NewEndTime = nil, nil
	if _, err := svc.Create(ctx, "admin-1", in); !errors.Is(err, ErrScheduleChangeValidation) {
		t.Errorf("expected validation error for move without new_date, got %v", err)
	}

	// move с new_date == change_date
	sameDate := mustParseDate(t, "2024-03-11")
	in.NewDate = &sameDate
	if _, err := svc.Create(ctx, "admin-1", in); !errors.Is(err, ErrScheduleChangeValidation) {
		t.Errorf("expected validation error for move to the same date, got %v", err)
	}

	// неизвестный тип
	in.ChangeType = "bogus"
	in.NewDate = nil
	if _, err := svc.Create(ctx, "admin-1", in); !errors.Is(err, ErrScheduleChangeValidation) {
		t.Errorf("expected validation error for unknown type, got %v", err)
	}
}

func TestScheduleChangeService_Create_ValidTypesPass(t *testing.T) {
	templates := newFakeScheduleTemplateRepository()
	templates.byID["template-1"] = &models.ScheduleTemplate{ID: "template-1", Status: models.ScheduleTemplateStatusActive}
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), templates, nil)
	ctx := context.Background()
	admin := "admin-1"

	// Каждая замена на свою дату, чтобы не ловить conflict.
	date := 10
	newDate := func() time.Time {
		date++
		return mustParseDate(t, "2024-03-1"+string(rune('0'+date-10)))
	}

	teacher := "teacher-2"
	in := baseScheduleChangeInput(t)
	in.ChangeType = models.ScheduleChangeReplaceTeacher
	in.ChangeDate = newDate()
	in.NewTeacherID = &teacher
	if _, err := svc.Create(ctx, admin, in); err != nil {
		t.Errorf("replace_teacher should pass: %v", err)
	}

	room := "room-2"
	in.ChangeType = models.ScheduleChangeReplaceRoom
	in.ChangeDate = newDate()
	in.NewTeacherID = nil
	in.NewRoomID = &room
	if _, err := svc.Create(ctx, admin, in); err != nil {
		t.Errorf("replace_room should pass: %v", err)
	}

	in.ChangeType = models.ScheduleChangeRescheduleTime
	in.ChangeDate = newDate()
	start := mustParseClock(t, "10:00")
	end := mustParseClock(t, "11:30")
	in.NewStartTime, in.NewEndTime = &start, &end
	if _, err := svc.Create(ctx, admin, in); err != nil {
		t.Errorf("reschedule_time should pass: %v", err)
	}

	in.ChangeType = models.ScheduleChangeMove
	in.ChangeDate = newDate()
	in.NewStartTime, in.NewEndTime = nil, nil
	moved := newDate()
	in.NewDate = &moved
	if _, err := svc.Create(ctx, admin, in); err != nil {
		t.Errorf("move should pass: %v", err)
	}
}

func TestScheduleChangeService_List_BothFiltersRejected(t *testing.T) {
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), newFakeScheduleTemplateRepository(), nil)

	from := mustParseDate(t, "2024-03-11")
	to := mustParseDate(t, "2024-03-17")
	_, err := svc.List(context.Background(), from, to, "group-1", "teacher-1")
	if !errors.Is(err, ErrScheduleChangeValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestScheduleChangeService_List_InvalidRange(t *testing.T) {
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), newFakeScheduleTemplateRepository(), nil)

	from := mustParseDate(t, "2024-03-17")
	to := mustParseDate(t, "2024-03-11")
	_, err := svc.List(context.Background(), from, to, "", "")
	if err != ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestScheduleChangeService_Update_Success(t *testing.T) {
	templates := newFakeScheduleTemplateRepository()
	templates.byID["template-1"] = &models.ScheduleTemplate{ID: "template-1", Status: models.ScheduleTemplateStatusActive}
	repo := newFakeScheduleChangeRepository()
	svc := NewScheduleChangeService(repo, templates, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, "admin-1", baseScheduleChangeInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reason := "teacher is sick"
	in := baseScheduleChangeInput(t)
	in.Reason = &reason

	updated, err := svc.Update(ctx, created.ID, in)
	if err != nil {
		t.Fatalf("unexpected error updating: %v", err)
	}
	if updated.Reason == nil || *updated.Reason != reason {
		t.Errorf("expected updated reason %q, got %v", reason, updated.Reason)
	}
}

func TestScheduleChangeService_Update_NotFound(t *testing.T) {
	svc := NewScheduleChangeService(newFakeScheduleChangeRepository(), newFakeScheduleTemplateRepository(), nil)

	_, err := svc.Update(context.Background(), "missing", baseScheduleChangeInput(t))
	if !errors.Is(err, repositories.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestScheduleChangeService_Delete(t *testing.T) {
	templates := newFakeScheduleTemplateRepository()
	templates.byID["template-1"] = &models.ScheduleTemplate{ID: "template-1", Status: models.ScheduleTemplateStatusActive}
	repo := newFakeScheduleChangeRepository()
	svc := NewScheduleChangeService(repo, templates, nil)
	ctx := context.Background()

	created, err := svc.Create(ctx, "admin-1", baseScheduleChangeInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
	if _, ok := repo.byID[created.ID]; ok {
		t.Error("expected change to be removed after delete")
	}
}
