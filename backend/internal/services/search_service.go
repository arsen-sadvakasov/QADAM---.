package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// ErrSearchValidation возвращается при некорректных параметрах поиска — HTTP 400.
var ErrSearchValidation = errors.New("invalid search input")

// MinSearchQueryLength — минимальная длина поискового запроса (защита от
// выгрузки всех записей через однобуквенные запросы).
const MinSearchQueryLength = 2

// MaxSearchResultsPerType — максимум результатов на тип сущности.
const MaxSearchResultsPerType = 10

// SearchService реализует глобальный поиск (Phase 11, раздел 25 спецификации):
// ILIKE/pg_trgm по ключевым полям с ограничением результатов по правам —
// студент не должен через поиск обходить ограничения на чувствительные данные.
type SearchService struct {
	search repositories.SearchRepository
}

// NewSearchService создаёт SearchService с внедрённым репозиторием.
func NewSearchService(search repositories.SearchRepository) *SearchService {
	return &SearchService{search: search}
}

// Search выполняет глобальный поиск. q — запрос (мин. 2 символа),
// searchType — фильтр по типу сущности (пусто = все доступные роли).
// Права (разделы 17, 25):
//   - admin: все типы;
//   - curator/teacher: студенты, преподаватели, кураторы, группы, предметы, кабинеты;
//   - student: группы, предметы, кабинеты (без людей — чувствительные данные).
func (s *SearchService) Search(ctx context.Context, role models.RoleKey, q, searchType string) ([]repositories.SearchResult, error) {
	q = strings.TrimSpace(q)
	if len([]rune(q)) < MinSearchQueryLength {
		return nil, fmt.Errorf("%w: query must be at least %d characters", ErrSearchValidation, MinSearchQueryLength)
	}

	allowedTypes := allowedSearchTypes(role)
	if len(allowedTypes) == 0 {
		// Неизвестная роль — пустой результат, а не ошибка.
		return nil, nil
	}

	if searchType != "" {
		t := repositories.SearchResultType(searchType)
		if !allowedTypes[t] {
			return nil, fmt.Errorf("%w: type %q is not allowed for your role", ErrSearchValidation, searchType)
		}
		return s.searchOne(ctx, q, t)
	}

	var result []repositories.SearchResult
	for _, t := range searchTypeOrder {
		if !allowedTypes[t] {
			continue
		}
		found, err := s.searchOne(ctx, q, t)
		if err != nil {
			return nil, err
		}
		result = append(result, found...)
	}
	return result, nil
}

// searchTypeOrder — стабильный порядок обхода типов при поиске по всем.
var searchTypeOrder = []repositories.SearchResultType{
	repositories.SearchTypeStudent,
	repositories.SearchTypeTeacher,
	repositories.SearchTypeCurator,
	repositories.SearchTypeGroup,
	repositories.SearchTypeSubject,
	repositories.SearchTypeRoom,
}

// allowedSearchTypes возвращает набор типов, доступных роли.
func allowedSearchTypes(role models.RoleKey) map[repositories.SearchResultType]bool {
	switch role {
	case models.RoleAdmin, models.RoleCurator, models.RoleTeacher:
		return map[repositories.SearchResultType]bool{
			repositories.SearchTypeStudent: true,
			repositories.SearchTypeTeacher: true,
			repositories.SearchTypeCurator: true,
			repositories.SearchTypeGroup:   true,
			repositories.SearchTypeSubject: true,
			repositories.SearchTypeRoom:    true,
		}
	case models.RoleStudent:
		// Студент ищет только «публичные» сущности: группы, предметы,
		// кабинеты. Люди скрыты (раздел 25: студент не должен обходить
		// ограничения на чувствительные данные).
		return map[repositories.SearchResultType]bool{
			repositories.SearchTypeGroup:   true,
			repositories.SearchTypeSubject: true,
			repositories.SearchTypeRoom:    true,
		}
	default:
		return nil
	}
}

func (s *SearchService) searchOne(ctx context.Context, q string, t repositories.SearchResultType) ([]repositories.SearchResult, error) {
	switch t {
	case repositories.SearchTypeStudent:
		return s.search.SearchStudents(ctx, q, MaxSearchResultsPerType)
	case repositories.SearchTypeTeacher:
		return s.search.SearchTeachers(ctx, q, MaxSearchResultsPerType)
	case repositories.SearchTypeCurator:
		return s.search.SearchCurators(ctx, q, MaxSearchResultsPerType)
	case repositories.SearchTypeGroup:
		return s.search.SearchGroups(ctx, q, MaxSearchResultsPerType)
	case repositories.SearchTypeSubject:
		return s.search.SearchSubjects(ctx, q, MaxSearchResultsPerType)
	case repositories.SearchTypeRoom:
		return s.search.SearchRooms(ctx, q, MaxSearchResultsPerType)
	default:
		return nil, fmt.Errorf("%w: unknown search type %q", ErrSearchValidation, string(t))
	}
}
