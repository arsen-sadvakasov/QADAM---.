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

// fakeStudentRepository — in-memory реализация
// repositories.StudentRepository для unit-тестов хендлера.
type fakeStudentRepository struct {
	byID   map[string]*models.Student
	nextID int
}

func (f *fakeStudentRepository) FindByID(_ context.Context, id string) (*models.Student, error) {
	if s, ok := f.byID[id]; ok {
		return s, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeStudentRepository) ListByGroup(_ context.Context, groupID string) ([]*models.Student, error) {
	var result []*models.Student
	for _, s := range f.byID {
		if s.GroupID == groupID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (f *fakeStudentRepository) List(_ context.Context, groupID, status string) ([]*models.Student, error) {
	var result []*models.Student
	for _, s := range f.byID {
		if groupID != "" && s.GroupID != groupID {
			continue
		}
		if status != "" && string(s.Status) != status {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func (f *fakeStudentRepository) Create(_ context.Context, s *models.Student) (string, error) {
	f.nextID++
	s.ID = string(rune('0' + f.nextID))
	clone := *s
	f.byID[s.ID] = &clone
	return s.ID, nil
}

func (f *fakeStudentRepository) Update(_ context.Context, s *models.Student) error {
	if _, ok := f.byID[s.ID]; !ok {
		return repositories.ErrNotFound
	}
	clone := *s
	f.byID[s.ID] = &clone
	return nil
}

func (f *fakeStudentRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

// fakeCuratorGroupRepository — in-memory реализация GroupRepository.
type fakeCuratorGroupRepository struct {
	byID      map[string]*models.Group
	byCurator map[string][]string
}

func (f *fakeCuratorGroupRepository) FindByID(_ context.Context, id string) (*models.Group, error) {
	if g, ok := f.byID[id]; ok {
		return g, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeCuratorGroupRepository) List(_ context.Context) ([]*models.Group, error) {
	var result []*models.Group
	for _, g := range f.byID {
		result = append(result, g)
	}
	return result, nil
}

func (f *fakeCuratorGroupRepository) ListByCurator(_ context.Context, curatorID string) ([]*models.Group, error) {
	var result []*models.Group
	for _, id := range f.byCurator[curatorID] {
		if g, ok := f.byID[id]; ok {
			result = append(result, g)
		}
	}
	return result, nil
}

func (f *fakeCuratorGroupRepository) Create(_ context.Context, g *models.Group) (string, error) {
	f.byID[g.ID] = g
	if g.CuratorID != nil {
		f.byCurator[*g.CuratorID] = append(f.byCurator[*g.CuratorID], g.ID)
	}
	return g.ID, nil
}

func (f *fakeCuratorGroupRepository) Update(_ context.Context, g *models.Group) error {
	f.byID[g.ID] = g
	return nil
}

func (f *fakeCuratorGroupRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

func newStudentsTestHandler() (*StudentsHandler, *fakeStudentRepository) {
	students := &fakeStudentRepository{byID: make(map[string]*models.Student)}
	groups := &fakeCuratorGroupRepository{
		byID:      make(map[string]*models.Group),
		byCurator: make(map[string][]string),
	}
	curatorID := "curator-1"
	groups.byID["group-1"] = &models.Group{ID: "group-1", CuratorID: &curatorID}
	groups.byCurator["curator-1"] = []string{"group-1"}
	groups.byID["group-2"] = &models.Group{ID: "group-2"} // без куратора

	svc := services.NewCuratorService(students, groups)
	return NewStudentsHandler(svc), students
}

func TestStudentsCreate_Success(t *testing.T) {
	h, repo := newStudentsTestHandler()

	body, _ := json.Marshal(map[string]any{"group_id": "group-1", "status": "active"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/students", bytes.NewReader(body))
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "curator-1", models.RoleCurator))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var dto studentDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if dto.GroupID != "group-1" || dto.Status != "active" {
		t.Errorf("unexpected DTO: %+v", dto)
	}
	if len(repo.byID) != 1 {
		t.Errorf("expected 1 student stored, got %d", len(repo.byID))
	}
}

func TestStudentsCreate_ForbiddenForeignGroup(t *testing.T) {
	h, _ := newStudentsTestHandler()

	body, _ := json.Marshal(map[string]any{"group_id": "group-2"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/students", bytes.NewReader(body))
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "curator-1", models.RoleCurator))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for foreign group, got %d", rec.Code)
	}
}

func TestStudentsList_CuratorOwnGroupOnly(t *testing.T) {
	h, repo := newStudentsTestHandler()

	repo.byID["s-1"] = &models.Student{ID: "s-1", GroupID: "group-1", Status: models.StudentStatusActive}
	repo.byID["s-2"] = &models.Student{ID: "s-2", GroupID: "group-2", Status: models.StudentStatusActive}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/students", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "curator-1", models.RoleCurator))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Students []studentDTO `json:"students"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Students) != 1 || resp.Students[0].GroupID != "group-1" {
		t.Errorf("expected only own group students, got %+v", resp.Students)
	}
}

func TestStudentsList_AdminSeesAll(t *testing.T) {
	h, repo := newStudentsTestHandler()

	repo.byID["s-1"] = &models.Student{ID: "s-1", GroupID: "group-1", Status: models.StudentStatusActive}
	repo.byID["s-2"] = &models.Student{ID: "s-2", GroupID: "group-2", Status: models.StudentStatusExpelled}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/students", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp struct {
		Students []studentDTO `json:"students"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Students) != 2 {
		t.Errorf("expected all students for admin, got %d", len(resp.Students))
	}
}

func TestStudentsUpdate_ChangeStatus(t *testing.T) {
	h, repo := newStudentsTestHandler()

	repo.byID["s-1"] = &models.Student{ID: "s-1", GroupID: "group-1", Status: models.StudentStatusActive}

	body, _ := json.Marshal(map[string]any{"group_id": "group-1", "status": "expelled"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/students/s-1", bytes.NewReader(body))
	req.SetPathValue("id", "s-1")
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "curator-1", models.RoleCurator))
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.byID["s-1"].Status != models.StudentStatusExpelled {
		t.Errorf("expected status expelled, got %q", repo.byID["s-1"].Status)
	}
}

func TestStudentsGet_NotFound(t *testing.T) {
	h, _ := newStudentsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/students/missing", nil)
	req.SetPathValue("id", "missing")
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestStudentsDelete_Success(t *testing.T) {
	h, repo := newStudentsTestHandler()

	repo.byID["s-1"] = &models.Student{ID: "s-1", GroupID: "group-1", Status: models.StudentStatusActive}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/students/s-1", nil)
	req.SetPathValue("id", "s-1")
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "curator-1", models.RoleCurator))
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if _, ok := repo.byID["s-1"]; ok {
		t.Error("expected student to be deleted")
	}
}
