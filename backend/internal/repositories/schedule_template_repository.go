package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// ScheduleTemplateRepository описывает доступ к таблице schedule_templates.
type ScheduleTemplateRepository interface {
	FindByID(ctx context.Context, id string) (*models.ScheduleTemplate, error)
	ListByGroup(ctx context.Context, groupID string) ([]*models.ScheduleTemplate, error)
	ListByTeacher(ctx context.Context, teacherID string) ([]*models.ScheduleTemplate, error)
	ListByRoom(ctx context.Context, roomID string) ([]*models.ScheduleTemplate, error)
	Create(ctx context.Context, t *models.ScheduleTemplate) (string, error)
	Update(ctx context.Context, t *models.ScheduleTemplate) error
	SoftDelete(ctx context.Context, id string) error
}

type pgScheduleTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewScheduleTemplateRepository создаёт реализацию ScheduleTemplateRepository
// на базе pgx.
func NewScheduleTemplateRepository(pool *pgxpool.Pool) ScheduleTemplateRepository {
	return &pgScheduleTemplateRepository{pool: pool}
}

// scheduleTemplateSelectColumns подгружает связанные названия джойнами,
// чтобы карточка занятия (FR-3 спецификации) собиралась одним запросом.
const scheduleTemplateSelectColumns = `
	st.id, st.group_id, st.subject_id, st.teacher_id, st.room_id,
	st.day_of_week, st.start_time, st.end_time, st.lesson_type, st.week_parity,
	st.valid_from, st.valid_to, st.status, st.created_at, st.updated_at, st.deleted_at,
	sub.name, u.full_name, r.number, g.name
`

const scheduleTemplateJoins = `
	FROM schedule_templates st
	JOIN subjects sub ON sub.id = st.subject_id
	JOIN teachers t ON t.id = st.teacher_id
	JOIN users u ON u.id = t.user_id
	JOIN rooms r ON r.id = st.room_id
	JOIN groups g ON g.id = st.group_id
`

func scanScheduleTemplate(row pgx.Row) (*models.ScheduleTemplate, error) {
	var st models.ScheduleTemplate
	err := row.Scan(
		&st.ID, &st.GroupID, &st.SubjectID, &st.TeacherID, &st.RoomID,
		&st.DayOfWeek, &st.StartTime, &st.EndTime, &st.LessonType, &st.WeekParity,
		&st.ValidFrom, &st.ValidTo, &st.Status, &st.CreatedAt, &st.UpdatedAt, &st.DeletedAt,
		&st.SubjectName, &st.TeacherName, &st.RoomNumber, &st.GroupName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &st, nil
}

func (r *pgScheduleTemplateRepository) FindByID(ctx context.Context, id string) (*models.ScheduleTemplate, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+scheduleTemplateSelectColumns+scheduleTemplateJoins+`
		 WHERE st.id = $1 AND st.deleted_at IS NULL`,
		id,
	)
	return scanScheduleTemplate(row)
}

func (r *pgScheduleTemplateRepository) ListByGroup(ctx context.Context, groupID string) ([]*models.ScheduleTemplate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+scheduleTemplateSelectColumns+scheduleTemplateJoins+`
		 WHERE st.group_id = $1 AND st.deleted_at IS NULL AND st.status = 'active'
		 ORDER BY st.day_of_week, st.start_time`,
		groupID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectScheduleTemplates(rows)
}

func (r *pgScheduleTemplateRepository) ListByTeacher(ctx context.Context, teacherID string) ([]*models.ScheduleTemplate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+scheduleTemplateSelectColumns+scheduleTemplateJoins+`
		 WHERE st.teacher_id = $1 AND st.deleted_at IS NULL AND st.status = 'active'
		 ORDER BY st.day_of_week, st.start_time`,
		teacherID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectScheduleTemplates(rows)
}

func (r *pgScheduleTemplateRepository) ListByRoom(ctx context.Context, roomID string) ([]*models.ScheduleTemplate, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+scheduleTemplateSelectColumns+scheduleTemplateJoins+`
		 WHERE st.room_id = $1 AND st.deleted_at IS NULL AND st.status = 'active'
		 ORDER BY st.day_of_week, st.start_time`,
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectScheduleTemplates(rows)
}

func collectScheduleTemplates(rows pgx.Rows) ([]*models.ScheduleTemplate, error) {
	var result []*models.ScheduleTemplate
	for rows.Next() {
		st, err := scanScheduleTemplate(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, st)
	}
	return result, rows.Err()
}

func (r *pgScheduleTemplateRepository) Create(ctx context.Context, t *models.ScheduleTemplate) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO schedule_templates
		 (group_id, subject_id, teacher_id, room_id, day_of_week, start_time, end_time,
		  lesson_type, week_parity, valid_from, valid_to, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING id`,
		t.GroupID, t.SubjectID, t.TeacherID, t.RoomID, t.DayOfWeek, t.StartTime, t.EndTime,
		t.LessonType, t.WeekParity, t.ValidFrom, t.ValidTo, t.Status,
	).Scan(&id)
	return id, err
}

func (r *pgScheduleTemplateRepository) Update(ctx context.Context, t *models.ScheduleTemplate) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE schedule_templates SET
			group_id = $1, subject_id = $2, teacher_id = $3, room_id = $4,
			day_of_week = $5, start_time = $6, end_time = $7,
			lesson_type = $8, week_parity = $9, valid_from = $10, valid_to = $11,
			status = $12, updated_at = now()
		 WHERE id = $13 AND deleted_at IS NULL`,
		t.GroupID, t.SubjectID, t.TeacherID, t.RoomID, t.DayOfWeek, t.StartTime, t.EndTime,
		t.LessonType, t.WeekParity, t.ValidFrom, t.ValidTo, t.Status, t.ID,
	)
	return err
}

func (r *pgScheduleTemplateRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE schedule_templates SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}
