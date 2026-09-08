package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// CuratorsHandler содержит HTTP-хендлеры для /api/v1/curators/*
// (раздел 35 API Plan). Admin only. Куратор в спецификации — это
// пользователь с ролью `curator` (раздел 34.1: отдельной таблицы curators
// нет, привязка к группе идёт через groups.curator_id).
type CuratorsHandler struct {
	userAdmin *services.UserAdminService
}

// NewCuratorsHandler создаёт CuratorsHandler с внедрённым UserAdminService.
func NewCuratorsHandler(userAdmin *services.UserAdminService) *CuratorsHandler {
	return &CuratorsHandler{userAdmin: userAdmin}
}

// List обрабатывает GET /api/v1/curators. Admin only.
func (h *CuratorsHandler) List(w http.ResponseWriter, r *http.Request) {
	curatorRole := models.RoleCurator
	users, err := h.userAdmin.List(r.Context(), repositories.UserListFilter{RoleKey: &curatorRole})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	dtos := make([]adminUserDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, adminUserDTOFromModel(u))
	}
	writeJSON(w, http.StatusOK, map[string]any{"curators": dtos})
}

type createCuratorRequest struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	FullName string  `json:"full_name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
}

// Create обрабатывает POST /api/v1/curators. Admin only.
func (h *CuratorsHandler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createCuratorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" || req.FullName == "" {
		writeError(w, http.StatusBadRequest, "username, password and full_name are required")
		return
	}

	user, err := h.userAdmin.Create(r.Context(), actorID, services.CreateUserInput{
		Username: req.Username,
		Password: req.Password,
		FullName: req.FullName,
		Role:     models.RoleCurator,
		Email:    req.Email,
		Phone:    req.Phone,
	})
	if err != nil {
		handleUserAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, adminUserDTOFromModel(user))
}

type updateCuratorRequest struct {
	FullName *string `json:"full_name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	IsActive *bool   `json:"is_active"`
}

// Update обрабатывает PATCH /api/v1/curators/{id}. Admin only.
func (h *CuratorsHandler) Update(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := r.PathValue("id")

	var req updateCuratorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.userAdmin.Update(r.Context(), actorID, id, services.UpdateUserInput{
		FullName: req.FullName,
		Email:    req.Email,
		Phone:    req.Phone,
		IsActive: req.IsActive,
	})
	if err != nil {
		handleUserAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, adminUserDTOFromModel(user))
}
