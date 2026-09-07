package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// ScheduleChangesHandler содержит HTTP-хендлеры для /api/v1/schedule-changes/*
// (раздел 35 API Plan). CRUD — Admin; просмотр списка — все авторизованные
// с фильтрацией (уведомления затронутым пользователям — Phase 8).
type ScheduleChangesHandler struct {
	changes *services.ScheduleChangeService
}

// NewScheduleChangesHandler создаёт ScheduleChangesHandler с внедрённым
// сервисом.
func NewScheduleChangesHandler(changes *services.ScheduleChangeService) *ScheduleChangesHandler {
	return &ScheduleChangesHandler{changes: changes}
}

type scheduleChangeDTO struct {
	ID                 string  `json:"id"`
	ScheduleTemplateID string  `json:"schedule_template_id"`
	ChangeDate         string  `json:"change_date"`
	ChangeType         string  `json:"change_type"`
	Reason             *string `json:"reason"`
	CreatedBy          string  `json:"created_by"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`

	// "Было" — данные исходного занятия
	OriginalTeacherName string `json:"original_teacher_name"`
	OriginalRoomNumber  string `json:"original_room_number"`
	OriginalStartTime   string `json:"original_start_time"`
	OriginalEndTime     string `json:"original_end_time"`
	SubjectName         string `json:"subject_name"`
	GroupName           string `json:"group_name"`

	// "Стало" — новые значения (null, если поле не меняется)
	NewTeacherID   *string `json:"new_teacher_id"`
	NewTeacherName *string `json:"new_teacher_name"`
	NewRoomID      *string `json:"new_room_id"`
	NewRoomNumber  *string `json:"new_room_number"`
	NewStartTime   *string `json:"new_start_time"`
	NewEndTime     *string `json:"new_end_time"`
	NewDate        *string `json:"new_date"`
}

func scheduleChangeDTOFromModel(c *models.ScheduleChangeWithNames) scheduleChangeDTO {
	dto := scheduleChangeDTO{
		ID:                  c.ID,
		ScheduleTemplateID:  c.ScheduleTemplateID,
		ChangeDate:          c.ChangeDate.Format(dateLayout),
		ChangeType:          string(c.ChangeType),
		Reason:              c.Reason,
		CreatedBy:           c.CreatedBy,
		CreatedAt:           c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           c.UpdatedAt.Format(time.RFC3339),
		OriginalTeacherName: c.OriginalTeacherName,
		OriginalRoomNumber:  c.OriginalRoomNumber,
		OriginalStartTime:   c.OriginalStartTime.Format(timeLayout),
		OriginalEndTime:     c.OriginalEndTime.Format(timeLayout),
		SubjectName:         c.SubjectName,
		GroupName:           c.GroupName,
		NewTeacherID:        c.NewTeacherID,
		NewTeacherName:      c.NewTeacherName,
		NewRoomID:           c.NewRoomID,
		NewRoomNumber:       c.NewRoomNumber,
	}
	if c.NewStartTime != nil {
		formatted := c.NewStartTime.Format(timeLayout)
		dto.NewStartTime = &formatted
	}
	if c.NewEndTime != nil {
		formatted := c.NewEndTime.Format(timeLayout)
		dto.NewEndTime = &formatted
	}
	if c.NewDate != nil {
		formatted := c.NewDate.Format(dateLayout)
		dto.NewDate = &formatted
	}
	return dto
}

func scheduleChangeDTOs(changes []*models.ScheduleChangeWithNames) []scheduleChangeDTO {
	result := make([]scheduleChangeDTO, 0, len(changes))
	for _, c := range changes {
		result = append(result, scheduleChangeDTOFromModel(c))
	}
	return result
}

type scheduleChangeWriteRequest struct {
	ScheduleTemplateID string  `json:"schedule_template_id"`
	ChangeDate         string  `json:"change_date"`
	ChangeType         string  `json:"change_type"`
	NewTeacherID       *string `json:"new_teacher_id"`
	NewRoomID          *string `json:"new_room_id"`
	NewStartTime       *string `json:"new_start_time"`
	NewEndTime         *string `json:"new_end_time"`
	NewDate            *string `json:"new_date"`
	Reason             *string `json:"reason"`
}

var validChangeTypes = map[string]bool{
	string(models.ScheduleChangeReplaceTeacher): true,
	string(models.ScheduleChangeReplaceRoom):    true,
	string(models.ScheduleChangeRescheduleTime): true,
	string(models.ScheduleChangeCancel):         true,
	string(models.ScheduleChangeMove):           true,
}

// toInput преобразует JSON-запрос в services.ScheduleChangeInput, парся
// даты (YYYY-MM-DD) и времена (HH:MM). partial=true допускает пустые
// schedule_template_id/change_date при редактировании (они не меняются).
func (req scheduleChangeWriteRequest) toInput(partial bool) (services.ScheduleChangeInput, error) {
	in := services.ScheduleChangeInput{
		ScheduleTemplateID: req.ScheduleTemplateID,
		NewTeacherID:       req.NewTeacherID,
		NewRoomID:          req.NewRoomID,
		Reason:             req.Reason,
	}

	if req.ChangeType != "" {
		if !validChangeTypes[req.ChangeType] {
			return services.ScheduleChangeInput{}, errors.New("invalid change_type")
		}
		in.ChangeType = models.ScheduleChangeType(req.ChangeType)
	}

	if req.ChangeDate != "" {
		parsed, err := time.Parse(dateLayout, req.ChangeDate)
		if err != nil {
			return services.ScheduleChangeInput{}, errors.New("invalid change_date format, expected YYYY-MM-DD")
		}
		in.ChangeDate = parsed
	} else if !partial {
		return services.ScheduleChangeInput{}, errors.New("change_date is required")
	}

	if req.NewStartTime != nil && *req.NewStartTime != "" {
		parsed, err := time.Parse(timeLayout, *req.NewStartTime)
		if err != nil {
			return services.ScheduleChangeInput{}, errors.New("invalid new_start_time format, expected HH:MM")
		}
		in.NewStartTime = &parsed
	}
	if req.NewEndTime != nil && *req.NewEndTime != "" {
		parsed, err := time.Parse(timeLayout, *req.NewEndTime)
		if err != nil {
			return services.ScheduleChangeInput{}, errors.New("invalid new_end_time format, expected HH:MM")
		}
		in.NewEndTime = &parsed
	}
	if req.NewDate != nil && *req.NewDate != "" {
		parsed, err := time.Parse(dateLayout, *req.NewDate)
		if err != nil {
			return services.ScheduleChangeInput{}, errors.New("invalid new_date format, expected YYYY-MM-DD")
		}
		in.NewDate = &parsed
	}

	return in, nil
}

// List обрабатывает GET /api/v1/schedule-changes?from=&to=&group_id=&teacher_id=
// Если from/to не заданы, используется текущая неделя (как в /schedules).
func (h *ScheduleChangesHandler) List(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	changes, err := h.changes.List(r.Context(), from, to,
		r.URL.Query().Get("group_id"), r.URL.Query().Get("teacher_id"))
	if err != nil {
		handleScheduleChangeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"changes": scheduleChangeDTOs(changes)})
}

// Get обрабатывает GET /api/v1/schedule-changes/{id}.
func (h *ScheduleChangesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	change, err := h.changes.Get(r.Context(), id)
	if err != nil {
		handleNotFoundOr500(w, err, "schedule change not found")
		return
	}
	writeJSON(w, http.StatusOK, scheduleChangeDTOFromModel(&models.ScheduleChangeWithNames{ScheduleChange: *change}))
}

// Create обрабатывает POST /api/v1/schedule-changes — создание замены
// ("было/стало"). Admin only.
func (h *ScheduleChangesHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req scheduleChangeWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ScheduleTemplateID == "" {
		writeError(w, http.StatusBadRequest, "schedule_template_id is required")
		return
	}
	if req.ChangeType == "" {
		writeError(w, http.StatusBadRequest, "change_type is required")
		return
	}

	in, err := req.toInput(false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	change, err := h.changes.Create(r.Context(), userID, in)
	if err != nil {
		handleScheduleChangeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, scheduleChangeDTOFromModel(&models.ScheduleChangeWithNames{ScheduleChange: *change}))
}

// Update обрабатывает PATCH /api/v1/schedule-changes/{id}. Admin only.
func (h *ScheduleChangesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req scheduleChangeWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	in, err := req.toInput(true)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// PATCH может не содержать change_type — тогда берём текущий, чтобы
	// валидация полей по типу отработала корректно.
	if in.ChangeType == "" {
		current, err := h.changes.Get(r.Context(), id)
		if err != nil {
			handleScheduleChangeError(w, err)
			return
		}
		in.ChangeType = current.ChangeType
	}

	change, err := h.changes.Update(r.Context(), id, in)
	if err != nil {
		handleScheduleChangeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, scheduleChangeDTOFromModel(&models.ScheduleChangeWithNames{ScheduleChange: *change}))
}

// Delete обрабатывает DELETE /api/v1/schedule-changes/{id}. Admin only.
func (h *ScheduleChangesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.changes.Delete(r.Context(), id); err != nil {
		handleNotFoundOr500(w, err, "schedule change not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleScheduleChangeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		writeError(w, http.StatusNotFound, "schedule template not found")
	case errors.Is(err, services.ErrScheduleChangeConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrScheduleChangeValidation),
		errors.Is(err, services.ErrInvalidDateRange):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
