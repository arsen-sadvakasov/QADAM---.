package handlers

import (
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

// fakeSearchRepository — in-memory реализация
// repositories.SearchRepository для unit-тестов хендлера.
type fakeSearchRepository struct {
	students []repositories.SearchResult
	groups   []repositories.SearchResult
}

func (f *fakeSearchRepository) SearchStudents(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return f.students, nil
}

func (f *fakeSearchRepository) SearchTeachers(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return nil, nil
}

func (f *fakeSearchRepository) SearchCurators(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return nil, nil
}

func (f *fakeSearchRepository) SearchGroups(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return f.groups, nil
}

func (f *fakeSearchRepository) SearchSubjects(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return nil, nil
}

func (f *fakeSearchRepository) SearchRooms(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return nil, nil
}

func TestSearchHandler_Success(t *testing.T) {
	repo := &fakeSearchRepository{
		groups: []repositories.SearchResult{{Type: repositories.SearchTypeGroup, ID: "g-1", Title: "ПО-23"}},
	}
	h := NewSearchHandler(services.NewSearchService(repo))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=ПО", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Results []repositories.SearchResult `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Title != "ПО-23" {
		t.Errorf("unexpected results: %+v", resp.Results)
	}
}

func TestSearchHandler_QueryTooShort(t *testing.T) {
	h := NewSearchHandler(services.NewSearchService(&fakeSearchRepository{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=a", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for short query, got %d", rec.Code)
	}
}

func TestSearchHandler_EmptyResultsIsArray(t *testing.T) {
	h := NewSearchHandler(services.NewSearchService(&fakeSearchRepository{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=ничего", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// results должен быть [] (JSON-массив), а не null
	if !json.Valid(rec.Body.Bytes()) {
		t.Error("expected valid JSON")
	}
	var resp struct {
		Results []repositories.SearchResult `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Results == nil {
		t.Error("expected results to be [], not null")
	}
}
