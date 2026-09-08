package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// TestLocalesGet_Success проверяет выдачу словаря.
func TestLocalesGet_Success(t *testing.T) {
	h := NewLocalesHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locales/ru", nil)
	req.SetPathValue("lang", "ru")
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "user-1", models.RoleStudent))
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Language     string            `json:"language"`
		Translations map[string]string `json:"translations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Language != "ru" {
		t.Errorf("expected language ru, got %q", resp.Language)
	}
	if resp.Translations["schedule.today"] != "Сегодня" {
		t.Errorf("expected schedule.today=Сегодня, got %q", resp.Translations["schedule.today"])
	}
}

// TestLocalesGet_UnknownLangFallsBack проверяет fallback на русский.
func TestLocalesGet_UnknownLangFallsBack(t *testing.T) {
	h := NewLocalesHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/locales/de", nil)
	req.SetPathValue("lang", "de")
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "user-1", models.RoleStudent))
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (fallback, not error), got %d", rec.Code)
	}
	var resp struct {
		Language string `json:"language"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Language != "ru" {
		t.Errorf("expected fallback to ru, got %q", resp.Language)
	}
}

// fakeAdminUserRepository уже объявлен в users_test.go — переиспользуем.
func newLocalesTestHandler() (*LocalesHandler, *fakeAdminUserRepository, *fakeRoleRepository) {
	users := newFakeAdminUserRepository()
	roles := &fakeRoleRepository{}
	svc := services.NewUserAdminService(users, roles)
	return NewLocalesHandler(svc, users), users, roles
}

// TestChangeMyLanguage_Success проверяет смену языка пользователем.
func TestChangeMyLanguage_Success(t *testing.T) {
	h, users, roles := newLocalesTestHandler()

	// создаём пользователя (admin-сервис резолвит роль по ключу)
	_, err := services.NewUserAdminService(users, roles).Create(context.Background(), "", services.CreateUserInput{
		Username: "student-1",
		Password: "password123",
		FullName: "Student One",
		Role:     models.RoleStudent,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	// находим его ID
	created, ok := users.byUsername["student-1"]
	if !ok {
		t.Fatal("created user not found in fake repo")
	}

	body, _ := json.Marshal(map[string]string{"language": "kz"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me/language", bytes.NewReader(body))
	req = req.WithContext(middleware.ContextWithUser(req.Context(), created.ID, models.RoleStudent))
	rec := httptest.NewRecorder()

	h.ChangeMyLanguage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Language string `json:"language"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Language != "kz" {
		t.Errorf("expected language kz, got %q", resp.Language)
	}
}

// TestChangeMyLanguage_InvalidLanguage проверяет отклонение неизвестного языка.
func TestChangeMyLanguage_InvalidLanguage(t *testing.T) {
	h, users, roles := newLocalesTestHandler()

	_, _ = services.NewUserAdminService(users, roles).Create(context.Background(), "", services.CreateUserInput{
		Username: "student-2",
		Password: "password123",
		FullName: "Student Two",
		Role:     models.RoleStudent,
	})
	created := users.byUsername["student-2"]

	body, _ := json.Marshal(map[string]string{"language": "de"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me/language", bytes.NewReader(body))
	req = req.WithContext(middleware.ContextWithUser(req.Context(), created.ID, models.RoleStudent))
	rec := httptest.NewRecorder()

	h.ChangeMyLanguage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported language, got %d", rec.Code)
	}
}

// TestMyLanguage_Success проверяет получение текущего языка.
func TestMyLanguage_Success(t *testing.T) {
	h, users, roles := newLocalesTestHandler()

	_, _ = services.NewUserAdminService(users, roles).Create(context.Background(), "", services.CreateUserInput{
		Username: "student-3",
		Password: "password123",
		FullName: "Student Three",
		Role:     models.RoleStudent,
		Language: "en",
	})
	created := users.byUsername["student-3"]

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/language", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), created.ID, models.RoleStudent))
	rec := httptest.NewRecorder()

	h.MyLanguage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Language string `json:"language"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Language != "en" {
		t.Errorf("expected language en, got %q", resp.Language)
	}
}

// Проверка совместимости с users_test.go: fakeAdminUserRepository должен
// реализовывать UserRepository; интерфейсные проверки в других файлах.
var _ repositories.UserRepository = (*fakeAdminUserRepository)(nil)
