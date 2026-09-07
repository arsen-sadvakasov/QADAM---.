package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// SubjectRepository описывает доступ к таблице subjects.
type SubjectRepository interface {
	FindByID(ctx context.Context, id string) (*models.Subject, error)
	List(ctx context.Context) ([]*models.Subject, error)
	Create(ctx context.Context, s *models.Subject) (string, error)
	Update(ctx context.Context, s *models.Subject) error
	SoftDelete(ctx context.Context, id string) error
}

type pgSubjectRepository struct {
	pool *pgxpool.Pool
}

// NewSubjectRepository создаёт реализацию SubjectRepository на базе pgx.
func NewSubjectRepository(pool *pgxpool.Pool) SubjectRepository {
	return &pgSubjectRepository{pool: pool}
}

const subjectSelectColumns = `id, college_id, name, code, created_at, updated_at, deleted_at`

func scanSubject(row pgx.Row) (*models.Subject, error) {
	var s models.Subject
	err := row.Scan(&s.ID, &s.CollegeID, &s.Name, &s.Code, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *pgSubjectRepository) FindByID(ctx context.Context, id string) (*models.Subject, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+subjectSelectColumns+` FROM subjects WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanSubject(row)
}

func (r *pgSubjectRepository) List(ctx context.Context) ([]*models.Subject, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+subjectSelectColumns+` FROM subjects WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Subject
	for rows.Next() {
		s, err := scanSubject(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *pgSubjectRepository) Create(ctx context.Context, s *models.Subject) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO subjects (college_id, name, code) VALUES ($1, $2, $3) RETURNING id`,
		s.CollegeID, s.Name, s.Code,
	).Scan(&id)
	return id, err
}

func (r *pgSubjectRepository) Update(ctx context.Context, s *models.Subject) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE subjects SET name = $1, code = $2, updated_at = now() WHERE id = $3 AND deleted_at IS NULL`,
		s.Name, s.Code, s.ID,
	)
	return err
}

func (r *pgSubjectRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE subjects SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}
