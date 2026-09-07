// Package repositories содержит слой доступа к данным (PostgreSQL) через
// интерфейсы, чтобы сервисы бизнес-логики оставались тестируемыми и не
// зависели напрямую от конкретной СУБД (принцип Dependency Inversion,
// см. раздел 12 спецификации).
package repositories

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// ErrNotFound возвращается, когда запрошенная сущность не найдена в БД.
var ErrNotFound = errors.New("not found")

// UserRepository описывает доступ к таблице users.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	UpdateLastLogin(ctx context.Context, userID string) error
}

type pgUserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создаёт реализацию UserRepository на базе pgx.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &pgUserRepository{pool: pool}
}

const userSelectColumns = `
	u.id, u.college_id, u.email, u.phone, u.username, u.password_hash,
	u.full_name, u.avatar_url, u.role_id, r.key, u.language, u.theme_preference,
	u.is_active, u.last_login_at, u.created_at, u.updated_at, u.deleted_at
`

func scanUser(row pgx.Row) (*models.User, error) {
	var u models.User
	err := row.Scan(
		&u.ID, &u.CollegeID, &u.Email, &u.Phone, &u.Username, &u.PasswordHash,
		&u.FullName, &u.AvatarURL, &u.RoleID, &u.RoleKey, &u.Language, &u.ThemePreference,
		&u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *pgUserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT ` + userSelectColumns + `
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.username = $1 AND u.deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, username)
	return scanUser(row)
}

func (r *pgUserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT ` + userSelectColumns + `
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`
	row := r.pool.QueryRow(ctx, query, id)
	return scanUser(row)
}

func (r *pgUserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, userID)
	return err
}
