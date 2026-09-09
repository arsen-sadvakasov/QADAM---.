package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// ScheduleChangeRepository описывает доступ к таблице schedule_changes
// (Phase 6 — Schedule Changes, раздел 34.1 спецификации).
type ScheduleChangeRepository interface {
	FindByID(ctx context.Context, id string) (*models.ScheduleChange, error)
	FindByTemplateAndDate(ctx context.Context, templateID string, date time.Time) (*models.ScheduleChange, error)
	ListByDateRange(ctx context.Context, from, to time.Time) ([]*models.ScheduleChangeWithNames, error)
	ListByGroupAndDateRange(ctx context.Context, groupID string, from, to time.Time) ([]*models.ScheduleChangeWithNames, error)
	ListByTeacherAndDateRange(ctx context.Context, teacherID string, from, to time.Time) ([]*models.ScheduleChangeWithNames, error)
	Create(ctx context.Context, c *models.ScheduleChange) (string, error)
	Update(ctx context.Context, c *models.ScheduleChange) error
	Delete(ctx context.Context, id string) error
}

type pgScheduleChangeRepository struct {
	pool *pgxpool.Pool
}

// NewScheduleChangeRepository создаёт реализацию ScheduleChangeRepository
// на базе pgx.
func NewScheduleChangeRepository(pool *pgxpool.Pool) ScheduleChangeRepository {
	return &pgScheduleChangeRepository{pool: pool}
}

const scheduleChangeColumns = `
	sc.id, sc.schedule_template_id, sc.change_date, sc.change_type,
	sc.new_teacher_id, sc.new_room_id, sc.new_start_time, sc.new_end_time, sc.new_date,
	sc.reason, sc.created_by, sc.created_at, sc.updated_at
`

const scheduleChangeWithNamesColumns = scheduleChangeColumns + `,
	u1.full_name,
	r1.number,
	st.start_time, st.end_time,
	sub.name, g.name,
	u2.full_name,
	r2.number
`

const scheduleChangeWithNamesJoins = `
	FROM schedule_changes sc
	JOIN schedule_templates st ON st.id = sc.schedule_template_id
	JOIN teachers t1 ON t1.id = st.teacher_id
	JOIN users u1 ON u1.id = t1.user_id
	JOIN rooms r1 ON r1.id = st.room_id
	JOIN subjects sub ON sub.id = st.subject_id
	JOIN groups g ON g.id = st.group_id
	LEFT JOIN teachers t2 ON t2.id = sc.new_teacher_id
	LEFT JOIN users u2 ON u2.id = t2.user_id
	LEFT JOIN rooms r2 ON r2.id = sc.new_room_id
`

func scanScheduleChange(row pgx.Row) (*models.ScheduleChange, error) {
	var c models.ScheduleChange
	err := row.Scan(
		&c.ID, &c.ScheduleTemplateID, &c.ChangeDate, &c.ChangeType,
		&c.NewTeacherID, &c.NewRoomID, &c.NewStartTime, &c.NewEndTime, &c.NewDate,
		&c.Reason, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

// scanScheduleChangeWithNames сканирует строку, дополненную джойнами имён
// ("было → стало"). Порядок колонок — scheduleChangeWithNamesColumns.
func scanScheduleChangeWithNames(row pgx.Row) (*models.ScheduleChangeWithNames, error) {
	var (
		c         models.ScheduleChangeWithNames
		newTeach  *string
		newRoom   *string
	)
	err := row.Scan(
		&c.ID, &c.ScheduleTemplateID, &c.ChangeDate, &c.ChangeType,
		&c.NewTeacherID, &c.NewRoomID, &c.NewStartTime, &c.NewEndTime, &c.NewDate,
		&c.Reason, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		&c.OriginalTeacherName, &c.OriginalRoomNumber, &c.OriginalStartTime, &c.OriginalEndTime,
		&c.SubjectName, &c.GroupName,
		&newTeach, &newRoom,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.NewTeacherName = newTeach
	c.NewRoomNumber = newRoom
	return &c, nil
}

func (r *pgScheduleChangeRepository) FindByID(ctx context.Context, id string) (*models.ScheduleChange, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+scheduleChangeColumns+` FROM schedule_changes sc WHERE sc.id = $1`, id)
	return scanScheduleChange(row)
}

func (r *pgScheduleChangeRepository) FindByTemplateAndDate(ctx context.Context, templateID string, date time.Time) (*models.ScheduleChange, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+scheduleChangeColumns+` FROM schedule_changes sc
		 WHERE sc.schedule_template_id = $1 AND sc.change_date = $2`,
		templateID, date)
	return scanScheduleChange(row)
}

func (r *pgScheduleChangeRepository) ListByDateRange(ctx context.Context, from, to time.Time) ([]*models.ScheduleChangeWithNames, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+scheduleChangeWithNamesColumns+scheduleChangeWithNamesJoins+`
		 WHERE sc.change_date >= $1 AND sc.change_date <= $2
		 ORDER BY sc.change_date, st.start_time`,
		from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectScheduleChangesWithNames(rows)
}

func (r *pgScheduleChangeRepository) ListByGroupAndDateRange(ctx context.Context, groupID string, from, to time.Time) ([]*models.ScheduleChangeWithNames, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+scheduleChangeWithNamesColumns+scheduleChangeWithNamesJoins+`
		 WHERE sc.change_date >= $2 AND sc.change_date <= $3 AND st.group_id = $1
		 ORDER BY sc.change_date, st.start_time`,
		groupID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectScheduleChangesWithNames(rows)
}

func (r *pgScheduleChangeRepository) ListByTeacherAndDateRange(ctx context.Context, teacherID string, from, to time.Time) ([]*models.ScheduleChangeWithNames, error) {
	// Замена затрагивает преподавателя, если он вёл занятие изначально
	// (st.teacher_id) или назначен новым преподавателем (sc.new_teacher_id).
	rows, err := r.pool.Query(ctx,
		`SELECT `+scheduleChangeWithNamesColumns+scheduleChangeWithNamesJoins+`
		 WHERE sc.change_date >= $2 AND sc.change_date <= $3
		   AND (st.teacher_id = $1 OR sc.new_teacher_id = $1)
		 ORDER BY sc.change_date, st.start_time`,
		teacherID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectScheduleChangesWithNames(rows)
}

func collectScheduleChangesWithNames(rows pgx.Rows) ([]*models.ScheduleChangeWithNames, error) {
	var result []*models.ScheduleChangeWithNames
	for rows.Next() {
		c, err := scanScheduleChangeWithNames(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *pgScheduleChangeRepository) Create(ctx context.Context, c *models.ScheduleChange) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO schedule_changes
		 (schedule_template_id, change_date, change_type,
		  new_teacher_id, new_room_id, new_start_time, new_end_time, new_date,
		  reason, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id`,
		c.ScheduleTemplateID, c.ChangeDate, c.ChangeType,
		c.NewTeacherID, c.NewRoomID, c.NewStartTime, c.NewEndTime, c.NewDate,
		c.Reason, c.CreatedBy,
	).Scan(&id)
	return id, err
}

func (r *pgScheduleChangeRepository) Update(ctx context.Context, c *models.ScheduleChange) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE schedule_changes SET
			change_type = $1,
			new_teacher_id = $2, new_room_id = $3,
			new_start_time = $4, new_end_time = $5, new_date = $6,
			reason = $7, updated_at = now()
		 WHERE id = $8`,
		c.ChangeType,
		c.NewTeacherID, c.NewRoomID,
		c.NewStartTime, c.NewEndTime, c.NewDate,
		c.Reason, c.ID,
	)
	return err
}

func (r *pgScheduleChangeRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM schedule_changes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
