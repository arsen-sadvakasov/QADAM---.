package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// CourseRepository описывает доступ к таблице courses.
type CourseRepository interface {
	FindByID(ctx context.Context, id string) (*models.Course, error)
	ListBySpecialty(ctx context.Context, specialtyID string) ([]*models.Course, error)
	Create(ctx context.Context, c *models.Course) (string, error)
	Update(ctx context.Context, c *models.Course) error
	SoftDelete(ctx context.Context, id string) error
}

type pgCourseRepository struct {
	pool *pgxpool.Pool
}

// NewCourseRepository создаёт реализацию CourseRepository на базе pgx.
func NewCourseRepository(pool *pgxpool.Pool) CourseRepository {
	return &pgCourseRepository{pool: pool}
}

const courseSelectColumns = `id, specialty_id, year_number, created_at, updated_at, deleted_at`

func scanCourse(row pgx.Row) (*models.Course, error) {
	var c models.Course
	err := row.Scan(&c.ID, &c.SpecialtyID, &c.YearNumber, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *pgCourseRepository) FindByID(ctx context.Context, id string) (*models.Course, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+courseSelectColumns+` FROM courses WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanCourse(row)
}

func (r *pgCourseRepository) ListBySpecialty(ctx context.Context, specialtyID string) ([]*models.Course, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+courseSelectColumns+` FROM courses WHERE specialty_id = $1 AND deleted_at IS NULL ORDER BY year_number`,
		specialtyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *pgCourseRepository) Create(ctx context.Context, c *models.Course) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO courses (specialty_id, year_number) VALUES ($1, $2) RETURNING id`,
		c.SpecialtyID, c.YearNumber,
	).Scan(&id)
	return id, err
}

func (r *pgCourseRepository) Update(ctx context.Context, c *models.Course) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE courses SET year_number = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL`,
		c.YearNumber, c.ID,
	)
	return err
}

func (r *pgCourseRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE courses SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}
