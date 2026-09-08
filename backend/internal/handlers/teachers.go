package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// TeachersHandler содержит HTTP-хендлеры для /api/v1/teachers/*
// (раздел 35 API Plan). Список/детали — все авторизованные (с
// ограниченным набором полей для student — см. ограничение прав ниже),
// создание/редактирование — Admin.
type TeachersHandler struct {
	teachers *services.TeacherAdminService
}

// NewTeachersHandler создаёт TeachersHandler с внедрённым TeacherAdminService.
func NewTeachersHandler(teachers *services.TeacherAdminService) *TeachersHandler {
	return &TeachersHandler{teachers: teachers}
}

type teacherDTO struct {
	ID        string  `json:"id"`
	FullName  string  `json:"full_name"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	AvatarURL *string `json:"avatar_url"`
}

// teacherDTOFromModel собирает DTO преподавателя. includeContacts — false
// для роли student (раздел 35: "детали — все авторизованные, ограниченный
// набор полей для student").
func teacherDTOFromModel(t *models.Teacher, includeContacts bool) teacherDTO {
	dto := teacherDTO{ID: t.ID, FullName: t.FullName, AvatarURL: t.AvatarURL}
	if includeContacts {
		dto.Email = t.Email
		dto.Phone = t.Phone
	}
	return dto
}

// List обрабатывает GET /api/v1/teachers. Admin, Curator (просмотр).
func (h *TeachersHandler) List(w http.ResponseWriter, r *http.Request) {
	teachers, err := h.teachers.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	includeContacts := roleAllowsTeacherContacts(r)
	dtos := make([]teacherDTO, 0, len(teachers))
	for _, t := range teachers {
		dtos = append(dtos, teacherDTOFromModel(t, includeContacts))
	}
	writeJSON(w, http.StatusOK, map[string]any{"teachers": dtos})
}

// Get обрабатывает GET /api/v1/teachers/{id}. Все авторизованные
// (ограниченный набор полей для student).
func (h *TeachersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := h.teachers.Get(r.Context(), id)
	if err != nil {
		handleNotFoundOr500(w, err, "teacher not found")
		return
	}
	writeJSON(w, http.StatusOK, teacherDTOFromModel(t, roleAllowsTeacherContacts(r)))
}

type createTeacherRequest struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	FullName string  `json:"full_name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
}

// Create обрабатывает POST /api/v1/teachers. Admin only.
func (h *TeachersHandler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createTeacherRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" || req.FullName == "" {
		writeError(w, http.StatusBadRequest, "username, password and full_name are required")
		return
	}

	teacher, err := h.teachers.Create(r.Context(), actorID, services.CreateTeacherInput{
		Username: req.Username,
		Password: req.Password,
		FullName: req.FullName,
		Email:    req.Email,
		Phone:    req.Phone,
	})
	if err != nil {
		handleTeacherAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, teacherDTOFromModel(teacher, true))
}

type assignSubjectRequest struct {
	SubjectID string `json:"subject_id"`
	GroupID   string `json:"group_id"`
}

// AssignSubject обрабатывает PATCH /api/v1/teachers/{id}/subjects —
// назначение преподавателя на предмет в конкретной группе. Admin only.
// (Расширение раздела 35: точечная операция над teacher_subjects, вместо
// перезаписи всего профиля преподавателя.)
func (h *TeachersHandler) AssignSubject(w http.ResponseWriter, r *http.Request) {
	teacherID := r.PathValue("id")

	var req assignSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SubjectID == "" || req.GroupID == "" {
		writeError(w, http.StatusBadRequest, "subject_id and group_id are required")
		return
	}

	ts := models.TeacherSubject{TeacherID: teacherID, SubjectID: req.SubjectID, GroupID: req.GroupID}
	if err := h.teachers.AssignSubject(r.Context(), ts); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleTeacherAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, services.ErrUsernameTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrUnknownRole):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// roleAllowsTeacherContacts определяет, можно ли отдавать email/телефон
// преподавателя текущему пользователю (раздел 35: "ограниченный набор
// полей для student").
func roleAllowsTeacherContacts(r *http.Request) bool {
	role, ok := middleware.RoleFromContext(r.Context())
	if !ok {
		return false
	}
	return role != models.RoleStudent
}
