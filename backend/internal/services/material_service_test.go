package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/storage"
)

// fakeMaterialRepository — in-memory реализация
// repositories.MaterialRepository для unit-тестов.
type fakeMaterialRepository struct {
	byID              map[string]*models.Material
	files             map[string]*models.MaterialFile
	nextID            int
	deletedIDs        []string
	subscriberUserIDs []string // подписчики предмета для уведомлений (Phase 8)
}

func newFakeMaterialRepository() *fakeMaterialRepository {
	return &fakeMaterialRepository{
		byID:   make(map[string]*models.Material),
		files:  make(map[string]*models.MaterialFile),
	}
}

func (f *fakeMaterialRepository) genID(prefix string) string {
	f.nextID++
	return prefix + "-" + string(rune('0'+f.nextID))
}

// ListSubjectSubscriberUserIDs — заглушка для уведомлений (Phase 8):
// возвращает настроенный список подписчиков (subscriberUserIDs).
func (f *fakeMaterialRepository) ListSubjectSubscriberUserIDs(_ context.Context, _ string) ([]string, error) {
	return f.subscriberUserIDs, nil
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
	if _, ok := f.byID[id]; !ok {
		return repositories.ErrNotFound
	}
	delete(f.byID, id)
	f.deletedIDs = append(f.deletedIDs, id)
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
	if _, ok := f.files[id]; !ok {
		return repositories.ErrNotFound
	}
	delete(f.files, id)
	return nil
}

func newMaterialTestService(t *testing.T) (*MaterialService, *fakeMaterialRepository) {
	t.Helper()
	fs, err := storage.NewLocalFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("failed to init storage: %v", err)
	}
	repo := newFakeMaterialRepository()
	return NewMaterialService(repo, fs, nil), repo
}

func TestMaterialService_Create_Success(t *testing.T) {
	svc, _ := newMaterialTestService(t)

	m, err := svc.Create(context.Background(), "teacher-1", MaterialInput{
		SubjectID: "subject-1",
		Title:     "Лекция 1: Введение",
		Category:  models.MaterialCategoryLecture,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID == "" || m.CreatedBy != "teacher-1" {
		t.Errorf("unexpected material: %+v", m)
	}
}

func TestMaterialService_Create_DefaultCategoryAndValidation(t *testing.T) {
	svc, _ := newMaterialTestService(t)
	ctx := context.Background()

	// Категория по умолчанию — extra
	m, err := svc.Create(ctx, "teacher-1", MaterialInput{SubjectID: "s1", Title: "T"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Category != models.MaterialCategoryExtra {
		t.Errorf("expected default category extra, got %q", m.Category)
	}

	// Без title
	_, err = svc.Create(ctx, "teacher-1", MaterialInput{SubjectID: "s1"})
	if !errors.Is(err, ErrMaterialValidation) {
		t.Errorf("expected validation error without title, got %v", err)
	}

	// Неизвестная категория
	_, err = svc.Create(ctx, "teacher-1", MaterialInput{SubjectID: "s1", Title: "T", Category: "bogus"})
	if !errors.Is(err, ErrMaterialValidation) {
		t.Errorf("expected validation error for unknown category, got %v", err)
	}
}

func TestMaterialService_Update_ForbiddenForOtherTeacher(t *testing.T) {
	svc, _ := newMaterialTestService(t)
	ctx := context.Background()

	m, err := svc.Create(ctx, "teacher-1", MaterialInput{SubjectID: "s1", Title: "Original"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Другой преподаватель не может редактировать
	_, err = svc.Update(ctx, "teacher-2", false, m.ID, MaterialInput{SubjectID: "s1", Title: "Hacked"})
	if !errors.Is(err, ErrMaterialForbidden) {
		t.Fatalf("expected ErrMaterialForbidden, got %v", err)
	}

	// Автор может
	updated, err := svc.Update(ctx, "teacher-1", false, m.ID, MaterialInput{SubjectID: "s1", Title: "Updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Updated" {
		t.Errorf("expected title Updated, got %q", updated.Title)
	}

	// Admin может чужой
	_, err = svc.Update(ctx, "admin-1", true, m.ID, MaterialInput{SubjectID: "s1", Title: "AdminEdit"})
	if err != nil {
		t.Fatalf("admin update should pass, got %v", err)
	}
}

func TestMaterialService_Delete_ForbiddenForOtherTeacher(t *testing.T) {
	svc, _ := newMaterialTestService(t)
	ctx := context.Background()

	m, _ := svc.Create(ctx, "teacher-1", MaterialInput{SubjectID: "s1", Title: "X"})

	if err := svc.Delete(ctx, "teacher-2", false, m.ID); !errors.Is(err, ErrMaterialForbidden) {
		t.Fatalf("expected ErrMaterialForbidden, got %v", err)
	}
	if err := svc.Delete(ctx, "teacher-1", false, m.ID); err != nil {
		t.Fatalf("author delete should pass, got %v", err)
	}
}

func TestMaterialService_UploadFile_Success(t *testing.T) {
	svc, repo := newMaterialTestService(t)
	ctx := context.Background()

	m, _ := svc.Create(ctx, "teacher-1", MaterialInput{SubjectID: "subject-9", Title: "With file"})

	content := []byte("%PDF-1.4 fake pdf content")
	f, err := svc.UploadFile(ctx, "teacher-1", false, m.ID, "lecture1.pdf", int64(len(content)), bytes.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.FileType != models.MaterialFileTypePDF {
		t.Errorf("expected pdf type, got %q", f.FileType)
	}
	if f.StorageKey == nil || !strings.HasPrefix(*f.StorageKey, "materials/subject-9/") {
		t.Errorf("unexpected storage key: %v", f.StorageKey)
	}
	if _, ok := repo.files[f.ID]; !ok {
		t.Error("expected file metadata to be saved")
	}

	// Файл можно открыть и прочитать
	_, rc, err := svc.OpenFile(ctx, f.ID)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch after upload/open")
	}
}

func TestMaterialService_UploadFile_ForbiddenAndUnsupported(t *testing.T) {
	svc, _ := newMaterialTestService(t)
	ctx := context.Background()

	m, _ := svc.Create(ctx, "teacher-1", MaterialInput{SubjectID: "s1", Title: "X"})

	// чужой преподаватель
	_, err := svc.UploadFile(ctx, "teacher-2", false, m.ID, "a.pdf", 3, strings.NewReader("abc"))
	if !errors.Is(err, ErrMaterialForbidden) {
		t.Fatalf("expected ErrMaterialForbidden, got %v", err)
	}

	// неподдерживаемый тип (exe)
	if _, err := svc.UploadFile(ctx, "teacher-1", false, m.ID, "virus.exe", 3, strings.NewReader("abc")); !errors.Is(err, ErrMaterialValidation) {
		t.Fatalf("expected validation error for .exe, got %v", err)
	}
}

func TestMaterialService_AddLink(t *testing.T) {
	svc, _ := newMaterialTestService(t)
	ctx := context.Background()

	m, _ := svc.Create(ctx, "teacher-1", MaterialInput{SubjectID: "s1", Title: "Links"})

	// Валидная видео-ссылка
	f, err := svc.AddLink(ctx, "teacher-1", false, m.ID, MaterialLinkInput{
		FileType: models.MaterialFileTypeVideoLink,
		URL:      "https://youtube.com/watch?v=abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.ExternalURL == nil || f.FileName != "https://youtube.com/watch?v=abc" {
		t.Errorf("unexpected link file: %+v", f)
	}

	// Не-http URL
	_, err = svc.AddLink(ctx, "teacher-1", false, m.ID, MaterialLinkInput{
		FileType: models.MaterialFileTypeLink,
		URL:      "ftp://example.com",
	})
	if !errors.Is(err, ErrMaterialValidation) {
		t.Errorf("expected validation error for ftp URL, got %v", err)
	}

	// Неправильный тип
	_, err = svc.AddLink(ctx, "teacher-1", false, m.ID, MaterialLinkInput{
		FileType: models.MaterialFileTypePDF,
		URL:      "https://example.com/doc.pdf",
	})
	if !errors.Is(err, ErrMaterialValidation) {
		t.Errorf("expected validation error for pdf as link, got %v", err)
	}
}

func TestMaterialService_DeleteFile_RemovesFromStorage(t *testing.T) {
	svc, _ := newMaterialTestService(t)
	ctx := context.Background()

	m, _ := svc.Create(ctx, "teacher-1", MaterialInput{SubjectID: "s1", Title: "X"})
	f, err := svc.UploadFile(ctx, "teacher-1", false, m.ID, "doc.pdf", 3, strings.NewReader("abc"))
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	if err := svc.DeleteFile(ctx, "teacher-1", false, f.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	_, _, err = svc.OpenFile(ctx, f.ID)
	if err == nil {
		t.Error("expected error opening deleted file")
	}
}

func TestMaterialService_ListBySubject_RequiresSubjectID(t *testing.T) {
	svc, _ := newMaterialTestService(t)
	_, err := svc.ListBySubject(context.Background(), "")
	if !errors.Is(err, ErrMaterialValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
