package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// StudentRepository описывает доступ к таблице students.
type StudentRepository interface {
	FindByID(ctx context.Context, id string) (*models.Student, error)
	ListByGroup(ctx context.Context, groupID string) ([]*models.Student, error)
	// List возвращает студентов с фильтрами (Phase 9 — Curator Module).
	// Пустые groupID/status означают «без фильтра».
	List(ctx context.Context, groupID, status string) ([]*models.Student, error)
	Create(ctx context.Context, s *models.Student) (string, error)
	Update(ctx context.Context, s *models.Student) error
	SoftDelete(ctx context.Context, id string) error
}

type pgStudentRepository struct {
	pool *pgxpool.Pool
}

// NewStudentRepository создаёт реализацию StudentRepository на базе pgx.
func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &pgStudentRepository{pool: pool}
}

// studentSelectColumns подгружает поля пользователя джойном (LEFT JOIN,
// т.к. у студента может не быть учётной записи — раздел 34.1: "студент
// может существовать до создания login-аккаунта").
const studentSelectColumns = `
	s.id, s.user_id, s.group_id, s.status, s.academic_status_id, s.scholarship_status,
	s.created_at, s.updated_at, s.deleted_at,
	u.full_name, u.email, u.phone, u.avatar_url
`

func scanStudent(row pgx.Row) (*models.Student, error) {
	var s models.Student
	err := row.Scan(
		&s.ID, &s.UserID, &s.GroupID, &s.Status, &s.AcademicStatusID, &s.ScholarshipStatus,
		&s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
		&s.FullName, &s.Email, &s.Phone, &s.AvatarURL,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *pgStudentRepository) FindByID(ctx context.Context, id string) (*models.Student, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+studentSelectColumns+`
		 FROM students s LEFT JOIN users u ON u.id = s.user_id
		 WHERE s.id = $1 AND s.deleted_at IS NULL`,
		id,
	)
	return scanStudent(row)
}

func (r *pgStudentRepository) ListByGroup(ctx context.Context, groupID string) ([]*models.Student, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+studentSelectColumns+`
		 FROM students s LEFT JOIN users u ON u.id = s.user_id
		 WHERE s.group_id = $1 AND s.deleted_at IS NULL
		 ORDER BY u.full_name`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Student
	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *pgStudentRepository) List(ctx context.Context, groupID, status string) ([]*models.Student, error) {
	query := `SELECT ` + studentSelectColumns + `
		 FROM students s LEFT JOIN users u ON u.id = s.user_id
		 WHERE s.deleted_at IS NULL`
	args := []any{}
	if groupID != "" {
		args = append(args, groupID)
		query += fmt.Sprintf(" AND s.group_id = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND s.status = $%d", len(args))
	}
	query += " ORDER BY u.full_name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Student
	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *pgStudentRepository) Create(ctx context.Context, s *models.Student) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO students (user_id, group_id, status, academic_status_id, scholarship_status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		s.UserID, s.GroupID, s.Status, s.AcademicStatusID, s.ScholarshipStatus,
	).Scan(&id)
	return id, err
}

func (r *pgStudentRepository) Update(ctx context.Context, s *models.Student) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE students SET group_id = $1, status = $2, academic_status_id = $3, scholarship_status = $4, updated_at = now()
		 WHERE id = $5 AND deleted_at IS NULL`,
		s.GroupID, s.Status, s.AcademicStatusID, s.ScholarshipStatus, s.ID,
	)
	return err
}

func (r *pgStudentRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE students SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}
