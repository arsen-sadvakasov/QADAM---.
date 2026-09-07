package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/repositories"
)

// UsersHandler содержит HTTP-хендлеры для /api/v1/users/*.
type UsersHandler struct {
	users repositories.UserRepository
}

// NewUsersHandler создаёт UsersHandler с внедрённым UserRepository.
func NewUsersHandler(users repositories.UserRepository) *UsersHandler {
	return &UsersHandler{users: users}
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dto)
}
