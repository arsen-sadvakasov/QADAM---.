package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// RoleRepository описывает доступ к таблице roles. Используется Admin
// Panel (Phase 5) для резолва RoleKey (`student`/`teacher`/...), приходящего
// в запросах создания/редактирования пользователя, в roles.id для FK.
type RoleRepository interface {
	FindByKey(ctx context.Context, key models.RoleKey) (*models.Role, error)
	List(ctx context.Context) ([]*models.Role, error)
}

type pgRoleRepository struct {
	pool *pgxpool.Pool
}

// NewRoleRepository создаёт реализацию RoleRepository на базе pgx.
func NewRoleRepository(pool *pgxpool.Pool) RoleRepository {
	return &pgRoleRepository{pool: pool}
}

func scanRole(row pgx.Row) (*models.Role, error) {
	var r models.Role
	if err := row.Scan(&r.ID, &r.Key, &r.Name); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (r *pgRoleRepository) FindByKey(ctx context.Context, key models.RoleKey) (*models.Role, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, key, name FROM roles WHERE key = $1`, key)
	return scanRole(row)
}

func (r *pgRoleRepository) List(ctx context.Context) ([]*models.Role, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, key, name FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, role)
	}
	return result, rows.Err()
}
