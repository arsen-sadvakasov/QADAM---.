package services

import (
	"context"
	"testing"
	"time"

	"github.com/qadam/backend/internal/models"
)

// scheduleWithChangesTest — хелпер: сервис расписания с шаблоном по
// понедельникам и репозиторием замен, в который можно добавлять замены.
type scheduleWithChangesTest struct {
	templates *fakeScheduleTemplateRepository
	changes   *fakeScheduleChangeRepository
	svc       *ScheduleService
}

func newScheduleWithChangesTest(t *testing.T) *scheduleWithChangesTest {
	t.Helper()
	templates := newFakeScheduleTemplateRepository()
	templates.add(baseTemplate(t)) // по понедельникам 09:00-10:30
	changes := newFakeScheduleChangeRepository()
	return &scheduleWithChangesTest{
		templates: templates,
		changes:   changes,
		svc:       NewScheduleService(templates, changes),
	}
}

func (s *scheduleWithChangesTest) addChange(t *testing.T, changeType models.ScheduleChangeType, date string, mutate func(*models.ScheduleChange)) {
	t.Helper()
	c := &models.ScheduleChange{
		ID:                 "chg-" + date + "-" + string(changeType),
		ScheduleTemplateID: "tpl-1",
		ChangeDate:         mustParseDate(t, date),
		ChangeType:         changeType,
	}
	if mutate != nil {
		mutate(c)
	}
	s.changes.byID[c.ID] = c
}

// monday возвращает дату понедельника в диапазоне шаблона.
func mondayDate() time.Time {
	return time.Date(2024, 3, 11, 0, 0, 0, 0, time.UTC) // понедельник
}

func TestScheduleService_CancelledLessonIsHidden(t *testing.T) {
	tc := newScheduleWithChangesTest(t)
	tc.addChange(t, models.ScheduleChangeCancel, "2024-03-11", nil)

	from := mondayDate()
	to := from.AddDate(0, 0, 6)

	lessons, err := tc.svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lessons) != 0 {
		t.Errorf("expected cancelled lesson to be hidden, got %d lessons", len(lessons))
	}
}

func TestScheduleService_ReplaceTeacherApplied(t *testing.T) {
	tc := newScheduleWithChangesTest(t)
	newTeacher := "teacher-2"
	tc.addChange(t, models.ScheduleChangeReplaceTeacher, "2024-03-11", func(c *models.ScheduleChange) {
		c.NewTeacherID = &newTeacher
	})

	from := mondayDate()
	to := from.AddDate(0, 0, 6)

	lessons, err := tc.svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("expected 1 lesson, got %d", len(lessons))
	}
	if lessons[0].TeacherID != newTeacher {
		t.Errorf("expected teacher replaced to %q, got %q", newTeacher, lessons[0].TeacherID)
	}
}

func TestScheduleService_ReplaceRoomApplied(t *testing.T) {
	tc := newScheduleWithChangesTest(t)
	newRoom := "room-2"
	tc.addChange(t, models.ScheduleChangeReplaceRoom, "2024-03-11", func(c *models.ScheduleChange) {
		c.NewRoomID = &newRoom
	})

	from := mondayDate()
	to := from.AddDate(0, 0, 6)

	lessons, err := tc.svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("expected 1 lesson, got %d", len(lessons))
	}
	if lessons[0].RoomID != newRoom {
		t.Errorf("expected room replaced to %q, got %q", newRoom, lessons[0].RoomID)
	}
}

func TestScheduleService_RescheduleTimeApplied(t *testing.T) {
	tc := newScheduleWithChangesTest(t)
	newStart := mustParseClock(t, "12:00")
	newEnd := mustParseClock(t, "13:30")
	tc.addChange(t, models.ScheduleChangeRescheduleTime, "2024-03-11", func(c *models.ScheduleChange) {
		c.NewStartTime = &newStart
		c.NewEndTime = &newEnd
	})

	from := mondayDate()
	to := from.AddDate(0, 0, 6)

	lessons, err := tc.svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("expected 1 lesson, got %d", len(lessons))
	}
	if !lessons[0].StartTime.Equal(newStart) || !lessons[0].EndTime.Equal(newEnd) {
		t.Errorf("expected time 12:00-13:30, got %v-%v", lessons[0].StartTime, lessons[0].EndTime)
	}
}

func TestScheduleService_MoveHidesOnOriginalDateAndAppearsOnNewDate(t *testing.T) {
	tc := newScheduleWithChangesTest(t)
	newDate := mustParseDate(t, "2024-03-12") // вторник
	tc.addChange(t, models.ScheduleChangeMove, "2024-03-11", func(c *models.ScheduleChange) {
		c.NewDate = &newDate
	})

	from := mondayDate()
	to := from.AddDate(0, 0, 6) // вся неделя 11-17 марта

	lessons, err := tc.svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("expected exactly 1 lesson (on the new date), got %d", len(lessons))
	}
	if !normalizeDate(lessons[0].Date).Equal(normalizeDate(newDate)) {
		t.Errorf("expected moved lesson on 2024-03-12, got %v", lessons[0].Date)
	}
}

func TestScheduleService_ChangeOnOtherDateDoesNotAffect(t *testing.T) {
	tc := newScheduleWithChangesTest(t)
	tc.addChange(t, models.ScheduleChangeCancel, "2024-03-18", nil) // следующий понедельник, вне диапазона

	from := mondayDate()
	to := from.AddDate(0, 0, 6)

	lessons, err := tc.svc.GetGroupSchedule(context.Background(), "group-1", from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lessons) != 1 {
		t.Errorf("expected 1 lesson (cancel is out of range), got %d", len(lessons))
	}
}

func TestScheduleService_GetLessonDetails_AppliesChange(t *testing.T) {
	tc := newScheduleWithChangesTest(t)
	newTeacher := "teacher-2"
	tc.addChange(t, models.ScheduleChangeReplaceTeacher, "2024-03-11", func(c *models.ScheduleChange) {
		c.NewTeacherID = &newTeacher
	})

	occ, err := tc.svc.GetLessonDetails(context.Background(), "tpl-1", mondayDate())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if occ.TeacherID != newTeacher {
		t.Errorf("expected details with replaced teacher %q, got %q", newTeacher, occ.TeacherID)
	}
}
