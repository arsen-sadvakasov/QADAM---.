package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
	"github.com/qadam/backend/internal/storage"
)

// fakeMaterialRepository — in-memory реализация
// repositories.MaterialRepository для unit-тестов хендлера.
type fakeMaterialRepository struct {
	byID   map[string]*models.Material
	files  map[string]*models.MaterialFile
	nextID int
}

func (f *fakeMaterialRepository) genID(prefix string) string {
	f.nextID++
	return prefix + "-" + string(rune('0'+f.nextID))
}

// ListSubjectSubscriberUserIDs — заглушка для уведомлений (Phase 8).
func (f *fakeMaterialRepository) ListSubjectSubscriberUserIDs(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (f *fakeMaterialRepository) FindByID(_ context.Context, id string) (*models.Material, error) {
	if m, ok := f.byID[id]; ok {
		return m, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeMaterialRepository) ListBySubject(_ context.Context, subjectID string) ([]*models.Material, error) {
	var result []*models.Material
	for _, m := range f.byID {
		if m.SubjectID == subjectID {
			result = append(result, m)
		}
	}
	return result, nil
}

func (f *fakeMaterialRepository) Create(_ context.Context, m *models.Material) (string, error) {
	m.ID = f.genID("mat")
	clone := *m
	f.byID[m.ID] = &clone
	return m.ID, nil
}

func (f *fakeMaterialRepository) Update(_ context.Context, m *models.Material) error {
	if _, ok := f.byID[m.ID]; !ok {
		return repositories.ErrNotFound
	}
	clone := *m
	f.byID[m.ID] = &clone
	return nil
}

func (f *fakeMaterialRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

func (f *fakeMaterialRepository) FindFileByID(_ context.Context, id string) (*models.MaterialFile, error) {
	if file, ok := f.files[id]; ok {
		return file, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeMaterialRepository) ListFilesByMaterial(_ context.Context, materialID string) ([]*models.MaterialFile, error) {
	var result []*models.MaterialFile
	for _, file := range f.files {
		if file.MaterialID == materialID {
			result = append(result, file)
		}
	}
	return result, nil
}

func (f *fakeMaterialRepository) CreateFile(_ context.Context, file *models.MaterialFile) (string, error) {
	file.ID = f.genID("file")
	clone := *file
	f.files[file.ID] = &clone
	return file.ID, nil
}

func (f *fakeMaterialRepository) DeleteFile(_ context.Context, id string) error {
	delete(f.files, id)
	return nil
}

func newMaterialsTestHandler() (*MaterialsHandler, *fakeMaterialRepository) {
	fs, err := storage.NewLocalFileStorage("/tmp/qadam-test-materials")
	if err != nil {
		panic(err)
	}
	repo := &fakeMaterialRepository{
		byID:  make(map[string]*models.Material),
		files: make(map[string]*models.MaterialFile),
	}
	svc := services.NewMaterialService(repo, fs, nil)
	return NewMaterialsHandler(svc), repo
}

func authedTeacher(userID string, role models.RoleKey) func(*http.Request) *http.Request {
	return func(r *http.Request) *http.Request {
		return r.WithContext(middleware.ContextWithUser(r.Context(), userID, role))
	}
}

func TestMaterialsCreate_Success(t *testing.T) {
	h, _ := newMaterialsTestHandler()

	body, _ := json.Marshal(map[string]any{
		"subject_id": "subject-1",
		"category":   "lecture",
		"title":      "Лекция 1",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/materials", bytes.NewReader(body))
	req = authedTeacher("teacher-1", models.RoleTeacher)(req)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var dto materialDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if dto.Title != "Лекция 1" || dto.Category != "lecture" {
		t.Errorf("unexpected DTO: %+v", dto)
	}
}

func TestMaterialsCreate_Validation(t *testing.T) {
	h, _ := newMaterialsTestHandler()

	// без title
	body, _ := json.Marshal(map[string]any{"subject_id": "subject-1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/materials", bytes.NewReader(body))
	req = authedTeacher("teacher-1", models.RoleTeacher)(req)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without title, got %d", rec.Code)
	}
}

func TestMaterialsList_RequiresSubjectID(t *testing.T) {
	h, _ := newMaterialsTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/materials", nil)
	req = authedTeacher("teacher-1", models.RoleTeacher)(req)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without subject_id, got %d", rec.Code)
	}
}

func TestMaterialsUpdate_ForbiddenForOtherTeacher(t *testing.T) {
	h, repo := newMaterialsTestHandler()

	repo.byID["mat-1"] = &models.Material{
		ID: "mat-1", SubjectID: "subject-1", Title: "Original",
		Category: models.MaterialCategoryExtra, CreatedBy: "teacher-1",
		SubjectName: "S", AuthorName: "T1",
	}

	body, _ := json.Marshal(map[string]any{"subject_id": "subject-1", "title": "Hacked"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/materials/mat-1", bytes.NewReader(body))
	req.SetPathValue("id", "mat-1")
	req = authedTeacher("teacher-2", models.RoleTeacher)(req)
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestMaterialsUploadFile_Success(t *testing.T) {
	h, _ := newMaterialsTestHandler()

	// создаём материал
	body, _ := json.Marshal(map[string]any{"subject_id": "subject-1", "title": "M"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/materials", bytes.NewReader(body))
	createReq = authedTeacher("teacher-1", models.RoleTeacher)(createReq)
	createRec := httptest.NewRecorder()
	h.Create(createRec, createReq)

	var created materialDTO
	_ = json.Unmarshal(createRec.Body.Bytes(), &created)

	// multipart upload
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "lecture.pdf")
	fw.Write([]byte("%PDF-1.4 test"))
	mw.Close()

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/materials/"+created.ID+"/files", &buf)
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	uploadReq.SetPathValue("id", created.ID)
	uploadReq = authedTeacher("teacher-1", models.RoleTeacher)(uploadReq)
	uploadRec := httptest.NewRecorder()

	h.UploadFile(uploadRec, uploadReq)

	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", uploadRec.Code, uploadRec.Body.String())
	}
	var fileDTO materialFileDTO
	_ = json.Unmarshal(uploadRec.Body.Bytes(), &fileDTO)
	if fileDTO.FileType != "pdf" || fileDTO.DownloadURL == "" {
		t.Errorf("unexpected file DTO: %+v", fileDTO)
	}
}

func TestMaterialsDownloadFile(t *testing.T) {
	h, _ := newMaterialsTestHandler()

	// материал + файл
	body, _ := json.Marshal(map[string]any{"subject_id": "subject-1", "title": "M"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/materials", bytes.NewReader(body))
	createReq = authedTeacher("teacher-1", models.RoleTeacher)(createReq)
	createRec := httptest.NewRecorder()
	h.Create(createRec, createReq)
	var created materialDTO
	_ = json.Unmarshal(createRec.Body.Bytes(), &created)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "notes.pdf")
	fw.Write([]byte("hello world"))
	mw.Close()
	uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/materials/"+created.ID+"/files", &buf)
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	uploadReq.SetPathValue("id", created.ID)
	uploadReq = authedTeacher("teacher-1", models.RoleTeacher)(uploadReq)
	uploadRec := httptest.NewRecorder()
	h.UploadFile(uploadRec, uploadReq)

	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("upload failed with %d: %s (material id %q)", uploadRec.Code, uploadRec.Body.String(), created.ID)
	}

	var fileDTO materialFileDTO
	if err := json.Unmarshal(uploadRec.Body.Bytes(), &fileDTO); err != nil {
		t.Fatalf("unmarshal failed: %v (body: %s)", err, uploadRec.Body.String())
	}
	if fileDTO.DownloadURL == "" {
		t.Fatalf("expected non-empty download URL, got: %+v", fileDTO)
	}

	// скачивание
	dlReq := httptest.NewRequest(http.MethodGet, fileDTO.DownloadURL, nil)
	dlReq.SetPathValue("fileID", fileDTO.ID)
	dlReq = authedTeacher("teacher-1", models.RoleTeacher)(dlReq)
	dlRec := httptest.NewRecorder()
	h.DownloadFile(dlRec, dlReq)

	if dlRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", dlRec.Code)
	}
	if !strings.Contains(dlRec.Body.String(), "hello world") {
		t.Errorf("expected file content, got %q", dlRec.Body.String())
	}
}

func TestMaterialsAddLink(t *testing.T) {
	h, _ := newMaterialsTestHandler()

	body, _ := json.Marshal(map[string]any{"subject_id": "subject-1", "title": "M"})
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/materials", bytes.NewReader(body))
	createReq = authedTeacher("teacher-1", models.RoleTeacher)(createReq)
	createRec := httptest.NewRecorder()
	h.Create(createRec, createReq)
	var created materialDTO
	_ = json.Unmarshal(createRec.Body.Bytes(), &created)

	linkBody, _ := json.Marshal(map[string]any{
		"file_type": "video_link",
		"url":       "https://youtube.com/watch?v=xyz",
	})
	linkReq := httptest.NewRequest(http.MethodPost, "/api/v1/materials/"+created.ID+"/links", bytes.NewReader(linkBody))
	linkReq.SetPathValue("id", created.ID)
	linkReq = authedTeacher("teacher-1", models.RoleTeacher)(linkReq)
	linkRec := httptest.NewRecorder()
	h.AddLink(linkRec, linkReq)

	if linkRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", linkRec.Code, linkRec.Body.String())
	}
	var fileDTO materialFileDTO
	_ = json.Unmarshal(linkRec.Body.Bytes(), &fileDTO)
	if fileDTO.FileType != "video_link" || fileDTO.ExternalURL == nil {
		t.Errorf("unexpected link DTO: %+v", fileDTO)
	}
}

func TestMaterialsDeleteFile_NotFound(t *testing.T) {
	h, _ := newMaterialsTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/materials/files/missing", nil)
	req.SetPathValue("fileID", "missing")
	req = authedTeacher("teacher-1", models.RoleTeacher)(req)
	rec := httptest.NewRecorder()

	h.DeleteFile(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
