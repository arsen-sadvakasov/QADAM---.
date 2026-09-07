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

type fakeRoomRepository struct {
	byID map[string]*models.Room
}

func newFakeRoomRepository() *fakeRoomRepository {
	return &fakeRoomRepository{byID: make(map[string]*models.Room)}
}

func (f *fakeRoomRepository) FindByID(_ context.Context, id string) (*models.Room, error) {
	rm, ok := f.byID[id]
	if !ok {
		return nil, repositories.ErrNotFound
	}
	return rm, nil
}

func (f *fakeRoomRepository) List(_ context.Context) ([]*models.Room, error) {
	var result []*models.Room
	for _, rm := range f.byID {
		result = append(result, rm)
	}
	return result, nil
}

func (f *fakeRoomRepository) Create(_ context.Context, rm *models.Room) (string, error) {
	rm.ID = "generated-room"
	f.byID[rm.ID] = rm
	return rm.ID, nil
}

func (f *fakeRoomRepository) Update(_ context.Context, rm *models.Room) error {
	f.byID[rm.ID] = rm
	return nil
}

func (f *fakeRoomRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

func TestRoomsCreate_Success(t *testing.T) {
	h := NewRoomsHandler(newFakeRoomRepository())
	body, _ := json.Marshal(map[string]string{"number": "301", "type": "lecture"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRoomsCreate_InvalidType(t *testing.T) {
	h := NewRoomsHandler(newFakeRoomRepository())
	body, _ := json.Marshal(map[string]string{"number": "301", "type": "not-a-real-type"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRoomsCreate_MissingNumber(t *testing.T) {
	h := NewRoomsHandler(newFakeRoomRepository())
	body, _ := json.Marshal(map[string]string{"type": "lecture"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rooms", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRoomsList_Success(t *testing.T) {
	repo := newFakeRoomRepository()
	repo.byID["r1"] = &models.Room{ID: "r1", Number: "301", Type: models.RoomTypeLecture}
	h := NewRoomsHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rooms", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
