package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/qadam/backend/internal/middleware"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
	"github.com/qadam/backend/internal/storage"
)

// MaterialsHandler содержит HTTP-хендлеры для /api/v1/materials/*
// (раздел 35 API Plan): просмотр — все авторизованные; создание/загрузка/
// удаление — Teacher (свои материалы) и Admin.
type MaterialsHandler struct {
	materials *services.MaterialService
}

// NewMaterialsHandler создаёт MaterialsHandler с внедрённым сервисом.
func NewMaterialsHandler(materials *services.MaterialService) *MaterialsHandler {
	return &MaterialsHandler{materials: materials}
}

type materialDTO struct {
	ID          string  `json:"id"`
	SubjectID   string  `json:"subject_id"`
	SubjectName string  `json:"subject_name"`
	Category    string  `json:"category"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	CreatedBy   string  `json:"created_by"`
	AuthorName  string  `json:"author_name"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func materialDTOFromModel(m *models.Material) materialDTO {
	return materialDTO{
		ID:          m.ID,
		SubjectID:   m.SubjectID,
		SubjectName: m.SubjectName,
		Category:    string(m.Category),
		Title:       m.Title,
		Description: m.Description,
		CreatedBy:   m.CreatedBy,
		AuthorName:  m.AuthorName,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339),
	}
}

type materialFileDTO struct {
	ID          string  `json:"id"`
	MaterialID  string  `json:"material_id"`
	FileType    string  `json:"file_type"`
	FileName    string  `json:"file_name"`
	ExternalURL *string `json:"external_url"`
	SizeBytes   *int64  `json:"size_bytes"`
	MimeType    *string `json:"mime_type"`
	UploadedBy  string  `json:"uploaded_by"`
	CreatedAt   string  `json:"created_at"`
	// DownloadURL задаётся только для файлов из хранилища (не для ссылок).
	DownloadURL string `json:"download_url,omitempty"`
}

func materialFileDTOFromModel(f *models.MaterialFile) materialFileDTO {
	dto := materialFileDTO{
		ID:          f.ID,
		MaterialID:  f.MaterialID,
		FileType:    string(f.FileType),
		FileName:    f.FileName,
		ExternalURL: f.ExternalURL,
		SizeBytes:   f.SizeBytes,
		MimeType:    f.MimeType,
		UploadedBy:  f.UploadedBy,
		CreatedAt:   f.CreatedAt.Format(time.RFC3339),
	}
	if f.StorageKey != nil {
		dto.DownloadURL = "/api/v1/materials/files/" + f.ID + "/download"
	}
	return dto
}

// List обрабатывает GET /api/v1/materials?subject_id= — список материалов
// предмета. Все авторизованные (уточнение прав "свои предметы" для студента
// выполняется на уровне группы/предмета в будущих фазах).
func (h *MaterialsHandler) List(w http.ResponseWriter, r *http.Request) {
	subjectID := r.URL.Query().Get("subject_id")
	if subjectID == "" {
		writeError(w, http.StatusBadRequest, "subject_id query parameter is required")
		return
	}

	list, err := h.materials.ListBySubject(r.Context(), subjectID)
	if err != nil {
		handleMaterialError(w, err)
		return
	}

	dtos := make([]materialDTO, 0, len(list))
	for _, m := range list {
		dtos = append(dtos, materialDTOFromModel(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{"materials": dtos})
}

// Get обрабатывает GET /api/v1/materials/{id} — материал с файлами.
func (h *MaterialsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	m, files, err := h.materials.Get(r.Context(), id)
	if err != nil {
		handleNotFoundOr500(w, err, "material not found")
		return
	}

	fileDTOs := make([]materialFileDTO, 0, len(files))
	for _, f := range files {
		fileDTOs = append(fileDTOs, materialFileDTOFromModel(f))
	}

	dto := materialDTOFromModel(m)
	writeJSON(w, http.StatusOK, map[string]any{
		"material": dto,
		"files":    fileDTOs,
	})
}

type materialWriteRequest struct {
	SubjectID   string  `json:"subject_id"`
	Category    string  `json:"category"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
}

// Create обрабатывает POST /api/v1/materials — создание материала.
// Teacher (свои предметы — проверка владения материалом на обновлении/удалении;
// принадлежность предмета преподавателю уточняется через расписание) и Admin.
func (h *MaterialsHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req materialWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	m, err := h.materials.Create(r.Context(), userID, services.MaterialInput{
		SubjectID:   req.SubjectID,
		Category:    models.MaterialCategory(req.Category),
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		handleMaterialError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, materialDTOFromModel(m))
}

// Update обрабатывает PATCH /api/v1/materials/{id}.
func (h *MaterialsHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := r.PathValue("id")

	var req materialWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	m, err := h.materials.Update(r.Context(), userID, role == models.RoleAdmin, id, services.MaterialInput{
		SubjectID:   req.SubjectID,
		Category:    models.MaterialCategory(req.Category),
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		handleMaterialError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, materialDTOFromModel(m))
}

// Delete обрабатывает DELETE /api/v1/materials/{id}.
func (h *MaterialsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.materials.Delete(r.Context(), userID, role == models.RoleAdmin, r.PathValue("id")); err != nil {
		handleMaterialError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UploadFile обрабатывает POST /api/v1/materials/{id}/files — multipart-загрузка
// файла (поле "file"). Размер ограничен MaxUploadBytes; по разделу 14 большие
// файлы в будущем пойдут напрямую в S3 по pre-signed URL.
const MaxUploadBytes = 50 << 20 // 50 МБ

func (h *MaterialsHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	materialID := r.PathValue("id")

	// ограничиваем тело запроса с запасом на multipart-overhead
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes+(1<<20))

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "multipart form field 'file' is required")
		return
	}
	defer file.Close()

	f, err := h.materials.UploadFile(r.Context(), userID, role == models.RoleAdmin,
		materialID, header.Filename, header.Size, file)
	if err != nil {
		handleMaterialError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, materialFileDTOFromModel(f))
}

type materialLinkRequest struct {
	FileType string `json:"file_type"` // video_link | link
	URL      string `json:"url"`
	FileName string `json:"file_name"`
}

// AddLink обрабатывает POST /api/v1/materials/{id}/links — добавление
// внешней ссылки (видео YouTube/Vimeo или ресурса).
func (h *MaterialsHandler) AddLink(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	materialID := r.PathValue("id")

	var req materialLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	f, err := h.materials.AddLink(r.Context(), userID, role == models.RoleAdmin, materialID, services.MaterialLinkInput{
		FileType: models.MaterialFileType(req.FileType),
		URL:      req.URL,
		FileName: req.FileName,
	})
	if err != nil {
		handleMaterialError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, materialFileDTOFromModel(f))
}

// DownloadFile обрабатывает GET /api/v1/materials/files/{fileID}/download —
// стримит файл из хранилища клиенту.
func (h *MaterialsHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("fileID")

	f, rc, err := h.materials.OpenFile(r.Context(), fileID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeError(w, http.StatusNotFound, "file not found in storage")
			return
		}
		handleMaterialError(w, err)
		return
	}
	defer rc.Close()

	contentType := "application/octet-stream"
	if f.MimeType != nil && *f.MimeType != "" {
		contentType = *f.MimeType
	} else if mt := mime.TypeByExtension(fileNameExtension(f.FileName)); mt != "" {
		contentType = mt
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%q", f.FileName))
	if f.SizeBytes != nil {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", *f.SizeBytes))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

// DeleteFile обрабатывает DELETE /api/v1/materials/files/{fileID}.
func (h *MaterialsHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	userID, role, ok := userAndRole(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.materials.DeleteFile(r.Context(), userID, role == models.RoleAdmin, r.PathValue("fileID")); err != nil {
		handleMaterialError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// userAndRole извлекает ID и роль пользователя, помещённые middleware Auth.
func userAndRole(r *http.Request) (userID string, role models.RoleKey, ok bool) {
	userID, ok = middleware.UserIDFromContext(r.Context())
	if !ok {
		return "", "", false
	}
	role, ok = middleware.RoleFromContext(r.Context())
	if !ok {
		return "", "", false
	}
	return userID, role, true
}

func fileNameExtension(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i:]
		}
	}
	return ""
}

func handleMaterialError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		writeError(w, http.StatusNotFound, "material not found")
	case errors.Is(err, services.ErrMaterialForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, services.ErrMaterialValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, storage.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
