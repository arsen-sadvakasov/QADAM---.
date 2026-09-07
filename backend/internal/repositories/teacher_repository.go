package repositories

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qadam/backend/internal/models"
)

// TeacherRepository описывает доступ к таблице teachers (и связанной
// таблице teacher_subjects).
type TeacherRepository interface {
	FindByID(ctx context.Context, id string) (*models.Teacher, error)
	List(ctx context.Context) ([]*models.Teacher, error)
	Create(ctx context.Context, userID string, collegeID *string) (string, error)
	SoftDelete(ctx context.Context, id string) error

	AssignSubject(ctx context.Context, ts models.TeacherSubject) error
	UnassignSubject(ctx context.Context, ts models.TeacherSubject) error
	ListSubjectsByTeacher(ctx context.Context, teacherID string) ([]models.TeacherSubject, error)
}

type pgTeacherRepository struct {
	pool *pgxpool.Pool
}

// NewTeacherRepository создаёт реализацию TeacherRepository на базе pgx.
func NewTeacherRepository(pool *pgxpool.Pool) TeacherRepository {
	return &pgTeacherRepository{pool: pool}
}

// teacherSelectColumns подгружает поля пользователя джойном, чтобы API
// сразу мог вернуть имя/контакты преподавателя без дополнительного запроса.
const teacherSelectColumns = `
	t.id, t.user_id, t.college_id, t.created_at, t.updated_at, t.deleted_at,
	u.full_name, u.email, u.phone, u.avatar_url
`

func scanTeacher(row pgx.Row) (*models.Teacher, error) {
	var t models.Teacher
	err := row.Scan(
		&t.ID, &t.UserID, &t.CollegeID, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt,
		&t.FullName, &t.Email, &t.Phone, &t.AvatarURL,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *pgTeacherRepository) FindByID(ctx context.Context, id string) (*models.Teacher, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+teacherSelectColumns+`
		 FROM teachers t JOIN users u ON u.id = t.user_id
		 WHERE t.id = $1 AND t.deleted_at IS NULL`,
		id,
	)
	return scanTeacher(row)
}

func (r *pgTeacherRepository) List(ctx context.Context) ([]*models.Teacher, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+teacherSelectColumns+`
		 FROM teachers t JOIN users u ON u.id = t.user_id
		 WHERE t.deleted_at IS NULL ORDER BY u.full_name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Teacher
	for rows.Next() {
		t, err := scanTeacher(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *pgTeacherRepository) Create(ctx context.Context, userID string, collegeID *string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`INSERT INTO teachers (user_id, college_id) VALUES ($1, $2) RETURNING id`,
		userID, collegeID,
	).Scan(&id)
	return id, err
}

func (r *pgTeacherRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE teachers SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *pgTeacherRepository) AssignSubject(ctx context.Context, ts models.TeacherSubject) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO teacher_subjects (teacher_id, subject_id, group_id) VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		ts.TeacherID, ts.SubjectID, ts.GroupID,
	)
	return err
}

func (r *pgTeacherRepository) UnassignSubject(ctx context.Context, ts models.TeacherSubject) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM teacher_subjects WHERE teacher_id = $1 AND subject_id = $2 AND group_id = $3`,
		ts.TeacherID, ts.SubjectID, ts.GroupID,
	)
	return err
}

func (r *pgTeacherRepository) ListSubjectsByTeacher(ctx context.Context, teacherID string) ([]models.TeacherSubject, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT teacher_id, subject_id, group_id FROM teacher_subjects WHERE teacher_id = $1`,
		teacherID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.TeacherSubject
	for rows.Next() {
		var ts models.TeacherSubject
		if err := rows.Scan(&ts.TeacherID, &ts.SubjectID, &ts.GroupID); err != nil {
			return nil, err
		}
		result = append(result, ts)
	}
	return result, rows.Err()
}
