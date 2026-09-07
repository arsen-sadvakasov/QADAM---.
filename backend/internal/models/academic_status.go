package models

// AcademicStatus — справочная сущность успеваемости студента
// (таблица academic_statuses, раздел 34.1 спецификации).
type AcademicStatus struct {
	ID        string
	Key       string
	Name      string
	SortOrder int
}
