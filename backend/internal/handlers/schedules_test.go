package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// fakeScheduleTemplateRepository — заглушка для HTTP-тестов хендлеров
// расписания (изолирована от services-пакета, т.к. живёт в другом Go-пакете).
// Поддерживает реальное создание/обновление в памяти, чтобы можно было
// проверить CRUD-хендлеры и проверку конфликтов (Phase 5).
type fakeScheduleTemplateRepository struct {
	byID    map[string]*models.ScheduleTemplate
	byGroup map[string][]*models.ScheduleTemplate
}

func (f *fakeScheduleTemplateRepository) FindByID(_ context.Context, id string) (*models.ScheduleTemplate, error) {
	if f.byID != nil {
		if t, ok := f.byID[id]; ok {
			return t, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeScheduleTemplateRepository) ListByGroup(_ context.Context, groupID string) ([]*models.ScheduleTemplate, error) {
	return f.byGroup[groupID], nil
}

func (f *fakeScheduleTemplateRepository) ListByTeacher(_ context.Context, teacherID string) ([]*models.ScheduleTemplate, error) {
	var result []*models.ScheduleTemplate
	for _, t := range f.byID {
		if t.TeacherID == teacherID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (f *fakeScheduleTemplateRepository) ListByRoom(_ context.Context, roomID string) ([]*models.ScheduleTemplate, error) {
	var result []*models.ScheduleTemplate
	for _, t := range f.byID {
		if t.RoomID == roomID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (f *fakeScheduleTemplateRepository) Create(_ context.Context, t *models.ScheduleTemplate) (string, error) {
	if f.byID == nil {
		f.byID = make(map[string]*models.ScheduleTemplate)
	}
	if t.ID == "" {
		t.ID = fmt.Sprintf("tpl-%d", len(f.byID)+1)
	}
	f.byID[t.ID] = t
	return t.ID, nil
}

func (f *fakeScheduleTemplateRepository) Update(_ context.Context, t *models.ScheduleTemplate) error {
	if f.byID == nil {
		f.byID = make(map[string]*models.ScheduleTemplate)
	}
	f.byID[t.ID] = t
	return nil
}

func (f *fakeScheduleTemplateRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

// ListActiveStudentUserIDsByGroup — заглушка для уведомлений (Phase 8).
func (f *fakeScheduleTemplateRepository) ListActiveStudentUserIDsByGroup(_ context.Context, groupID string) ([]string, error) {
	return nil, nil
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
	return NewSchedulesHandler(services.NewScheduleService(repo, nil), services.NewScheduleAdminService(repo))
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

func validScheduleTemplateBody() []byte {
	body, _ := json.Marshal(map[string]any{
		"group_id":    "group-1",
		"subject_id":  "subject-1",
		"teacher_id":  "teacher-1",
		"room_id":     "room-1",
		"day_of_week": 1,
		"start_time":  "09:00",
		"end_time":    "10:30",
		"valid_from":  "2024-01-01",
	})
	return body
}

func TestCreateTemplate_Success(t *testing.T) {
	h := newTestSchedulesHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(validScheduleTemplateBody()))
	rec := httptest.NewRecorder()

	h.CreateTemplate(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateTemplate_MissingFields(t *testing.T) {
	h := newTestSchedulesHandler()
	body, _ := json.Marshal(map[string]any{"group_id": "group-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.CreateTemplate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateTemplate_Conflict(t *testing.T) {
	h := newTestSchedulesHandler()

	first := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(validScheduleTemplateBody()))
	firstRec := httptest.NewRecorder()
	h.CreateTemplate(firstRec, first)
	if firstRec.Code != http.StatusCreated {
		t.Fatalf("expected first create to succeed, got %d: %s", firstRec.Code, firstRec.Body.String())
	}

	// Same teacher/room/day/time -> should conflict.
	second := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(validScheduleTemplateBody()))
	secondRec := httptest.NewRecorder()
	h.CreateTemplate(secondRec, second)

	if secondRec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", secondRec.Code, secondRec.Body.String())
	}
}

func TestUpdateTemplate_NotFound(t *testing.T) {
	h := newTestSchedulesHandler()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedules/missing", bytes.NewReader(validScheduleTemplateBody()))
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	h.UpdateTemplate(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteTemplate_Success(t *testing.T) {
	h := newTestSchedulesHandler()

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(validScheduleTemplateBody()))
	createRec := httptest.NewRecorder()
	h.CreateTemplate(createRec, createReq)
	var created scheduleTemplateDTO
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode created template: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/schedules/"+created.ID, nil)
	deleteReq.SetPathValue("id", created.ID)
	deleteRec := httptest.NewRecorder()
	h.DeleteTemplate(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", deleteRec.Code)
	}
}
