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
)

type fakeGroupRepository struct {
	byID map[string]*models.Group
}

func newFakeGroupRepository() *fakeGroupRepository {
	return &fakeGroupRepository{byID: make(map[string]*models.Group)}
}

func (f *fakeGroupRepository) FindByID(_ context.Context, id string) (*models.Group, error) {
	g, ok := f.byID[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return g, nil
}

func (f *fakeGroupRepository) List(_ context.Context) ([]*models.Group, error) {
	var result []*models.Group
	for _, g := range f.byID {
		result = append(result, g)
	}
	return result, nil
}

func (f *fakeGroupRepository) ListByCurator(_ context.Context, curatorID string) ([]*models.Group, error) {
	var result []*models.Group
	for _, g := range f.byID {
		if g.CuratorID != nil && *g.CuratorID == curatorID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (f *fakeGroupRepository) Create(_ context.Context, g *models.Group) (string, error) {
	g.ID = "generated-group"
	f.byID[g.ID] = g
	return g.ID, nil
}

func (f *fakeGroupRepository) Update(_ context.Context, g *models.Group) error {
	f.byID[g.ID] = g
	return nil
}

func (f *fakeGroupRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

func TestGroupsCreate_Success(t *testing.T) {
	h := NewGroupsHandler(newFakeGroupRepository())
	body, _ := json.Marshal(map[string]string{
		"specialty_id": "spec-1",
		"course_id":    "course-1",
		"name":         "PO-23",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGroupsCreate_MissingFields(t *testing.T) {
	h := NewGroupsHandler(newFakeGroupRepository())
	body, _ := json.Marshal(map[string]string{"name": "PO-23"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGroupsGet_NotFound(t *testing.T) {
	h := NewGroupsHandler(newFakeGroupRepository())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/missing", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestGroupsDelete_Success(t *testing.T) {
	repo := newFakeGroupRepository()
	repo.byID["g1"] = &models.Group{ID: "g1", Name: "PO-23"}
	h := NewGroupsHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/groups/g1", nil)
	req.SetPathValue("id", "g1")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
