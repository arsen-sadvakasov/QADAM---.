package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/services"
)

// fakeTokenService — генерирует и парсит реальные JWT через TokenService.
func newTestTokenService() *services.TokenService {
	return services.NewTokenService("test-secret-32-bytes-long-enough!")
}

func TestAuth_MissingHeader(t *testing.T) {
	tokens := newTestTokenService()
	handler := Auth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next must not be called without auth")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_MalformedHeader(t *testing.T) {
	tokens := newTestTokenService()
	handler := Auth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next must not be called on malformed header")
	}))

	cases := map[string]string{
		"no bearer prefix": "sometoken",
		"wrong scheme":     "Basic dXNlcjpwYXNz",
		"empty token":      "Bearer ",
		"extra parts":      "Bearer a b",
	}
	for name, header := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", header)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: expected 401, got %d", name, rec.Code)
		}
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	tokens := newTestTokenService()
	handler := Auth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next must not be called with invalid token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_ValidToken_SetsContext(t *testing.T) {
	tokens := newTestTokenService()
	handler := Auth(tokens)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok || userID != "user-1" {
			t.Errorf("expected userID user-1 in context, got %q (ok=%v)", userID, ok)
		}
		role, ok := RoleFromContext(r.Context())
		if !ok || role != models.RoleAdmin {
			t.Errorf("expected role admin in context, got %q (ok=%v)", role, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Генерируем валидный токен через TokenService (как AuthService.Login).
	token, err := tokens.GenerateAccessToken("user-1", models.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireRole_Allowed(t *testing.T) {
	handler := RequireRole(models.RoleAdmin, models.RoleCurator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(ContextWithUser(req.Context(), "user-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for allowed role, got %d", rec.Code)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	handler := RequireRole(models.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next must not be called for forbidden role")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(ContextWithUser(req.Context(), "user-1", models.RoleStudent))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequireRole_NoUserInContext(t *testing.T) {
	handler := RequireRole(models.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next must not be called without user")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without user context, got %d", rec.Code)
	}
}

func TestSecurityHeaders_Present(t *testing.T) {
	enableHSTS := true
	handler := SecurityHeaders(enableHSTS)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}
	for header, want := range checks {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("expected %s=%q, got %q", header, want, got)
		}
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("expected CSP header to be set")
	}
}

func TestSecurityHeaders_HSTSOmittedInDev(t *testing.T) {
	handler := SecurityHeaders(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must be omitted when disabled (dev/HTTP)")
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	handler := CORS("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("expected allowed origin echo, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected credentials header, got %q", got)
	}
}

func TestCORS_DisallowedOrigin_NoHeaders(t *testing.T) {
	handler := CORS("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS headers for disallowed origin")
	}
	if rec.Code != http.StatusOK {
		// Запрос проходит дальше без CORS-заголовков; браузер его заблокирует.
		t.Errorf("expected request to pass through, got %d", rec.Code)
	}
}

func TestCORS_PreflightShortCircuit(t *testing.T) {
	called := false
	handler := CORS("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if called {
		t.Error("preflight must not reach the next handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 on preflight, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods on preflight")
	}
}

func TestContextWithUser_TestHelper(t *testing.T) {
	ctx := context.Background()
	ctx = ContextWithUser(ctx, "u-9", models.RoleTeacher)

	userID, ok := UserIDFromContext(ctx)
	if !ok || userID != "u-9" {
		t.Errorf("expected u-9, got %q (ok=%v)", userID, ok)
	}
	role, ok := RoleFromContext(ctx)
	if !ok || role != models.RoleTeacher {
		t.Errorf("expected teacher role, got %q (ok=%v)", role, ok)
	}
}
