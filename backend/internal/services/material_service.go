package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/storage"
)

// ErrMaterialValidation возвращается при нарушении правил валидации данных
// материала/файла — всегда должна отвечать HTTP 400, а не 500.
var ErrMaterialValidation = errors.New("invalid material input")

// ErrMaterialForbidden возвращается, когда пользователь не имеет права
// выполнить операцию с материалом (например, преподаватель редактирует
// чужой материал) — HTTP 403.
var ErrMaterialForbidden = errors.New("not allowed to manage this material")

// MaxUploadedFileBytes — ограничение размера загружаемого файла (50 МБ)
// (раздел 14: большие файлы в будущем пойдут напрямую в S3 по pre-signed URL).
const MaxUploadedFileBytes = 50 << 20

// допустимые расширения загружаемых файлов (раздел 21: PDF, DOCX, PPTX,
// изображения; видео — ссылками на YouTube/Vimeo).
var allowedUploadExtensions = map[string]models.MaterialFileType{
	".pdf":  models.MaterialFileTypePDF,
	".docx": models.MaterialFileTypeDocx,
	".doc":  models.MaterialFileTypeDocx,
	".pptx": models.MaterialFileTypePptx,
	".ppt":  models.MaterialFileTypePptx,
	".png":  models.MaterialFileTypeImage,
	".jpg":  models.MaterialFileTypeImage,
	".jpeg": models.MaterialFileTypeImage,
	".webp": models.MaterialFileTypeImage,
	".gif":  models.MaterialFileTypeImage,
}

var validCategories = map[models.MaterialCategory]bool{
	models.MaterialCategoryLecture:  true,
	models.MaterialCategoryPractice: true,
	models.MaterialCategoryLab:      true,
	models.MaterialCategoryExtra:    true,
}

var validLinkTypes = map[models.MaterialFileType]bool{
	models.MaterialFileTypeVideoLink: true,
	models.MaterialFileTypeLink:      true,
}

// MaterialInput — входные данные для создания/редактирования материала.
type MaterialInput struct {
	SubjectID   string
	Category    models.MaterialCategory
	Title       string
	Description *string
}

// MaterialLinkInput — входные данные для добавления внешней ссылки
// (видео/ресурс) в материал.
type MaterialLinkInput struct {
	FileType models.MaterialFileType
	URL      string
	FileName string
}

// MaterialService реализует бизнес-логику учебных материалов (Phase 7):
// CRUD материалов, загрузка файлов через FileStorage, права доступа
// (upload: teacher по своим предметам + admin; просмотр: все авторизованные).
type MaterialService struct {
	materials repositories.MaterialRepository
	storage   storage.FileStorage
}

// NewMaterialService создаёт MaterialService с внедрёнными зависимостями.
func NewMaterialService(materials repositories.MaterialRepository, fileStorage storage.FileStorage) *MaterialService {
	return &MaterialService{materials: materials, storage: fileStorage}
}

// Create создаёт материал. createdByID — ID пользователя (teacher/admin).
// Проверка "преподаватель ведёт этот предмет" выполняется на уровне хендлера
// через расписание (teacher_subjects); сервис проверяет корректность данных.
func (s *MaterialService) Create(ctx context.Context, createdByID string, in MaterialInput) (*models.Material, error) {
	if in.SubjectID == "" || in.Title == "" {
		return nil, fmt.Errorf("%w: subject_id and title are required", ErrMaterialValidation)
	}
	if in.Category == "" {
		in.Category = models.MaterialCategoryExtra
	}
	if !validCategories[in.Category] {
		return nil, fmt.Errorf("%w: unknown category %q", ErrMaterialValidation, in.Category)
	}

	m := &models.Material{
		SubjectID:   in.SubjectID,
		Category:    in.Category,
		Title:       in.Title,
		Description: in.Description,
		CreatedBy:   createdByID,
	}

	id, err := s.materials.Create(ctx, m)
	if err != nil {
		return nil, err
	}
	return s.materials.FindByID(ctx, id)
}

// Get возвращает материал вместе с его файлами.
func (s *MaterialService) Get(ctx context.Context, id string) (*models.Material, []*models.MaterialFile, error) {
	m, err := s.materials.FindByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	files, err := s.materials.ListFilesByMaterial(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return m, files, nil
}

// ListBySubject возвращает материалы предмета.
func (s *MaterialService) ListBySubject(ctx context.Context, subjectID string) ([]*models.Material, error) {
	if subjectID == "" {
		return nil, fmt.Errorf("%w: subject_id is required", ErrMaterialValidation)
	}
	return s.materials.ListBySubject(ctx, subjectID)
}

// Update редактирует материал. Проверяется право: admin — любой,
// teacher — только созданный им.
func (s *MaterialService) Update(ctx context.Context, userID string, isAdmin bool, id string, in MaterialInput) (*models.Material, error) {
	existing, err := s.materials.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isAdmin && existing.CreatedBy != userID {
		return nil, ErrMaterialForbidden
	}

	if in.SubjectID == "" || in.Title == "" {
		return nil, fmt.Errorf("%w: subject_id and title are required", ErrMaterialValidation)
	}
	if in.Category == "" {
		in.Category = existing.Category
	}
	if !validCategories[in.Category] {
		return nil, fmt.Errorf("%w: unknown category %q", ErrMaterialValidation, in.Category)
	}

	existing.SubjectID = in.SubjectID
	existing.Category = in.Category
	existing.Title = in.Title
	existing.Description = in.Description

	if err := s.materials.Update(ctx, existing); err != nil {
		return nil, err
	}
	return s.materials.FindByID(ctx, id)
}

// Delete удаляет материал (soft delete). Проверяется право: admin — любой,
// teacher — только созданный им.
func (s *MaterialService) Delete(ctx context.Context, userID string, isAdmin bool, id string) error {
	existing, err := s.materials.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !isAdmin && existing.CreatedBy != userID {
		return ErrMaterialForbidden
	}
	return s.materials.SoftDelete(ctx, id)
}

// UploadFile загружает файл в хранилище и создаёт запись метаданных.
// Поток по разделу 14: backend валидирует и сохраняет файл, клиент не
// обращается к хранилищу напрямую (pre-signed URL появится с MinIO).
func (s *MaterialService) UploadFile(ctx context.Context, userID string, isAdmin bool, materialID, originalFileName string, size int64, r io.Reader) (*models.MaterialFile, error) {
	m, err := s.materials.FindByID(ctx, materialID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && m.CreatedBy != userID {
		return nil, ErrMaterialForbidden
	}

	fileType, ok := uploadTypeForFileName(originalFileName)
	if !ok {
		return nil, fmt.Errorf("%w: unsupported file type for %q (allowed: pdf, docx, pptx, images)", ErrMaterialValidation, originalFileName)
	}
	if size > MaxUploadedFileBytes {
		return nil, fmt.Errorf("%w: file exceeds maximum size of %d bytes", ErrMaterialValidation, MaxUploadedFileBytes)
	}

	key, err := storage.NewKey(m.SubjectID, originalFileName)
	if err != nil {
		return nil, err
	}

	if err := s.storage.Upload(ctx, key, r, ""); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	f := &models.MaterialFile{
		MaterialID: materialID,
		FileType:   fileType,
		StorageKey: &key,
		FileName:   originalFileName,
		SizeBytes:  &size,
		UploadedBy: userID,
	}

	fileID, err := s.materials.CreateFile(ctx, f)
	if err != nil {
		// откатываем загрузку, чтобы не оставлять «осиротевший» объект
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}
	return s.materials.FindFileByID(ctx, fileID)
}

// AddLink добавляет внешнюю ссылку (видео/ресурс) в материал.
func (s *MaterialService) AddLink(ctx context.Context, userID string, isAdmin bool, materialID string, in MaterialLinkInput) (*models.MaterialFile, error) {
	m, err := s.materials.FindByID(ctx, materialID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && m.CreatedBy != userID {
		return nil, ErrMaterialForbidden
	}

	if !validLinkTypes[in.FileType] {
		return nil, fmt.Errorf("%w: file_type must be video_link or link for external references", ErrMaterialValidation)
	}
	if !strings.HasPrefix(in.URL, "http://") && !strings.HasPrefix(in.URL, "https://") {
		return nil, fmt.Errorf("%w: url must start with http:// or https://", ErrMaterialValidation)
	}
	fileName := in.FileName
	if fileName == "" {
		fileName = in.URL
	}

	f := &models.MaterialFile{
		MaterialID:  materialID,
		FileType:    in.FileType,
		ExternalURL: &in.URL,
		FileName:    fileName,
		UploadedBy:  userID,
	}

	fileID, err := s.materials.CreateFile(ctx, f)
	if err != nil {
		return nil, err
	}
	return s.materials.FindFileByID(ctx, fileID)
}

// OpenFile возвращает поток чтения файла из хранилища (для стриминга клиенту
// через API). Файлы-ссылки (external_url) не открываются этим методом.
func (s *MaterialService) OpenFile(ctx context.Context, fileID string) (*models.MaterialFile, io.ReadSeekCloser, error) {
	f, err := s.materials.FindFileByID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if f.StorageKey == nil {
		return nil, nil, fmt.Errorf("%w: file is an external link", ErrMaterialValidation)
	}
	rc, err := s.storage.Open(ctx, *f.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return f, rc, nil
}

// DeleteFile удаляет файл материала: запись из БД и объект из хранилища.
func (s *MaterialService) DeleteFile(ctx context.Context, userID string, isAdmin bool, fileID string) error {
	f, err := s.materials.FindFileByID(ctx, fileID)
	if err != nil {
		return err
	}
	m, err := s.materials.FindByID(ctx, f.MaterialID)
	if err != nil {
		return err
	}
	if !isAdmin && m.CreatedBy != userID {
		return ErrMaterialForbidden
	}
	if err := s.materials.DeleteFile(ctx, fileID); err != nil {
		return err
	}
	if f.StorageKey != nil {
		// ошибка удаления объекта не должна откатывать удаление записи —
		// объект без метаданных безвреден; чистится администрированием хранилища
		_ = s.storage.Delete(ctx, *f.StorageKey)
	}
	return nil
}

// uploadTypeForFileName определяет MaterialFileType по расширению файла.
func uploadTypeForFileName(name string) (models.MaterialFileType, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	ft, ok := allowedUploadExtensions[ext]
	return ft, ok
}
