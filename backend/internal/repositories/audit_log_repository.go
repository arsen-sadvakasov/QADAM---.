package repositories

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// AuditLogFilter — фильтры просмотра журнала (раздел 29: по пользователю,
// дате, сущности). Пустые значения = без фильтра.
type AuditLogFilter struct {
	ActorID    string
	EntityType string
	EntityID   string
	Limit      int
}

// AuditLogRepository описывает доступ к таблице audit_logs
// (Phase 14, раздел 29). Только Create и чтение.
type AuditLogRepository interface {
	Create(ctx context.Context, log *models.AuditLog) error
	List(ctx context.Context, filter AuditLogFilter) ([]*models.AuditLog, error)
}

type pgAuditLogRepository struct {
	pool *pgxpool.Pool
}

// NewAuditLogRepository создаёт реализацию AuditLogRepository на базе pgx.
func NewAuditLogRepository(pool *pgxpool.Pool) AuditLogRepository {
	return &pgAuditLogRepository{pool: pool}
}

func (r *pgAuditLogRepository) Create(ctx context.Context, log *models.AuditLog) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, description)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		log.ActorID, log.Action, log.EntityType, log.EntityID, log.Description,
	).Scan(&log.ID, &log.CreatedAt)
}

const auditLogColumns = `
	a.id, a.actor_id, a.action, a.entity_type, a.entity_id, a.description, a.created_at,
	u.full_name
`

const auditLogFrom = `
	FROM audit_logs a
	JOIN users u ON u.id = a.actor_id
`

func scanAuditLog(row pgx.Row) (*models.AuditLog, error) {
	var a models.AuditLog
	err := row.Scan(
		&a.ID, &a.ActorID, &a.Action, &a.EntityType, &a.EntityID, &a.Description, &a.CreatedAt,
		&a.ActorName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *pgAuditLogRepository) List(ctx context.Context, filter AuditLogFilter) ([]*models.AuditLog, error) {
	query := `SELECT ` + auditLogColumns + auditLogFrom + ` WHERE 1=1`
	args := []any{}

	if filter.ActorID != "" {
		args = append(args, filter.ActorID)
		query += fmt.Sprintf(" AND a.actor_id = $%d", len(args))
	}
	if filter.EntityType != "" {
		args = append(args, filter.EntityType)
		query += fmt.Sprintf(" AND a.entity_type = $%d", len(args))
	}
	if filter.EntityID != "" {
		args = append(args, filter.EntityID)
		query += fmt.Sprintf(" AND a.entity_id = $%d", len(args))
	}

	query += " ORDER BY a.created_at DESC"

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	args = append(args, limit)
	query += fmt.Sprintf(" LIMIT $%d", len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.AuditLog
	for rows.Next() {
		a, err := scanAuditLog(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
