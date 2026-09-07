package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// GroupRepository описывает доступ к таблице groups.
type GroupRepository interface {
	FindByID(ctx context.Context, id string) (*models.Group, error)
	List(ctx context.Context) ([]*models.Group, error)
	ListByCurator(ctx context.Context, curatorID string) ([]*models.Group, error)
	Create(ctx context.Context, g *models.Group) (string, error)
	Update(ctx context.Context, g *models.Group) error
	SoftDelete(ctx context.Context, id string) error
}

type pgGroupRepository struct {
	pool *pgxpool.Pool
}

// NewGroupRepository создаёт реализацию GroupRepository на базе pgx.
func NewGroupRepository(pool *pgxpool.Pool) GroupRepository {
	return &pgGroupRepository{pool: pool}
}

const groupSelectColumns = `id, college_id, specialty_id, course_id, curator_id, name, created_at, updated_at, deleted_at`

func scanGroup(row pgx.Row) (*models.Group, error) {
	var g models.Group
	err := row.Scan(
		&g.ID, &g.CollegeID, &g.SpecialtyID, &g.CourseID, &g.CuratorID, &g.Name,
		&g.CreatedAt, &g.UpdatedAt, &g.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}

func (r *pgGroupRepository) FindByID(ctx context.Context, id string) (*models.Group, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+groupSelectColumns+` FROM groups WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanGroup(row)
}

func (r *pgGroupRepository) List(ctx context.Context) ([]*models.Group, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+groupSelectColumns+` FROM groups WHERE deleted_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectGroups(rows)
}

func (r *pgGroupRepository) ListByCurator(ctx context.Context, curatorID string) ([]*models.Group, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+groupSelectColumns+` FROM groups WHERE curator_id = $1 AND deleted_at IS NULL ORDER BY name`,
		curatorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectGroups(rows)
}

func collectGroups(rows pgx.Rows) ([]*models.Group, error) {
	var result []*models.Group
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

func (r *pgGroupRepository) Create(ctx context.Context, g *models.Group) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO groups (college_id, specialty_id, course_id, curator_id, name)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		g.CollegeID, g.SpecialtyID, g.CourseID, g.CuratorID, g.Name,
	).Scan(&id)
	return id, err
}

func (r *pgGroupRepository) Update(ctx context.Context, g *models.Group) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE groups SET specialty_id = $1, course_id = $2, curator_id = $3, name = $4, updated_at = now()
		 WHERE id = $5 AND deleted_at IS NULL`,
		g.SpecialtyID, g.CourseID, g.CuratorID, g.Name, g.ID,
	)
	return err
}

func (r *pgGroupRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE groups SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}
