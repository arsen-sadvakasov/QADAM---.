package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// SubjectsHandler содержит HTTP-хендлеры для /api/v1/subjects/*
// (раздел 35 API Plan). Чтение — все авторизованные, запись — Admin.
type SubjectsHandler struct {
	subjects repositories.SubjectRepository
}

// NewSubjectsHandler создаёт SubjectsHandler с внедрённым SubjectRepository.
func NewSubjectsHandler(subjects repositories.SubjectRepository) *SubjectsHandler {
	return &SubjectsHandler{subjects: subjects}
}

type subjectDTO struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Code *string `json:"code"`
}

func subjectDTOFromModel(s *models.Subject) subjectDTO {
	return subjectDTO{ID: s.ID, Name: s.Name, Code: s.Code}
}

// List обрабатывает GET /api/v1/subjects. Все авторизованные.
func (h *SubjectsHandler) List(w http.ResponseWriter, r *http.Request) {
	subjects, err := h.subjects.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	dtos := make([]subjectDTO, 0, len(subjects))
	for _, s := range subjects {
		dtos = append(dtos, subjectDTOFromModel(s))
	}
	writeJSON(w, http.StatusOK, map[string]any{"subjects": dtos})
}

// Get обрабатывает GET /api/v1/subjects/{id}.
func (h *SubjectsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s, err := h.subjects.FindByID(r.Context(), id)
	if err != nil {
		handleNotFoundOr500(w, err, "subject not found")
		return
	}
	writeJSON(w, http.StatusOK, subjectDTOFromModel(s))
}

type subjectWriteRequest struct {
	Name string  `json:"name"`
	Code *string `json:"code"`
}

// Create обрабатывает POST /api/v1/subjects. Admin only.
func (h *SubjectsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req subjectWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	s := &models.Subject{Name: req.Name, Code: req.Code}
	id, err := h.subjects.Create(r.Context(), s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	s.ID = id
	writeJSON(w, http.StatusCreated, subjectDTOFromModel(s))
}

// Update обрабатывает PATCH /api/v1/subjects/{id}. Admin only.
func (h *SubjectsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := h.subjects.FindByID(r.Context(), id)
	if err != nil {
		handleNotFoundOr500(w, err, "subject not found")
		return
	}

	var req subjectWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.Code = req.Code

	if err := h.subjects.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, subjectDTOFromModel(existing))
}
