package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// GroupsHandler содержит HTTP-хендлеры для /api/v1/groups/*
// (раздел 35 API Plan). Чтение — все авторизованные, запись — Admin.
type GroupsHandler struct {
	groups repositories.GroupRepository
}

// NewGroupsHandler создаёт GroupsHandler с внедрённым GroupRepository.
func NewGroupsHandler(groups repositories.GroupRepository) *GroupsHandler {
	return &GroupsHandler{groups: groups}
}

type groupDTO struct {
	ID          string  `json:"id"`
	SpecialtyID string  `json:"specialty_id"`
	CourseID    string  `json:"course_id"`
	CuratorID   *string `json:"curator_id"`
	Name        string  `json:"name"`
}

func groupDTOFromModel(g *models.Group) groupDTO {
	return groupDTO{
		ID:          g.ID,
		SpecialtyID: g.SpecialtyID,
		CourseID:    g.CourseID,
		CuratorID:   g.CuratorID,
		Name:        g.Name,
	}
}

// List обрабатывает GET /api/v1/groups. Все авторизованные.
func (h *GroupsHandler) List(w http.ResponseWriter, r *http.Request) {
	groups, err := h.groups.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	dtos := make([]groupDTO, 0, len(groups))
	for _, g := range groups {
		dtos = append(dtos, groupDTOFromModel(g))
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": dtos})
}

// Get обрабатывает GET /api/v1/groups/{id}.
func (h *GroupsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	g, err := h.groups.FindByID(r.Context(), id)
	if err != nil {
		handleNotFoundOr500(w, err, "group not found")
		return
	}
	writeJSON(w, http.StatusOK, groupDTOFromModel(g))
}

type groupWriteRequest struct {
	SpecialtyID string  `json:"specialty_id"`
	CourseID    string  `json:"course_id"`
	CuratorID   *string `json:"curator_id"`
	Name        string  `json:"name"`
}

// Create обрабатывает POST /api/v1/groups. Admin only.
func (h *GroupsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req groupWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SpecialtyID == "" || req.CourseID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "specialty_id, course_id and name are required")
		return
	}

	g := &models.Group{
		SpecialtyID: req.SpecialtyID,
		CourseID:    req.CourseID,
		CuratorID:   req.CuratorID,
		Name:        req.Name,
	}
	id, err := h.groups.Create(r.Context(), g)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	g.ID = id
	writeJSON(w, http.StatusCreated, groupDTOFromModel(g))
}

// Update обрабатывает PATCH /api/v1/groups/{id}. Admin only.
func (h *GroupsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	existing, err := h.groups.FindByID(r.Context(), id)
	if err != nil {
		handleNotFoundOr500(w, err, "group not found")
		return
	}

	var req groupWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SpecialtyID != "" {
		existing.SpecialtyID = req.SpecialtyID
	}
	if req.CourseID != "" {
		existing.CourseID = req.CourseID
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.CuratorID = req.CuratorID

	if err := h.groups.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, groupDTOFromModel(existing))
}

// Delete обрабатывает DELETE /api/v1/groups/{id}. Admin only.
func (h *GroupsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.groups.SoftDelete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleNotFoundOr500(w http.ResponseWriter, err error, notFoundMessage string) {
	if errors.Is(err, repositories.ErrNotFound) {
		writeError(w, http.StatusNotFound, notFoundMessage)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}
