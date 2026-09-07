package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// dateLayout — формат даты в query-параметрах API (YYYY-MM-DD).
const dateLayout = "2006-01-02"

// timeLayout — формат времени начала/окончания занятия в ответе API.
const timeLayout = "15:04"

// SchedulesHandler содержит HTTP-хендлеры для /api/v1/schedules/*
// (раздел 35 API Plan). CRUD для schedule_templates (создание/редактирование
// занятий администратором) добавляется в Phase 5 — Admin Panel; здесь —
// только чтение вычисленного расписания (FR-2, FR-3 спецификации).
type SchedulesHandler struct {
	schedule *services.ScheduleService
}

// NewSchedulesHandler создаёт SchedulesHandler с внедрённым ScheduleService.
func NewSchedulesHandler(schedule *services.ScheduleService) *SchedulesHandler {
	return &SchedulesHandler{schedule: schedule}
}

type lessonOccurrenceDTO struct {
	TemplateID  string `json:"template_id"`
	Date        string `json:"date"`
	GroupID     string `json:"group_id"`
	GroupName   string `json:"group_name"`
	SubjectID   string `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	TeacherID   string `json:"teacher_id"`
	TeacherName string `json:"teacher_name"`
	RoomID      string `json:"room_id"`
	RoomNumber  string `json:"room_number"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	LessonType  string `json:"lesson_type"`
}

func lessonDTO(o models.LessonOccurrence) lessonOccurrenceDTO {
	return lessonOccurrenceDTO{
		TemplateID:  o.TemplateID,
		Date:        o.Date.Format(dateLayout),
		GroupID:     o.GroupID,
		GroupName:   o.GroupName,
		SubjectID:   o.SubjectID,
		SubjectName: o.SubjectName,
		TeacherID:   o.TeacherID,
		TeacherName: o.TeacherName,
		RoomID:      o.RoomID,
		RoomNumber:  o.RoomNumber,
		StartTime:   o.StartTime.Format(timeLayout),
		EndTime:     o.EndTime.Format(timeLayout),
		LessonType:  string(o.LessonType),
	}
}

func lessonDTOs(occurrences []models.LessonOccurrence) []lessonOccurrenceDTO {
	result := make([]lessonOccurrenceDTO, 0, len(occurrences))
	for _, o := range occurrences {
		result = append(result, lessonDTO(o))
	}
	return result
}

// GetSchedule обрабатывает GET /api/v1/schedules?group_id=&teacher_id=&from=&to=
// Ровно один из group_id/teacher_id обязателен. from/to — YYYY-MM-DD; если
// не указаны, по умолчанию используется текущая неделя (раздел 20:
// "поддерживаемые представления: сегодня, завтра, неделя, месяц").
func (h *SchedulesHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	groupID := r.URL.Query().Get("group_id")
	teacherID := r.URL.Query().Get("teacher_id")
	if groupID == "" && teacherID == "" {
		writeError(w, http.StatusBadRequest, "group_id or teacher_id is required")
		return
	}
	if groupID != "" && teacherID != "" {
		writeError(w, http.StatusBadRequest, "provide only one of group_id or teacher_id")
		return
	}

	from, to, err := parseDateRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var occurrences []models.LessonOccurrence
	if groupID != "" {
		occurrences, err = h.schedule.GetGroupSchedule(r.Context(), groupID, from, to)
	} else {
		occurrences, err = h.schedule.GetTeacherSchedule(r.Context(), teacherID, from, to)
	}
	if err != nil {
		handleScheduleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"lessons": lessonDTOs(occurrences)})
}

// GetLessonDetails обрабатывает GET /api/v1/schedules/lesson/{id}?date=YYYY-MM-DD
// — детальная карточка занятия (FR-3 спецификации).
func (h *SchedulesHandler) GetLessonDetails(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("id")
	if templateID == "" {
		writeError(w, http.StatusBadRequest, "lesson id is required")
		return
	}

	dateParam := r.URL.Query().Get("date")
	if dateParam == "" {
		writeError(w, http.StatusBadRequest, "date query parameter (YYYY-MM-DD) is required")
		return
	}
	date, err := time.Parse(dateLayout, dateParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date format, expected YYYY-MM-DD")
		return
	}

	occurrence, err := h.schedule.GetLessonDetails(r.Context(), templateID, date)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lesson not found on the given date")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, lessonDTO(*occurrence))
}

func parseDateRange(r *http.Request) (from, to time.Time, err error) {
	now := time.Now().UTC()
	fromParam := r.URL.Query().Get("from")
	toParam := r.URL.Query().Get("to")

	if fromParam == "" {
		from = startOfWeek(now)
	} else {
		from, err = time.Parse(dateLayout, fromParam)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("invalid 'from' date format, expected YYYY-MM-DD")
		}
	}

	if toParam == "" {
		to = from.AddDate(0, 0, 6)
	} else {
		to, err = time.Parse(dateLayout, toParam)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("invalid 'to' date format, expected YYYY-MM-DD")
		}
	}

	if to.Before(from) {
		return time.Time{}, time.Time{}, services.ErrInvalidDateRange
	}
	if to.Sub(from) > time.Duration(services.MaxScheduleRangeDays)*24*time.Hour {
		return time.Time{}, time.Time{}, errors.New("requested date range is too wide")
	}

	return from, to, nil
}

func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysSinceMonday := weekday - 1
	monday := t.AddDate(0, 0, -daysSinceMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

func handleScheduleError(w http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrInvalidDateRange) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
