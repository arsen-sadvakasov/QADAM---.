package models

import "time"

// MaterialCategory — категория материала (раздел 21 спецификации:
// Лекции / Практические / Лабораторные / Дополнительные).
type MaterialCategory string

const (
	MaterialCategoryLecture  MaterialCategory = "lecture"
	MaterialCategoryPractice MaterialCategory = "practice"
	MaterialCategoryLab      MaterialCategory = "lab"
	MaterialCategoryExtra    MaterialCategory = "extra"
)

// MaterialFileType — тип файла/ссылки в материале (раздел 34.1).
type MaterialFileType string

const (
	MaterialFileTypePDF       MaterialFileType = "pdf"
	MaterialFileTypeDocx      MaterialFileType = "docx"
	MaterialFileTypePptx      MaterialFileType = "pptx"
	MaterialFileTypeImage     MaterialFileType = "image"
	MaterialFileTypeVideoLink MaterialFileType = "video_link"
	MaterialFileTypeLink      MaterialFileType = "link"
)

// IsExternalLink — истинно для типов, хранящихся как внешняя ссылка,
// а не как файл в объектном хранилище.
func (t MaterialFileType) IsExternalLink() bool {
	return t == MaterialFileTypeVideoLink || t == MaterialFileTypeLink
}

// Material — логическая единица материала (таблица materials):
// например "Лекция №1: Введение в БД" по предмету.
type Material struct {
	ID          string
	SubjectID   string
	Category    MaterialCategory
	Title       string
	Description *string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time

	// Поля, подгружаемые джойнами для отображения в API.
	SubjectName string
	AuthorName  string
}

// MaterialFile — физический файл/ссылка, привязанная к материалу
// (таблица material_files). Содержимое файла хранится во внешнем
// хранилище (FileStorage), здесь — только метаданные (раздел 14).
type MaterialFile struct {
	ID          string
	MaterialID  string
	FileType    MaterialFileType
	StorageKey  *string
	ExternalURL *string
	FileName    string
	SizeBytes   *int64
	MimeType    *string
	UploadedBy  string
	CreatedAt   time.Time
}
