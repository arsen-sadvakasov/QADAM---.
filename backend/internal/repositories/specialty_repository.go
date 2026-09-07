package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// SpecialtyRepository описывает доступ к таблице specialties.
type SpecialtyRepository interface {
	FindByID(ctx context.Context, id string) (*models.Specialty, error)
	List(ctx context.Context) ([]*models.Specialty, error)
	Create(ctx context.Context, s *models.Specialty) (string, error)
	Update(ctx context.Context, s *models.Specialty) error
	SoftDelete(ctx context.Context, id string) error
}

type pgSpecialtyRepository struct {
	pool *pgxpool.Pool
}

// NewSpecialtyRepository создаёт реализацию SpecialtyRepository на базе pgx.
func NewSpecialtyRepository(pool *pgxpool.Pool) SpecialtyRepository {
	return &pgSpecialtyRepository{pool: pool}
}

const specialtySelectColumns = `id, college_id, name, code, created_at, updated_at, deleted_at`

func scanSpecialty(row pgx.Row) (*models.Specialty, error) {
	var s models.Specialty
	err := row.Scan(&s.ID, &s.CollegeID, &s.Name, &s.Code, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *pgSpecialtyRepository) FindByID(ctx context.Context, id string) (*models.Specialty, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+specialtySelectColumns+` FROM specialties WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanSpecialty(row)
}

func (r *pgSpecialtyRepository) List(ctx context.Context) ([]*models.Specialty, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+specialtySelectColumns+` FROM specialties WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Specialty
	for rows.Next() {
		s, err := scanSpecialty(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *pgSpecialtyRepository) Create(ctx context.Context, s *models.Specialty) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO specialties (college_id, name, code) VALUES ($1, $2, $3) RETURNING id`,
		s.CollegeID, s.Name, s.Code,
	).Scan(&id)
	return id, err
}

func (r *pgSpecialtyRepository) Update(ctx context.Context, s *models.Specialty) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE specialties SET name = $1, code = $2, updated_at = now() WHERE id = $3 AND deleted_at IS NULL`,
		s.Name, s.Code, s.ID,
	)
	return err
}

func (r *pgSpecialtyRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE specialties SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}
