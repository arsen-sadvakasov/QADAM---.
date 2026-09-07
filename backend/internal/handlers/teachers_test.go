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

type fakeTeacherRepository struct {
	byID map[string]*models.Teacher
}

func newFakeTeacherRepository() *fakeTeacherRepository {
	return &fakeTeacherRepository{byID: make(map[string]*models.Teacher)}
}

func (f *fakeTeacherRepository) FindByID(_ context.Context, id string) (*models.Teacher, error) {
	t, ok := f.byID[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return t, nil
}

func (f *fakeTeacherRepository) List(_ context.Context) ([]*models.Teacher, error) {
	var result []*models.Teacher
	for _, t := range f.byID {
		result = append(result, t)
	}
	return result, nil
}

func (f *fakeTeacherRepository) Create(_ context.Context, userID string, collegeID *string) (string, error) {
	id := "generated-teacher"
	f.byID[id] = &models.Teacher{ID: id, UserID: userID, CollegeID: collegeID}
	return id, nil
}

func (f *fakeTeacherRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

func (f *fakeTeacherRepository) AssignSubject(_ context.Context, _ models.TeacherSubject) error {
	return nil
}
func (f *fakeTeacherRepository) UnassignSubject(_ context.Context, _ models.TeacherSubject) error {
	return nil
}
func (f *fakeTeacherRepository) ListSubjectsByTeacher(_ context.Context, _ string) ([]models.TeacherSubject, error) {
	return nil, nil
}

func strPtr(s string) *string { return &s }

func withRole(r *http.Request, role models.RoleKey) *http.Request {
	return r.WithContext(middleware.ContextWithUser(r.Context(), "test-user", role))
}

func newTestTeachersHandler() *TeachersHandler {
	userRepo := newFakeAdminUserRepository()
	userAdmin := services.NewUserAdminService(userRepo, fakeRoleRepository{})
	teacherRepo := newFakeTeacherRepository()
	teacherRepo.byID["t1"] = &models.Teacher{
		ID:       "t1",
		FullName: "Ivanov I.I.",
		Email:    strPtr("ivanov@example.com"),
		Phone:    strPtr("+77001234567"),
	}
	return NewTeachersHandler(services.NewTeacherAdminService(userAdmin, teacherRepo))
}

func TestTeachersGet_ContactsHiddenForStudent(t *testing.T) {
	h := newTestTeachersHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teachers/t1", nil)
	req.SetPathValue("id", "t1")
	req = withRole(req, models.RoleStudent)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var dto teacherDTO
	if err := json.NewDecoder(rec.Body).Decode(&dto); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if dto.Email != nil || dto.Phone != nil {
		t.Errorf("expected contacts hidden for student, got email=%v phone=%v", dto.Email, dto.Phone)
	}
}

func TestTeachersGet_ContactsVisibleForAdmin(t *testing.T) {
	h := newTestTeachersHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teachers/t1", nil)
	req.SetPathValue("id", "t1")
	req = withRole(req, models.RoleAdmin)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	var dto teacherDTO
	if err := json.NewDecoder(rec.Body).Decode(&dto); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if dto.Email == nil {
		t.Error("expected contacts visible for admin")
	}
}

func TestTeachersCreate_Success(t *testing.T) {
	h := newTestTeachersHandler()
	body, _ := json.Marshal(map[string]string{
		"username":  "newteacher",
		"password":  "secret123",
		"full_name": "New Teacher",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teachers", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}
