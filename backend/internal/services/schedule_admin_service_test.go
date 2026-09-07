package services

import (
	"context"
	"testing"

	"github.com/qadam/backend/internal/models"
)

func baseScheduleTemplateInput(t *testing.T) ScheduleTemplateInput {
	return ScheduleTemplateInput{
		GroupID:    "group-1",
		SubjectID:  "subject-1",
		TeacherID:  "teacher-1",
		RoomID:     "room-1",
		DayOfWeek:  1,
		StartTime:  mustParseClock(t, "09:00"),
		EndTime:    mustParseClock(t, "10:30"),
		LessonType: models.LessonTypeLecture,
		WeekParity: models.WeekParityAll,
		ValidFrom:  mustParseDate(t, "2024-01-01"),
		Status:     models.ScheduleTemplateStatusActive,
	}
}

func TestScheduleAdminService_Create_Success(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	created, err := svc.Create(context.Background(), baseScheduleTemplateInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID == "" {
		t.Error("expected non-empty template ID")
	}
}

func TestScheduleAdminService_Create_InvalidDayOfWeek(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	in := baseScheduleTemplateInput(t)
	in.DayOfWeek = 8

	_, err := svc.Create(context.Background(), in)
	if err == nil {
		t.Fatal("expected error for invalid day_of_week")
	}
}

func TestScheduleAdminService_Create_EndBeforeStart(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	in := baseScheduleTemplateInput(t)
	in.EndTime = mustParseClock(t, "08:00") // before start_time 09:00

	_, err := svc.Create(context.Background(), in)
	if err == nil {
		t.Fatal("expected error for end_time before start_time")
	}
}

func TestScheduleAdminService_Create_TeacherConflict(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	first := baseScheduleTemplateInput(t)
	if _, err := svc.Create(context.Background(), first); err != nil {
		t.Fatalf("unexpected error creating first template: %v", err)
	}

	// Same teacher, overlapping time, different group/room -> should conflict.
	second := baseScheduleTemplateInput(t)
	second.GroupID = "group-2"
	second.RoomID = "room-2"
	second.StartTime = mustParseClock(t, "10:00") // overlaps 09:00-10:30

	_, err := svc.Create(context.Background(), second)
	if err != ErrScheduleConflict {
		t.Fatalf("expected ErrScheduleConflict, got %v", err)
	}
}

func TestScheduleAdminService_Create_RoomConflict(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	first := baseScheduleTemplateInput(t)
	if _, err := svc.Create(context.Background(), first); err != nil {
		t.Fatalf("unexpected error creating first template: %v", err)
	}

	// Same room, overlapping time, different teacher/group -> should conflict.
	second := baseScheduleTemplateInput(t)
	second.GroupID = "group-2"
	second.TeacherID = "teacher-2"
	second.StartTime = mustParseClock(t, "10:00")

	_, err := svc.Create(context.Background(), second)
	if err != ErrScheduleConflict {
		t.Fatalf("expected ErrScheduleConflict, got %v", err)
	}
}

func TestScheduleAdminService_Create_NoConflict_DifferentDay(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	first := baseScheduleTemplateInput(t)
	if _, err := svc.Create(context.Background(), first); err != nil {
		t.Fatalf("unexpected error creating first template: %v", err)
	}

	second := baseScheduleTemplateInput(t)
	second.DayOfWeek = 2 // Tuesday instead of Monday

	if _, err := svc.Create(context.Background(), second); err != nil {
		t.Fatalf("expected no conflict for a different day, got %v", err)
	}
}

func TestScheduleAdminService_Create_NoConflict_NonOverlappingParity(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	first := baseScheduleTemplateInput(t)
	first.WeekParity = models.WeekParityOdd
	if _, err := svc.Create(context.Background(), first); err != nil {
		t.Fatalf("unexpected error creating first template: %v", err)
	}

	second := baseScheduleTemplateInput(t)
	second.WeekParity = models.WeekParityEven

	if _, err := svc.Create(context.Background(), second); err != nil {
		t.Fatalf("expected no conflict for non-overlapping week parity, got %v", err)
	}
}

func TestScheduleAdminService_Create_NoConflict_AdjacentTimes(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	first := baseScheduleTemplateInput(t)
	if _, err := svc.Create(context.Background(), first); err != nil {
		t.Fatalf("unexpected error creating first template: %v", err)
	}

	// Starts exactly when the first one ends -> not overlapping.
	second := baseScheduleTemplateInput(t)
	second.StartTime = mustParseClock(t, "10:30")
	second.EndTime = mustParseClock(t, "12:00")

	if _, err := svc.Create(context.Background(), second); err != nil {
		t.Fatalf("expected no conflict for adjacent (non-overlapping) times, got %v", err)
	}
}

func TestScheduleAdminService_Update_ExcludesSelfFromConflictCheck(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	created, err := svc.Create(context.Background(), baseScheduleTemplateInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Update the same template with a slightly different time - should not
	// conflict with itself.
	updateInput := baseScheduleTemplateInput(t)
	updateInput.StartTime = mustParseClock(t, "09:15")
	updateInput.EndTime = mustParseClock(t, "10:45")

	updated, err := svc.Update(context.Background(), created.ID, updateInput)
	if err != nil {
		t.Fatalf("unexpected error updating template: %v", err)
	}
	if updated.StartTime.Format("15:04") != "09:15" {
		t.Errorf("expected updated start time 09:15, got %v", updated.StartTime)
	}
}

func TestScheduleAdminService_Update_ConflictsWithOtherTemplate(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	first, err := svc.Create(context.Background(), baseScheduleTemplateInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	second := baseScheduleTemplateInput(t)
	second.DayOfWeek = 2 // different day, no conflict initially
	secondCreated, err := svc.Create(context.Background(), second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Now try to move the second template onto the same day/time as the first.
	conflictingUpdate := baseScheduleTemplateInput(t) // DayOfWeek 1, same time as first

	_, err = svc.Update(context.Background(), secondCreated.ID, conflictingUpdate)
	if err != ErrScheduleConflict {
		t.Fatalf("expected ErrScheduleConflict, got %v", err)
	}
	_ = first
}

func TestScheduleAdminService_Delete(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	created, err := svc.Create(context.Background(), baseScheduleTemplateInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("unexpected error deleting template: %v", err)
	}

	if _, ok := repo.byID[created.ID]; ok {
		t.Error("expected template to be removed from repository after delete")
	}
}

func TestScheduleAdminService_Create_InvalidValidToBeforeValidFrom(t *testing.T) {
	repo := newFakeScheduleTemplateRepository()
	svc := NewScheduleAdminService(repo)

	in := baseScheduleTemplateInput(t)
	invalidValidTo := mustParseDate(t, "2023-12-31") // before ValidFrom 2024-01-01
	in.ValidTo = &invalidValidTo

	_, err := svc.Create(context.Background(), in)
	if err != ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}
