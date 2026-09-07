package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/qadam/backend/internal/services"
)

// refreshCookieName — имя httpOnly cookie, хранящей refresh-токен
// (раздел 15 спецификации: "refresh token — httpOnly secure cookie").
const refreshCookieName = "qadam_refresh_token"

// AuthHandler содержит HTTP-хендлеры для /api/v1/auth/*.
type AuthHandler struct {
	auth *services.AuthService
}

// NewAuthHandler создаёт AuthHandler с внедрённым AuthService.
func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        userDTO   `json:"user"`
}

type userDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Language string `json:"language"`
	Theme    string `json:"theme"`
}

// Login обрабатывает POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	result, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid username or password")
		case errors.Is(err, services.ErrUserInactive):
			writeError(w, http.StatusForbidden, "account is blocked, contact your administrator")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	setRefreshCookie(w, result.RefreshToken)
	writeLoginResponse(w, result)
}

// Refresh обрабатывает POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}

	result, err := h.auth.Refresh(r.Context(), cookie.Value)
	if err != nil {
		clearRefreshCookie(w)
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	setRefreshCookie(w, result.RefreshToken)
	writeLoginResponse(w, result)
}

// Logout обрабатывает POST /api/v1/auth/logout (требует авторизации).
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(refreshCookieName); err == nil && cookie.Value != "" {
		_ = h.auth.Logout(r.Context(), cookie.Value)
	}
	clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func writeLoginResponse(w http.ResponseWriter, result *services.AuthResult) {
	resp := loginResponse{
		AccessToken: result.AccessToken,
		ExpiresAt:   time.Now().Add(services.AccessTokenTTL),
		User: userDTO{
			ID:       result.User.ID,
			Username: result.User.Username,
			FullName: result.User.FullName,
			Role:     string(result.User.RoleKey),
			Language: result.User.Language,
			Theme:    result.User.ThemePreference,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(services.RefreshTokenTTL.Seconds()),
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
