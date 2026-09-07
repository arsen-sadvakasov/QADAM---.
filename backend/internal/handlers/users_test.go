package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

type fakeAdminUserRepository struct {
	byID       map[string]*models.User
	byUsername map[string]*models.User
}

func newFakeAdminUserRepository() *fakeAdminUserRepository {
	return &fakeAdminUserRepository{
		byID:       make(map[string]*models.User),
		byUsername: make(map[string]*models.User),
	}
}

func (f *fakeAdminUserRepository) FindByUsername(_ context.Context, username string) (*models.User, error) {
	u, ok := f.byUsername[username]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return u, nil
}

func (f *fakeAdminUserRepository) FindByID(_ context.Context, id string) (*models.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return u, nil
}

func (f *fakeAdminUserRepository) UpdateLastLogin(_ context.Context, _ string) error { return nil }

func (f *fakeAdminUserRepository) List(_ context.Context, filter repositories.UserListFilter) ([]*models.User, error) {
	var result []*models.User
	for _, u := range f.byID {
		if filter.RoleKey != nil && u.RoleKey != *filter.RoleKey {
			continue
		}
		result = append(result, u)
	}
	return result, nil
}

func (f *fakeAdminUserRepository) Create(_ context.Context, u *models.User) (string, error) {
	u.ID = "generated-" + u.Username
	f.byID[u.ID] = u
	f.byUsername[u.Username] = u
	return u.ID, nil
}

func (f *fakeAdminUserRepository) Update(_ context.Context, u *models.User) error {
	f.byID[u.ID] = u
	return nil
}

func (f *fakeAdminUserRepository) SoftDelete(_ context.Context, id string) error {
	if u, ok := f.byID[id]; ok {
		u.IsActive = false
	}
	return nil
}

func (f *fakeAdminUserRepository) ExistsByUsername(_ context.Context, username string) (bool, error) {
	_, ok := f.byUsername[username]
	return ok, nil
}

type fakeRoleRepository struct{}

func (fakeRoleRepository) FindByKey(_ context.Context, key models.RoleKey) (*models.Role, error) {
	return &models.Role{ID: "role-" + string(key), Key: key, Name: string(key)}, nil
}

func (fakeRoleRepository) List(_ context.Context) ([]*models.Role, error) {
	return []*models.Role{{ID: "role-admin", Key: models.RoleAdmin, Name: "admin"}}, nil
}

func newTestUsersHandler() (*UsersHandler, *fakeAdminUserRepository) {
	userRepo := newFakeAdminUserRepository()
	adminService := services.NewUserAdminService(userRepo, fakeRoleRepository{})
	return NewUsersHandler(userRepo, adminService), userRepo
}

func TestUsersCreate_Success(t *testing.T) {
	h, _ := newTestUsersHandler()
	body, _ := json.Marshal(map[string]string{
		"username":  "newstudent",
		"password":  "secret123",
		"full_name": "New Student",
		"role":      "student",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var dto adminUserDTO
	if err := json.NewDecoder(rec.Body).Decode(&dto); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if dto.Username != "newstudent" {
		t.Errorf("expected username newstudent, got %q", dto.Username)
	}
}

func TestUsersCreate_MissingFields(t *testing.T) {
	h, _ := newTestUsersHandler()
	body, _ := json.Marshal(map[string]string{"username": "onlyusername"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUsersCreate_DuplicateUsername(t *testing.T) {
	h, repo := newTestUsersHandler()
	repo.byUsername["taken"] = &models.User{ID: "existing", Username: "taken"}

	body, _ := json.Marshal(map[string]string{
		"username":  "taken",
		"password":  "secret123",
		"full_name": "Someone",
		"role":      "student",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUsersUpdate_NotFound(t *testing.T) {
	h, _ := newTestUsersHandler()
	body, _ := json.Marshal(map[string]string{"full_name": "Updated"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/does-not-exist", bytes.NewReader(body))
	req.SetPathValue("id", "does-not-exist")
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestUsersBlock_Success(t *testing.T) {
	h, repo := newTestUsersHandler()
	repo.byID["user-1"] = &models.User{ID: "user-1", Username: "u1", IsActive: true}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/user-1", nil)
	req.SetPathValue("id", "user-1")
	rec := httptest.NewRecorder()

	h.Block(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if repo.byID["user-1"].IsActive {
		t.Error("expected user to be inactive after Block")
	}
}

func TestUsersList_FiltersByRole(t *testing.T) {
	h, repo := newTestUsersHandler()
	repo.byID["u1"] = &models.User{ID: "u1", Username: "u1", RoleKey: models.RoleStudent}
	repo.byID["u2"] = &models.User{ID: "u2", Username: "u2", RoleKey: models.RoleTeacher}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?role=teacher", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Users []adminUserDTO `json:"users"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.Users) != 1 || body.Users[0].Username != "u2" {
		t.Fatalf("expected only teacher u2, got %+v", body.Users)
	}
}
