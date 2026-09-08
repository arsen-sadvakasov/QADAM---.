package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/qadam/backend/internal/locales"
	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// LocalesHandler содержит HTTP-хендлеры локализации (Phase 12, раздел 26
// спецификации): выдача словаря переводов и смена языка пользователем.
type LocalesHandler struct {
	userAdmin *services.UserAdminService
	users     repositories.UserRepository
}

// NewLocalesHandler создаёт LocalesHandler с внедрёнными зависимостями.
func NewLocalesHandler(userAdmin *services.UserAdminService, users repositories.UserRepository) *LocalesHandler {
	return &LocalesHandler{userAdmin: userAdmin, users: users}
}

// Get обрабатывает GET /api/v1/locales/{lang} — словарь переводов для языка.
// Неизвестный язык нормализуется к fallback (русский).
func (h *LocalesHandler) Get(w http.ResponseWriter, r *http.Request) {
	lang := r.PathValue("lang")

	dict := locales.Get(lang)
	writeJSON(w, http.StatusOK, map[string]any{
		"language": string(locales.Normalize(lang)),
		"translations": dict,
	})
}

type changeLanguageRequest struct {
	Language string `json:"language"`
}

// ChangeMyLanguage обрабатывает PATCH /api/v1/users/me/language — смена
// языка интерфейса текущим пользователем (раздел 26: язык хранится в
// профиле users.language, fallback на русский).
func (h *LocalesHandler) ChangeMyLanguage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changeLanguageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if _, err := locales.Parse(req.Language); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.userAdmin.Update(r.Context(), userID, userID, services.UpdateUserInput{
		Language: &req.Language,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrUnknownLanguage):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, repositories.ErrNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"language": user.Language})
}

// MyLanguage обрабатывает GET /api/v1/users/me/language — текущий язык
// пользователя (для мгновенного применения интерфейса без полного профиля).
func (h *LocalesHandler) MyLanguage(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, map[string]string{"language": user.Language})
}
