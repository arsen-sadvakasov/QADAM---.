// Package repositories содержит слой доступа к данным (PostgreSQL) через
// интерфейсы, чтобы сервисы бизнес-логики оставались тестируемыми и не
// зависели напрямую от конкретной СУБД (принцип Dependency Inversion,
// см. раздел 12 спецификации).
package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// ErrNotFound возвращается, когда запрошенная сущность не найдена в БД.
var ErrNotFound = errors.New("not found")

// UserListFilter — фильтры для GET /api/v1/users (раздел 35 API Plan).
type UserListFilter struct {
	RoleKey *models.RoleKey
	Search  *string // поиск по full_name/username (ILIKE)
}

// UserRepository описывает доступ к таблице users.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	UpdateLastLogin(ctx context.Context, userID string) error

	List(ctx context.Context, filter UserListFilter) ([]*models.User, error)
	Create(ctx context.Context, u *models.User) (string, error)
	Update(ctx context.Context, u *models.User) error
	SoftDelete(ctx context.Context, id string) error
	ExistsByUsername(ctx context.Context, username string) (bool, error)
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

func (r *pgUserRepository) List(ctx context.Context, filter UserListFilter) ([]*models.User, error) {
	query := `
		SELECT ` + userSelectColumns + `
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.deleted_at IS NULL
	`
	args := make([]any, 0, 2)

	if filter.RoleKey != nil {
		args = append(args, *filter.RoleKey)
		query += fmt.Sprintf(" AND r.key = $%d", len(args))
	}
	if filter.Search != nil && *filter.Search != "" {
		args = append(args, "%"+*filter.Search+"%")
		query += fmt.Sprintf(" AND (u.full_name ILIKE $%d OR u.username ILIKE $%d)", len(args), len(args))
	}
	query += " ORDER BY u.full_name"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (r *pgUserRepository) Create(ctx context.Context, u *models.User) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (college_id, email, phone, username, password_hash, full_name, role_id, language, theme_preference, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
		u.CollegeID, u.Email, u.Phone, u.Username, u.PasswordHash, u.FullName, u.RoleID, u.Language, u.ThemePreference, u.IsActive,
	).Scan(&id)
	return id, err
}

func (r *pgUserRepository) Update(ctx context.Context, u *models.User) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET
			email = $1, phone = $2, full_name = $3, role_id = $4,
			language = $5, theme_preference = $6, is_active = $7, updated_at = now()
		 WHERE id = $8 AND deleted_at IS NULL`,
		u.Email, u.Phone, u.FullName, u.RoleID, u.Language, u.ThemePreference, u.IsActive, u.ID,
	)
	return err
}

func (r *pgUserRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET deleted_at = now(), is_active = FALSE WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *pgUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND deleted_at IS NULL)`, username,
	).Scan(&exists)
	return exists, err
}
