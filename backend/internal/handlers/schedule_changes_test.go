package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// fakeScheduleChangeRepo — in-memory реализация
// repositories.ScheduleChangeRepository для unit-тестов хендлера.
type fakeScheduleChangeRepo struct {
	byID map[string]*models.ScheduleChange
}

func newFakeScheduleChangeRepo() *fakeScheduleChangeRepo {
	return &fakeScheduleChangeRepo{byID: make(map[string]*models.ScheduleChange)}
}

func (f *fakeScheduleChangeRepo) FindByID(_ context.Context, id string) (*models.ScheduleChange, error) {
	if c, ok := f.byID[id]; ok {
		return c, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeScheduleChangeRepo) FindByTemplateAndDate(_ context.Context, templateID string, date time.Time) (*models.ScheduleChange, error) {
	for _, c := range f.byID {
		if c.ScheduleTemplateID == templateID && sameDay(c.ChangeDate, date) {
			return c, nil
		}
	}
	return nil, repositories.ErrNotFound
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func (f *fakeScheduleChangeRepo) ListByDateRange(_ context.Context, from, to time.Time) ([]*models.ScheduleChangeWithNames, error) {
	var result []*models.ScheduleChangeWithNames
	for _, c := range f.byID {
		if !c.ChangeDate.Before(from) && !c.ChangeDate.After(to) {
			result = append(result, &models.ScheduleChangeWithNames{ScheduleChange: *c})
		}
	}
	return result, nil
}

func (f *fakeScheduleChangeRepo) ListByGroupAndDateRange(_ context.Context, _ string, from, to time.Time) ([]*models.ScheduleChangeWithNames, error) {
	return f.ListByDateRange(context.Background(), from, to)
}

func (f *fakeScheduleChangeRepo) ListByTeacherAndDateRange(_ context.Context, _ string, from, to time.Time) ([]*models.ScheduleChangeWithNames, error) {
	return f.ListByDateRange(context.Background(), from, to)
}

func (f *fakeScheduleChangeRepo) Create(_ context.Context, c *models.ScheduleChange) (string, error) {
	c.ID = "generated-change"
	clone := *c
	f.byID[c.ID] = &clone
	return c.ID, nil
}

func (f *fakeScheduleChangeRepo) Update(_ context.Context, c *models.ScheduleChange) error {
	if _, ok := f.byID[c.ID]; !ok {
		return repositories.ErrNotFound
	}
	clone := *c
	f.byID[c.ID] = &clone
	return nil
}

func (f *fakeScheduleChangeRepo) Delete(_ context.Context, id string) error {
	if _, ok := f.byID[id]; !ok {
		return repositories.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

func newScheduleChangesTestHandler() (*ScheduleChangesHandler, *fakeScheduleChangeRepo) {
	repo := newFakeScheduleChangeRepo()
	templates := &fakeScheduleTemplateRepository{
		byID: map[string]*models.ScheduleTemplate{
			"template-1": {ID: "template-1", Status: models.ScheduleTemplateStatusActive},
		},
	}
	svc := services.NewScheduleChangeService(repo, templates)
	return NewScheduleChangesHandler(svc), repo
}

func TestScheduleChangesCreate_Success(t *testing.T) {
	h, repo := newScheduleChangesTestHandler()

	body, _ := json.Marshal(map[string]any{
		"schedule_template_id": "template-1",
		"change_date":          "2024-03-11",
		"change_type":          "cancel",
		"reason":               "public holiday",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule-changes", bytes.NewReader(body))
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var dto scheduleChangeDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if dto.ChangeType != "cancel" || dto.ScheduleTemplateID != "template-1" {
		t.Errorf("unexpected DTO: %+v", dto)
	}
	if len(repo.byID) != 1 {
		t.Errorf("expected 1 stored change, got %d", len(repo.byID))
	}
}

func TestScheduleChangesCreate_RequiresAuth(t *testing.T) {
	h, _ := newScheduleChangesTestHandler()

	body, _ := json.Marshal(map[string]any{
		"schedule_template_id": "template-1",
		"change_date":          "2024-03-11",
		"change_type":          "cancel",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule-changes", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without user in context, got %d", rec.Code)
	}
}

func TestScheduleChangesCreate_ValidationByType(t *testing.T) {
	h, _ := newScheduleChangesTestHandler()

	// replace_teacher без new_teacher_id — 400
	body, _ := json.Marshal(map[string]any{
		"schedule_template_id": "template-1",
		"change_date":          "2024-03-11",
		"change_type":          "replace_teacher",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule-changes", bytes.NewReader(body))
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for replace_teacher without new_teacher_id, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestScheduleChangesCreate_DuplicateConflict(t *testing.T) {
	h, _ := newScheduleChangesTestHandler()

	payload := func() []byte {
		b, _ := json.Marshal(map[string]any{
			"schedule_template_id": "template-1",
			"change_date":          "2024-03-11",
			"change_type":          "cancel",
		})
		return b
	}

	ctxUser := func(r *http.Request) *http.Request {
		return r.WithContext(middleware.ContextWithUser(r.Context(), "admin-1", models.RoleAdmin))
	}

	first := httptest.NewRequest(http.MethodPost, "/api/v1/schedule-changes", bytes.NewReader(payload()))
	rec1 := httptest.NewRecorder()
	h.Create(rec1, ctxUser(first))
	if rec1.Code != http.StatusCreated {
		t.Fatalf("expected 201 on first create, got %d: %s", rec1.Code, rec1.Body.String())
	}

	second := httptest.NewRequest(http.MethodPost, "/api/v1/schedule-changes", bytes.NewReader(payload()))
	rec2 := httptest.NewRecorder()
	h.Create(rec2, ctxUser(second))
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate, got %d: %s", rec2.Code, rec2.Body.String())
	}
}

func TestScheduleChangesList_Success(t *testing.T) {
	h, _ := newScheduleChangesTestHandler()

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/schedule-changes?from=2024-03-11&to=2024-03-17", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"changes"`) {
		t.Errorf("expected response to contain 'changes' key, got: %s", rec.Body.String())
	}
}

func TestScheduleChangesList_InvalidDate(t *testing.T) {
	h, _ := newScheduleChangesTestHandler()

	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/schedule-changes?from=not-a-date&to=2024-03-17", nil)
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid from date, got %d", rec.Code)
	}
}

func TestScheduleChangesGet_NotFound(t *testing.T) {
	h, _ := newScheduleChangesTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/schedule-changes/missing", nil)
	req.SetPathValue("id", "missing")
	req = req.WithContext(middleware.ContextWithUser(req.Context(), "admin-1", models.RoleAdmin))
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestScheduleChangesUpdate_Success(t *testing.T) {
	h, _ := newScheduleChangesTestHandler()

	// Сначала создаём замену
	createBody, _ := json.Marshal(map[string]any{
		"schedule_template_id": "template-1",
		"change_date":          "2024-03-11",
		"change_type":          "cancel",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/schedule-changes", bytes.NewReader(createBody))
	createReq = createReq.WithContext(middleware.ContextWithUser(createReq.Context(), "admin-1", models.RoleAdmin))
	createRec := httptest.NewRecorder()
	h.Create(createRec, createReq)

	var created scheduleChangeDTO
	_ = json.Unmarshal(createRec.Body.Bytes(), &created)

	// Обновляем причину
	reason := "teacher on sick leave"
	updateBody, _ := json.Marshal(map[string]any{
		"change_date": "2024-03-11",
		"change_type": "cancel",
		"reason":      reason,
	})
	updateReq := httptest.NewRequest(http.MethodPatch, "/api/v1/schedule-changes/"+created.ID, bytes.NewReader(updateBody))
	updateReq.SetPathValue("id", created.ID)
	updateReq = updateReq.WithContext(middleware.ContextWithUser(updateReq.Context(), "admin-1", models.RoleAdmin))
	updateRec := httptest.NewRecorder()

	h.Update(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	var updated scheduleChangeDTO
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to unmarshal update response: %v", err)
	}
	if updated.Reason == nil || *updated.Reason != reason {
		t.Errorf("expected reason %q, got %v", reason, updated.Reason)
	}
}

func TestScheduleChangesDelete_SuccessAndNotFound(t *testing.T) {
	h, repo := newScheduleChangesTestHandler()

	// Создаём
	createBody, _ := json.Marshal(map[string]any{
		"schedule_template_id": "template-1",
		"change_date":          "2024-03-11",
		"change_type":          "cancel",
	})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/schedule-changes", bytes.NewReader(createBody))
	createReq = createReq.WithContext(middleware.ContextWithUser(createReq.Context(), "admin-1", models.RoleAdmin))
	createRec := httptest.NewRecorder()
	h.Create(createRec, createReq)

	var created scheduleChangeDTO
	_ = json.Unmarshal(createRec.Body.Bytes(), &created)

	// Удаляем
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/schedule-changes/"+created.ID, nil)
	deleteReq.SetPathValue("id", created.ID)
	deleteReq = deleteReq.WithContext(middleware.ContextWithUser(deleteReq.Context(), "admin-1", models.RoleAdmin))
	deleteRec := httptest.NewRecorder()
	h.Delete(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 on delete, got %d", deleteRec.Code)
	}
	if len(repo.byID) != 0 {
		t.Errorf("expected repository to be empty after delete, got %d", len(repo.byID))
	}

	// Повторное удаление — 404
	deleteAgain := httptest.NewRequest(http.MethodDelete, "/api/v1/schedule-changes/"+created.ID, nil)
	deleteAgain.SetPathValue("id", created.ID)
	deleteAgain = deleteAgain.WithContext(middleware.ContextWithUser(deleteAgain.Context(), "admin-1", models.RoleAdmin))
	deleteAgainRec := httptest.NewRecorder()
	h.Delete(deleteAgainRec, deleteAgain)

	if deleteAgainRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on second delete, got %d", deleteAgainRec.Code)
	}
}
