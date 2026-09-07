package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// RoomRepository описывает доступ к таблице rooms.
type RoomRepository interface {
	FindByID(ctx context.Context, id string) (*models.Room, error)
	List(ctx context.Context) ([]*models.Room, error)
	Create(ctx context.Context, r *models.Room) (string, error)
	Update(ctx context.Context, r *models.Room) error
	SoftDelete(ctx context.Context, id string) error
}

type pgRoomRepository struct {
	pool *pgxpool.Pool
}

// NewRoomRepository создаёт реализацию RoomRepository на базе pgx.
func NewRoomRepository(pool *pgxpool.Pool) RoomRepository {
	return &pgRoomRepository{pool: pool}
}

const roomSelectColumns = `id, college_id, number, name, type, floor, building, notes, created_at, updated_at, deleted_at`

func scanRoom(row pgx.Row) (*models.Room, error) {
	var rm models.Room
	err := row.Scan(
		&rm.ID, &rm.CollegeID, &rm.Number, &rm.Name, &rm.Type, &rm.Floor, &rm.Building, &rm.Notes,
		&rm.CreatedAt, &rm.UpdatedAt, &rm.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rm, nil
}

func (r *pgRoomRepository) FindByID(ctx context.Context, id string) (*models.Room, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+roomSelectColumns+` FROM rooms WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanRoom(row)
}

func (r *pgRoomRepository) List(ctx context.Context) ([]*models.Room, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+roomSelectColumns+` FROM rooms WHERE deleted_at IS NULL ORDER BY building, number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Room
	for rows.Next() {
		rm, err := scanRoom(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rm)
	}
	return result, rows.Err()
}

func (r *pgRoomRepository) Create(ctx context.Context, rm *models.Room) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO rooms (college_id, number, name, type, floor, building, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		rm.CollegeID, rm.Number, rm.Name, rm.Type, rm.Floor, rm.Building, rm.Notes,
	).Scan(&id)
	return id, err
}

func (r *pgRoomRepository) Update(ctx context.Context, rm *models.Room) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE rooms SET number = $1, name = $2, type = $3, floor = $4, building = $5, notes = $6, updated_at = now()
		 WHERE id = $7 AND deleted_at IS NULL`,
		rm.Number, rm.Name, rm.Type, rm.Floor, rm.Building, rm.Notes, rm.ID,
	)
	return err
}

func (r *pgRoomRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE rooms SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}
