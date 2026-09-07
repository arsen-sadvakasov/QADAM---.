package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// fakeScheduleTemplateRepository — минимальная заглушка для HTTP-тестов
// хендлеров расписания (изолирована от services-пакета, т.к. живёт в
// другом Go-пакете).
type fakeScheduleTemplateRepository struct {
	byGroup map[string][]*models.ScheduleTemplate
}

func (f *fakeScheduleTemplateRepository) FindByID(_ context.Context, _ string) (*models.ScheduleTemplate, error) {
	return nil, repositories.ErrNotFound
}

func (f *fakeScheduleTemplateRepository) ListByGroup(_ context.Context, groupID string) ([]*models.ScheduleTemplate, error) {
	return f.byGroup[groupID], nil
}

func (f *fakeScheduleTemplateRepository) ListByTeacher(_ context.Context, _ string) ([]*models.ScheduleTemplate, error) {
	return nil, nil
}

func (f *fakeScheduleTemplateRepository) Create(_ context.Context, t *models.ScheduleTemplate) (string, error) {
	return t.ID, nil
}

func (f *fakeScheduleTemplateRepository) Update(_ context.Context, _ *models.ScheduleTemplate) error {
	return nil
}

func (f *fakeScheduleTemplateRepository) SoftDelete(_ context.Context, _ string) error {
	return nil
}

func newTestSchedulesHandler() *SchedulesHandler {
	clock, _ := time.Parse("15:04", "09:00")
	clockEnd, _ := time.Parse("15:04", "10:30")
	repo := &fakeScheduleTemplateRepository{
		byGroup: map[string][]*models.ScheduleTemplate{
			"group-1": {
				{
					ID:          "tpl-1",
					GroupID:     "group-1",
					SubjectID:   "subject-1",
					TeacherID:   "teacher-1",
					RoomID:      "room-1",
					DayOfWeek:   1,
					StartTime:   clock,
					EndTime:     clockEnd,
					LessonType:  models.LessonTypeLecture,
					WeekParity:  models.WeekParityAll,
					ValidFrom:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					Status:      models.ScheduleTemplateStatusActive,
					SubjectName: "Databases",
					TeacherName: "Ivanov I.I.",
					RoomNumber:  "301",
					GroupName:   "PO-23",
				},
			},
		},
	}
	return NewSchedulesHandler(services.NewScheduleService(repo))
}

func TestGetSchedule_MissingFilters(t *testing.T) {
	h := newTestSchedulesHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
	rec := httptest.NewRecorder()

	h.GetSchedule(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetSchedule_BothFiltersProvided(t *testing.T) {
	h := newTestSchedulesHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules?group_id=group-1&teacher_id=teacher-1", nil)
	rec := httptest.NewRecorder()

	h.GetSchedule(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetSchedule_GroupSchedule_Success(t *testing.T) {
	h := newTestSchedulesHandler()
	// 2024-06-03 is a Monday; 2024-06-09 is the following Sunday.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules?group_id=group-1&from=2024-06-03&to=2024-06-09", nil)
	rec := httptest.NewRecorder()

	h.GetSchedule(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Lessons []lessonOccurrenceDTO `json:"lessons"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Lessons) != 1 {
		t.Fatalf("expected 1 lesson, got %d", len(body.Lessons))
	}
	if body.Lessons[0].SubjectName != "Databases" {
		t.Errorf("expected subject name Databases, got %q", body.Lessons[0].SubjectName)
	}
	if body.Lessons[0].Date != "2024-06-03" {
		t.Errorf("expected date 2024-06-03, got %q", body.Lessons[0].Date)
	}
}

func TestGetSchedule_InvalidDateFormat(t *testing.T) {
	h := newTestSchedulesHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules?group_id=group-1&from=not-a-date", nil)
	rec := httptest.NewRecorder()

	h.GetSchedule(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetLessonDetails_MissingDate(t *testing.T) {
	h := newTestSchedulesHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/lesson/tpl-1", nil)
	req.SetPathValue("id", "tpl-1")
	rec := httptest.NewRecorder()

	h.GetLessonDetails(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetLessonDetails_NotFound(t *testing.T) {
	h := newTestSchedulesHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedules/lesson/does-not-exist?date=2024-06-03", nil)
	req.SetPathValue("id", "does-not-exist")
	rec := httptest.NewRecorder()

	h.GetLessonDetails(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
