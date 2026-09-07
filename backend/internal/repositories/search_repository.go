package repositories

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SearchResultType — тип сущности в результатах глобального поиска
// (раздел 25 спецификации).
type SearchResultType string

const (
	SearchTypeStudent  SearchResultType = "student"
	SearchTypeTeacher  SearchResultType = "teacher"
	SearchTypeCurator  SearchResultType = "curator"
	SearchTypeGroup    SearchResultType = "group"
	SearchTypeSubject  SearchResultType = "subject"
	SearchTypeRoom     SearchResultType = "room"
)

// SearchResult — одна запись результата глобального поиска.
type SearchResult struct {
	Type        SearchResultType `json:"type"`
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Subtitle    *string          `json:"subtitle,omitempty"`
	// GroupID заполняется для студентов — чтобы куратор/преподаватель мог
	// перейти к группе; для студента-пользователя не отдаётся.
	GroupID     *string          `json:"group_id,omitempty"`
}

// SearchRepository описывает глобальный поиск по ключевым полям сущностей
// (Phase 11, раздел 25 спецификации: ILIKE/pg_trgm с ограничением по правам).
type SearchRepository interface {
	SearchStudents(ctx context.Context, q string, limit int) ([]SearchResult, error)
	SearchTeachers(ctx context.Context, q string, limit int) ([]SearchResult, error)
	SearchCurators(ctx context.Context, q string, limit int) ([]SearchResult, error)
	SearchGroups(ctx context.Context, q string, limit int) ([]SearchResult, error)
	SearchSubjects(ctx context.Context, q string, limit int) ([]SearchResult, error)
	SearchRooms(ctx context.Context, q string, limit int) ([]SearchResult, error)
}

type pgSearchRepository struct {
	pool *pgxpool.Pool
}

// NewSearchRepository создаёт реализацию SearchRepository на базе pgx.
func NewSearchRepository(pool *pgxpool.Pool) SearchRepository {
	return &pgSearchRepository{pool: pool}
}

// likePattern экранирует пользовательский ввод для ILIKE-паттерна
// (%, _ и \ — спецсимволы LIKE).
func likePattern(q string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_")
	return "%" + replacer.Replace(q) + "%"
}

func (r *pgSearchRepository) SearchStudents(ctx context.Context, q string, limit int) ([]SearchResult, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.id, u.full_name, s.group_id
		 FROM students s
		 LEFT JOIN users u ON u.id = s.user_id
		 WHERE s.deleted_at IS NULL AND u.full_name ILIKE $1
		 ORDER BY u.full_name
		 LIMIT $2`,
		likePattern(q), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SearchResult
	for rows.Next() {
		var sr SearchResult
		if err := rows.Scan(&sr.ID, &sr.Title, &sr.GroupID); err != nil {
			return nil, err
		}
		sr.Type = SearchTypeStudent
		result = append(result, sr)
	}
	return result, rows.Err()
}

func (r *pgSearchRepository) SearchTeachers(ctx context.Context, q string, limit int) ([]SearchResult, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, u.full_name
		 FROM teachers t
		 JOIN users u ON u.id = t.user_id
		 WHERE t.deleted_at IS NULL AND u.full_name ILIKE $1
		 ORDER BY u.full_name
		 LIMIT $2`,
		likePattern(q), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SearchResult
	for rows.Next() {
		var sr SearchResult
		if err := rows.Scan(&sr.ID, &sr.Title); err != nil {
			return nil, err
		}
		sr.Type = SearchTypeTeacher
		result = append(result, sr)
	}
	return result, rows.Err()
}

func (r *pgSearchRepository) SearchCurators(ctx context.Context, q string, limit int) ([]SearchResult, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT u.id, u.full_name
		 FROM users u
		 JOIN groups g ON g.curator_id = u.id
		 WHERE u.full_name ILIKE $1
		 ORDER BY u.full_name
		 LIMIT $2`,
		likePattern(q), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SearchResult
	for rows.Next() {
		var sr SearchResult
		if err := rows.Scan(&sr.ID, &sr.Title); err != nil {
			return nil, err
		}
		sr.Type = SearchTypeCurator
		result = append(result, sr)
	}
	return result, rows.Err()
}

func (r *pgSearchRepository) SearchGroups(ctx context.Context, q string, limit int) ([]SearchResult, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT g.id, g.name
		 FROM groups g
		 WHERE g.deleted_at IS NULL AND g.name ILIKE $1
		 ORDER BY g.name
		 LIMIT $2`,
		likePattern(q), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SearchResult
	for rows.Next() {
		var sr SearchResult
		if err := rows.Scan(&sr.ID, &sr.Title); err != nil {
			return nil, err
		}
		sr.Type = SearchTypeGroup
		result = append(result, sr)
	}
	return result, rows.Err()
}

func (r *pgSearchRepository) SearchSubjects(ctx context.Context, q string, limit int) ([]SearchResult, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT sub.id, sub.name, sub.code
		 FROM subjects sub
		 WHERE sub.deleted_at IS NULL AND sub.name ILIKE $1
		 ORDER BY sub.name
		 LIMIT $2`,
		likePattern(q), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SearchResult
	for rows.Next() {
		var sr SearchResult
		if err := rows.Scan(&sr.ID, &sr.Title, &sr.Subtitle); err != nil {
			return nil, err
		}
		sr.Type = SearchTypeSubject
		result = append(result, sr)
	}
	return result, rows.Err()
}

func (r *pgSearchRepository) SearchRooms(ctx context.Context, q string, limit int) ([]SearchResult, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.number, r.name
		 FROM rooms r
		 WHERE r.deleted_at IS NULL AND (r.number ILIKE $1 OR r.name ILIKE $1)
		 ORDER BY r.number
		 LIMIT $2`,
		likePattern(q), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SearchResult
	for rows.Next() {
		var sr SearchResult
		if err := rows.Scan(&sr.ID, &sr.Title, &sr.Subtitle); err != nil {
			return nil, err
		}
		sr.Type = SearchTypeRoom
		result = append(result, sr)
	}
	return result, rows.Err()
}

