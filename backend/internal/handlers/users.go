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

// UsersHandler содержит HTTP-хендлеры для /api/v1/users/*
// (раздел 35 API Plan). GET /me — для всех авторизованных, остальные
// CRUD-операции — Admin Panel (Phase 5 спецификации).
type UsersHandler struct {
	users     repositories.UserRepository
	userAdmin *services.UserAdminService
}

// NewUsersHandler создаёт UsersHandler с внедрёнными зависимостями.
func NewUsersHandler(users repositories.UserRepository, userAdmin *services.UserAdminService) *UsersHandler {
	return &UsersHandler{users: users, userAdmin: userAdmin}
}

type adminUserDTO struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	FullName string  `json:"full_name"`
	Role     string  `json:"role"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Language string  `json:"language"`
	Theme    string  `json:"theme"`
	IsActive bool    `json:"is_active"`
}

func adminUserDTOFromModel(u *models.User) adminUserDTO {
	return adminUserDTO{
		ID:       u.ID,
		Username: u.Username,
		FullName: u.FullName,
		Role:     string(u.RoleKey),
		Email:    u.Email,
		Phone:    u.Phone,
		Language: u.Language,
		Theme:    u.ThemePreference,
		IsActive: u.IsActive,
	}
}

// Me обрабатывает GET /api/v1/users/me — возвращает профиль текущего
// авторизованного пользователя. Требует прохождения middleware.Auth.
func (h *UsersHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.users.FindByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	dto := userDTO{
		ID:       user.ID,
		Username: user.Username,
		FullName: user.FullName,
		Role:     string(user.RoleKey),
		Language: user.Language,
		Theme:    user.ThemePreference,
	}
	writeJSON(w, http.StatusOK, dto)
}

// List обрабатывает GET /api/v1/users?role=&search= — список пользователей
// с фильтрами. Admin only (раздел 35 API Plan).
func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) {
	var filter repositories.UserListFilter
	if roleParam := r.URL.Query().Get("role"); roleParam != "" {
		role := models.RoleKey(roleParam)
		filter.RoleKey = &role
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
	}

	users, err := h.userAdmin.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	dtos := make([]adminUserDTO, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, adminUserDTOFromModel(u))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": dtos})
}

// Get обрабатывает GET /api/v1/users/{id}. Admin only.
func (h *UsersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	user, err := h.userAdmin.Get(r.Context(), id)
	if err != nil {
		handleUserAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, adminUserDTOFromModel(user))
}

type createUserRequest struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	FullName string  `json:"full_name"`
	Role     string  `json:"role"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	Language string  `json:"language"`
	Theme    string  `json:"theme"`
}

// Create обрабатывает POST /api/v1/users — создание пользователя. Admin only.
func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" || req.FullName == "" || req.Role == "" {
		writeError(w, http.StatusBadRequest, "username, password, full_name and role are required")
		return
	}

	user, err := h.userAdmin.Create(r.Context(), actorID, services.CreateUserInput{
		Username: req.Username,
		Password: req.Password,
		FullName: req.FullName,
		Role:     models.RoleKey(req.Role),
		Email:    req.Email,
		Phone:    req.Phone,
		Language: req.Language,
		Theme:    req.Theme,
	})
	if err != nil {
		handleUserAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, adminUserDTOFromModel(user))
}

type updateUserRequest struct {
	FullName *string `json:"full_name"`
	Role     *string `json:"role"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	IsActive *bool   `json:"is_active"`
	Language *string `json:"language"`
}

// Update обрабатывает PATCH /api/v1/users/{id} — редактирование. Admin only.
func (h *UsersHandler) Update(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := r.PathValue("id")

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	in := services.UpdateUserInput{
		FullName: req.FullName,
		Email:    req.Email,
		Phone:    req.Phone,
		IsActive: req.IsActive,
		Language: req.Language,
	}
	if req.Role != nil {
		role := models.RoleKey(*req.Role)
		in.Role = &role
	}

	user, err := h.userAdmin.Update(r.Context(), actorID, id, in)
	if err != nil {
		handleUserAdminError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, adminUserDTOFromModel(user))
}

// Block обрабатывает DELETE /api/v1/users/{id} — блокировка/soft-delete. Admin only.
func (h *UsersHandler) Block(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := r.PathValue("id")
	if err := h.userAdmin.Block(r.Context(), actorID, id); err != nil {
		handleUserAdminError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleUserAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, services.ErrUsernameTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrUnknownRole):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrUnknownLanguage):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrUsernameInvalid),
		errors.Is(err, services.ErrPasswordTooShort):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
