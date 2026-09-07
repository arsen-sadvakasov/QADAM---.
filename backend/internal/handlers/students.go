package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// StudentsHandler содержит HTTP-хендлеры для /api/v1/students/*
// (раздел 35 API Plan). Доступ: Curator (своя группа) и Admin —
// проверка владения группой в CuratorService.
type StudentsHandler struct {
	curator *services.CuratorService
}

// NewStudentsHandler создаёт StudentsHandler с внедрённым сервисом.
func NewStudentsHandler(curator *services.CuratorService) *StudentsHandler {
	return &StudentsHandler{curator: curator}
}

type studentDTO struct {
	ID                string  `json:"id"`
	UserID            *string `json:"user_id"`
	GroupID           string  `json:"group_id"`
	Status            string  `json:"status"`
	AcademicStatusID  *string `json:"academic_status_id"`
	ScholarshipStatus *string `json:"scholarship_status"`
	FullName          *string `json:"full_name"`
	Email             *string `json:"email"`
	Phone             *string `json:"phone"`
	AvatarURL         *string `json:"avatar_url"`
}

func studentDTOFromModel(s *models.Student) studentDTO {
	return studentDTO{
		ID:                s.ID,
		UserID:            s.UserID,
		GroupID:           s.GroupID,
		Status:            string(s.Status),
		AcademicStatusID:  s.AcademicStatusID,
		ScholarshipStatus: s.ScholarshipStatus,
		FullName:          s.FullName,
		Email:             s.Email,
		Phone:             s.Phone,
		AvatarURL:         s.AvatarURL,
	}
}

// List обрабатывает GET /api/v1/students?group_id=&status=.
func (h *StudentsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	students, err := h.curator.ListStudents(r.Context(), userID, role == models.RoleAdmin,
		r.URL.Query().Get("group_id"), r.URL.Query().Get("status"))
	if err != nil {
		handleCuratorError(w, err)
		return
	}

	dtos := make([]studentDTO, 0, len(students))
	for _, s := range students {
		dtos = append(dtos, studentDTOFromModel(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"students": dtos})
}

// Get обрабатывает GET /api/v1/students/{id}.
func (h *StudentsHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	student, err := h.curator.GetStudent(r.Context(), userID, role == models.RoleAdmin, r.PathValue("id"))
	if err != nil {
		handleCuratorError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, studentDTOFromModel(student))
}

type studentWriteRequest struct {
	UserID            *string `json:"user_id"`
	GroupID           string  `json:"group_id"`
	Status            string  `json:"status"`
	AcademicStatusID  *string `json:"academic_status_id"`
	ScholarshipStatus *string `json:"scholarship_status"`
}

// Create обрабатывает POST /api/v1/students — добавление студента.
func (h *StudentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req studentWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	student, err := h.curator.CreateStudent(r.Context(), userID, role == models.RoleAdmin, services.StudentInput{
		UserID:            req.UserID,
		GroupID:           req.GroupID,
		Status:            models.StudentStatus(req.Status),
		AcademicStatusID:  req.AcademicStatusID,
		ScholarshipStatus: req.ScholarshipStatus,
	})
	if err != nil {
		handleCuratorError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, studentDTOFromModel(student))
}

// Update обрабатывает PATCH /api/v1/students/{id} — редактирование/смена
// статуса.
func (h *StudentsHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := r.PathValue("id")

	var req studentWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	student, err := h.curator.UpdateStudent(r.Context(), userID, role == models.RoleAdmin, id, services.StudentInput{
		UserID:            req.UserID,
		GroupID:           req.GroupID,
		Status:            models.StudentStatus(req.Status),
		AcademicStatusID:  req.AcademicStatusID,
		ScholarshipStatus: req.ScholarshipStatus,
	})
	if err != nil {
		handleCuratorError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, studentDTOFromModel(student))
}

// Delete обрабатывает DELETE /api/v1/students/{id} — soft delete.
func (h *StudentsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.curator.DeleteStudent(r.Context(), userID, role == models.RoleAdmin, r.PathValue("id")); err != nil {
		handleCuratorError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleCuratorError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		writeError(w, http.StatusNotFound, "student or group not found")
	case errors.Is(err, services.ErrCuratorForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrCuratorValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
