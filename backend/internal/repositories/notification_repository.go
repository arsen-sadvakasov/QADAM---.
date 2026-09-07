package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// NotificationRepository описывает доступ к таблицам
// notifications/notification_recipients (Phase 8, раздел 34.1).
type NotificationRepository interface {
	// Create сохраняет уведомление и разворачивает его на получателей
	// (bulk insert в notification_recipients) в одной транзакции.
	Create(ctx context.Context, n *models.Notification, recipientUserIDs []string) (string, error)

	// ListForUser возвращает уведомления пользователя (персональный статус
	// прочтения + содержимое), новые первыми. isRead: nil = все, true/false = фильтр.
	ListForUser(ctx context.Context, userID string, isRead *bool, limit int) ([]*models.UserNotification, error)

	// FindRecipient находит персональную запись получателя.
	FindRecipient(ctx context.Context, notificationID, userID string) (*models.NotificationRecipient, error)

	// MarkRead отмечает уведомление прочитанным.
	MarkRead(ctx context.Context, notificationID, userID string, readAt time.Time) error

	// CountUnread возвращает число непрочитанных уведомлений пользователя.
	CountUnread(ctx context.Context, userID string) (int, error)
}

type pgNotificationRepository struct {
	pool *pgxpool.Pool
}

// NewNotificationRepository создаёт реализацию NotificationRepository на базе pgx.
func NewNotificationRepository(pool *pgxpool.Pool) NotificationRepository {
	return &pgNotificationRepository{pool: pool}
}

func (r *pgNotificationRepository) Create(ctx context.Context, n *models.Notification, recipientUserIDs []string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id string
	err = tx.QueryRow(ctx,
		`INSERT INTO notifications (type, title, body, priority, related_entity_type, related_entity_id, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		n.Type, n.Title, n.Body, n.Priority, n.RelatedEntityType, n.RelatedEntityID, n.CreatedBy,
	).Scan(&id)
	if err != nil {
		return "", err
	}

	// Bulk fan-out на получателей: один запрос на весь список.
	if len(recipientUserIDs) > 0 {
		_, err = tx.Exec(ctx,
			`INSERT INTO notification_recipients (notification_id, user_id)
			 SELECT $1, u FROM unnest($2::uuid[]) AS u
			 ON CONFLICT DO NOTHING`,
			id, recipientUserIDs)
		if err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return id, nil
}

const userNotificationColumns = `
	r.id AS recipient_id,
	n.id AS notification_id,
	n.type, n.title, n.body, n.priority,
	n.related_entity_type, n.related_entity_id, n.created_by, n.created_at,
	r.is_read, r.read_at
`

const userNotificationFrom = `
	FROM notification_recipients r
	JOIN notifications n ON n.id = r.notification_id
`

func scanUserNotification(row pgx.Row) (*models.UserNotification, error) {
	var (
		un        models.UserNotification
		notifID   string
		recType   *string
		recID     *string
		recBy     *string
	)
	err := row.Scan(
		&un.RecipientID, &notifID,
		&un.Type, &un.Title, &un.Body, &un.Priority,
		&recType, &recID, &recBy, &un.CreatedAt,
		&un.IsRead, &un.ReadAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	un.ID = notifID
	un.RelatedEntityType = recType
	un.RelatedEntityID = recID
	un.CreatedBy = recBy
	return &un, nil
}

func (r *pgNotificationRepository) ListForUser(ctx context.Context, userID string, isRead *bool, limit int) ([]*models.UserNotification, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `SELECT ` + userNotificationColumns + userNotificationFrom + ` WHERE r.user_id = $1`
	args := []any{userID}
	if isRead != nil {
		query += ` AND r.is_read = $2`
		args = append(args, *isRead)
	}
	query += ` ORDER BY n.created_at DESC LIMIT $` + itoa(len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.UserNotification
	for rows.Next() {
		un, err := scanUserNotification(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, un)
	}
	return result, rows.Err()
}

func (r *pgNotificationRepository) FindRecipient(ctx context.Context, notificationID, userID string) (*models.NotificationRecipient, error) {
	var rec models.NotificationRecipient
	err := r.pool.QueryRow(ctx,
		`SELECT id, notification_id, user_id, is_read, read_at, created_at
		 FROM notification_recipients
		 WHERE notification_id = $1 AND user_id = $2`,
		notificationID, userID,
	).Scan(&rec.ID, &rec.NotificationID, &rec.UserID, &rec.IsRead, &rec.ReadAt, &rec.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

func (r *pgNotificationRepository) MarkRead(ctx context.Context, notificationID, userID string, readAt time.Time) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE notification_recipients
		 SET is_read = TRUE, read_at = $3
		 WHERE notification_id = $1 AND user_id = $2 AND NOT is_read`,
		notificationID, userID, readAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// либо не найдено, либо уже прочитано — различаем через выборку
		_, err := r.FindRecipient(ctx, notificationID, userID)
		return err
	}
	return nil
}

func (r *pgNotificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notification_recipients WHERE user_id = $1 AND NOT is_read`,
		userID,
	).Scan(&count)
	return count, err
}

// itoa — минимальный конвертер int → string для построения плейсхолдеров.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}
