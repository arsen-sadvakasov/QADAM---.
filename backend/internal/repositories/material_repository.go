package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// MaterialRepository описывает доступ к таблицам materials/material_files
// (Phase 7 — Materials, разделы 21 и 34.1 спецификации).
type MaterialRepository interface {
	// Материалы
	FindByID(ctx context.Context, id string) (*models.Material, error)
	ListBySubject(ctx context.Context, subjectID string) ([]*models.Material, error)
	// ListSubjectSubscriberUserIDs возвращает ID аккаунтов пользователей,
	// которым читается предмет (активные студенты групп, изучающих предмет
	// по расписанию) — для уведомлений о новых материалах (Phase 8).
	ListSubjectSubscriberUserIDs(ctx context.Context, subjectID string) ([]string, error)
	Create(ctx context.Context, m *models.Material) (string, error)
	Update(ctx context.Context, m *models.Material) error
	SoftDelete(ctx context.Context, id string) error

	// Файлы
	FindFileByID(ctx context.Context, id string) (*models.MaterialFile, error)
	ListFilesByMaterial(ctx context.Context, materialID string) ([]*models.MaterialFile, error)
	CreateFile(ctx context.Context, f *models.MaterialFile) (string, error)
	DeleteFile(ctx context.Context, id string) error
}

type pgMaterialRepository struct {
	pool *pgxpool.Pool
}

// NewMaterialRepository создаёт реализацию MaterialRepository на базе pgx.
func NewMaterialRepository(pool *pgxpool.Pool) MaterialRepository {
	return &pgMaterialRepository{pool: pool}
}

// materialSelectColumns подгружает название предмета и имя автора джойнами,
// чтобы список материалов отдавался одним запросом (FR-11 UX).
const materialSelectColumns = `
	m.id, m.subject_id, m.category, m.title, m.description, m.created_by,
	m.created_at, m.updated_at, m.deleted_at,
	sub.name, u.full_name
`

const materialFrom = `
	FROM materials m
	JOIN subjects sub ON sub.id = m.subject_id
	JOIN users u ON u.id = m.created_by
`

func scanMaterial(row pgx.Row) (*models.Material, error) {
	var m models.Material
	err := row.Scan(
		&m.ID, &m.SubjectID, &m.Category, &m.Title, &m.Description, &m.CreatedBy,
		&m.CreatedAt, &m.UpdatedAt, &m.DeletedAt,
		&m.SubjectName, &m.AuthorName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (r *pgMaterialRepository) FindByID(ctx context.Context, id string) (*models.Material, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+materialSelectColumns+materialFrom+` WHERE m.id = $1 AND m.deleted_at IS NULL`, id)
	return scanMaterial(row)
}

func (r *pgMaterialRepository) ListBySubject(ctx context.Context, subjectID string) ([]*models.Material, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+materialSelectColumns+materialFrom+`
		 WHERE m.subject_id = $1 AND m.deleted_at IS NULL
		 ORDER BY m.category, m.created_at DESC`,
		subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMaterials(rows)
}

func collectMaterials(rows pgx.Rows) ([]*models.Material, error) {
	var result []*models.Material
	for rows.Next() {
		m, err := scanMaterial(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *pgMaterialRepository) ListSubjectSubscriberUserIDs(ctx context.Context, subjectID string) ([]string, error) {
	// Подписчики предмета — активные студенты групп, у которых предмет есть
	// в активных шаблонах расписания (раздел 21: просмотр — студенты группы,
	// которым читается предмет).
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT s.user_id
		 FROM students s
		 JOIN schedule_templates st ON st.group_id = s.group_id
		 WHERE st.subject_id = $1 AND st.deleted_at IS NULL
		   AND st.status = 'active'
		   AND s.deleted_at IS NULL AND s.status = 'active'
		   AND s.user_id IS NOT NULL`,
		subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *pgMaterialRepository) Create(ctx context.Context, m *models.Material) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO materials (subject_id, category, title, description, created_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		m.SubjectID, m.Category, m.Title, m.Description, m.CreatedBy,
	).Scan(&id)
	return id, err
}

func (r *pgMaterialRepository) Update(ctx context.Context, m *models.Material) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE materials SET subject_id = $1, category = $2, title = $3, description = $4, updated_at = now()
		 WHERE id = $5 AND deleted_at IS NULL`,
		m.SubjectID, m.Category, m.Title, m.Description, m.ID,
	)
	return err
}

func (r *pgMaterialRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE materials SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

const materialFileColumns = `
	id, material_id, file_type, storage_key, external_url,
	file_name, size_bytes, mime_type, uploaded_by, created_at
`

func scanMaterialFile(row pgx.Row) (*models.MaterialFile, error) {
	var f models.MaterialFile
	err := row.Scan(
		&f.ID, &f.MaterialID, &f.FileType, &f.StorageKey, &f.ExternalURL,
		&f.FileName, &f.SizeBytes, &f.MimeType, &f.UploadedBy, &f.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *pgMaterialRepository) FindFileByID(ctx context.Context, id string) (*models.MaterialFile, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+materialFileColumns+` FROM material_files WHERE id = $1`, id)
	return scanMaterialFile(row)
}

func (r *pgMaterialRepository) ListFilesByMaterial(ctx context.Context, materialID string) ([]*models.MaterialFile, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+materialFileColumns+` FROM material_files WHERE material_id = $1 ORDER BY created_at`,
		materialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.MaterialFile
	for rows.Next() {
		f, err := scanMaterialFile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

func (r *pgMaterialRepository) CreateFile(ctx context.Context, f *models.MaterialFile) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO material_files
		 (material_id, file_type, storage_key, external_url, file_name, size_bytes, mime_type, uploaded_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		f.MaterialID, f.FileType, f.StorageKey, f.ExternalURL,
		f.FileName, f.SizeBytes, f.MimeType, f.UploadedBy,
	).Scan(&id)
	return id, err
}

func (r *pgMaterialRepository) DeleteFile(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM material_files WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

