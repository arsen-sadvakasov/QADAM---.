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

type fakeSubjectRepository struct {
	byID map[string]*models.Subject
}

func newFakeSubjectRepository() *fakeSubjectRepository {
	return &fakeSubjectRepository{byID: make(map[string]*models.Subject)}
}

func (f *fakeSubjectRepository) FindByID(_ context.Context, id string) (*models.Subject, error) {
	s, ok := f.byID[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return s, nil
}

func (f *fakeSubjectRepository) List(_ context.Context) ([]*models.Subject, error) {
	var result []*models.Subject
	for _, s := range f.byID {
		result = append(result, s)
	}
	return result, nil
}

func (f *fakeSubjectRepository) Create(_ context.Context, s *models.Subject) (string, error) {
	s.ID = "generated-subject"
	f.byID[s.ID] = s
	return s.ID, nil
}

func (f *fakeSubjectRepository) Update(_ context.Context, s *models.Subject) error {
	f.byID[s.ID] = s
	return nil
}

func (f *fakeSubjectRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

func TestSubjectsCreate_Success(t *testing.T) {
	h := NewSubjectsHandler(newFakeSubjectRepository())
	body, _ := json.Marshal(map[string]string{"name": "Databases"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subjects", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSubjectsCreate_MissingName(t *testing.T) {
	h := NewSubjectsHandler(newFakeSubjectRepository())
	body, _ := json.Marshal(map[string]string{"code": "DB101"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/subjects", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSubjectsUpdate_NotFound(t *testing.T) {
	h := NewSubjectsHandler(newFakeSubjectRepository())
	body, _ := json.Marshal(map[string]string{"name": "Updated"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/subjects/missing", bytes.NewReader(body))
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
