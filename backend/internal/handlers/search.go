package handlers

import (
	"errors"
	"net/http"

	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// SearchHandler содержит HTTP-хендлер глобального поиска
// (GET /api/v1/search?q=&type=, раздел 35 API Plan). Все авторизованные;
// набор типов результатов зависит от роли (раздел 25 спецификации).
type SearchHandler struct {
	search *services.SearchService
}

// NewSearchHandler создаёт SearchHandler с внедрённым сервисом.
func NewSearchHandler(search *services.SearchService) *SearchHandler {
	return &SearchHandler{search: search}
}

// Search обрабатывает GET /api/v1/search?q=&type=.
func (h *SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	_, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	results, err := h.search.Search(r.Context(), role,
		r.URL.Query().Get("q"), r.URL.Query().Get("type"))
	if err != nil {
		handleSearchError(w, err)
		return
	}

	if results == nil {
		results = []repositories.SearchResult{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func handleSearchError(w http.ResponseWriter, err error) {
	if errors.Is(err, services.ErrSearchValidation) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}
