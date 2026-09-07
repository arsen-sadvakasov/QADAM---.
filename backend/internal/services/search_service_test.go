package services

import (
	"context"
	"errors"
	"testing"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// fakeSearchRepository — in-memory реализация
// repositories.SearchRepository для unit-тестов.
type fakeSearchRepository struct {
	students []repositories.SearchResult
	teachers []repositories.SearchResult
	curators []repositories.SearchResult
	groups   []repositories.SearchResult
	subjects []repositories.SearchResult
	rooms    []repositories.SearchResult
	lastQuery string
}

func (f *fakeSearchRepository) SearchStudents(_ context.Context, q string, limit int) ([]repositories.SearchResult, error) {
	f.lastQuery = q
	return f.students, nil
}

func (f *fakeSearchRepository) SearchTeachers(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return f.teachers, nil
}

func (f *fakeSearchRepository) SearchCurators(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return f.curators, nil
}

func (f *fakeSearchRepository) SearchGroups(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return f.groups, nil
}

func (f *fakeSearchRepository) SearchSubjects(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return f.subjects, nil
}

func (f *fakeSearchRepository) SearchRooms(_ context.Context, _ string, _ int) ([]repositories.SearchResult, error) {
	return f.rooms, nil
}

func newSearchTestService() (*SearchService, *fakeSearchRepository) {
	repo := &fakeSearchRepository{}
	return NewSearchService(repo), repo
}

func TestSearchService_QueryTooShort(t *testing.T) {
	svc, _ := newSearchTestService()

	_, err := svc.Search(context.Background(), models.RoleAdmin, "а", "")
	if !errors.Is(err, ErrSearchValidation) {
		t.Fatalf("expected validation error for short query, got %v", err)
	}

	_, err = svc.Search(context.Background(), models.RoleAdmin, "  ", "")
	if !errors.Is(err, ErrSearchValidation) {
		t.Fatalf("expected validation error for blank query, got %v", err)
	}
}

func TestSearchService_AdminSeesAllTypes(t *testing.T) {
	svc, repo := newSearchTestService()
	repo.students = []repositories.SearchResult{{Type: repositories.SearchTypeStudent, ID: "s-1", Title: "Айдар С."}}
	repo.teachers = []repositories.SearchResult{{Type: repositories.SearchTypeTeacher, ID: "t-1", Title: "Иванов И.И."}}
	repo.groups = []repositories.SearchResult{{Type: repositories.SearchTypeGroup, ID: "g-1", Title: "ПО-23"}}
	repo.subjects = []repositories.SearchResult{{Type: repositories.SearchTypeSubject, ID: "sub-1", Title: "Базы данных"}}
	repo.rooms = []repositories.SearchResult{{Type: repositories.SearchTypeRoom, ID: "r-1", Title: "301"}}
	repo.curators = []repositories.SearchResult{{Type: repositories.SearchTypeCurator, ID: "c-1", Title: "Куратор К."}}

	results, err := svc.Search(context.Background(), models.RoleAdmin, "поиск", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 6 {
		t.Errorf("expected 6 results for admin, got %d", len(results))
	}
}

func TestSearchService_StudentCannotSearchPeople(t *testing.T) {
	svc, repo := newSearchTestService()
	repo.students = []repositories.SearchResult{{Type: repositories.SearchTypeStudent, ID: "s-1", Title: "Айдар С."}}
	repo.teachers = []repositories.SearchResult{{Type: repositories.SearchTypeTeacher, ID: "t-1", Title: "Иванов И.И."}}
	repo.groups = []repositories.SearchResult{{Type: repositories.SearchTypeGroup, ID: "g-1", Title: "ПО-23"}}
	repo.subjects = []repositories.SearchResult{{Type: repositories.SearchTypeSubject, ID: "sub-1", Title: "Базы данных"}}
	repo.rooms = []repositories.SearchResult{{Type: repositories.SearchTypeRoom, ID: "r-1", Title: "301"}}

	results, err := svc.Search(context.Background(), models.RoleStudent, "поиск", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Только группы/предметы/кабинеты — люди скрыты (раздел 25).
	if len(results) != 3 {
		t.Fatalf("expected 3 results for student (no people), got %d", len(results))
	}
	for _, r := range results {
		if r.Type == repositories.SearchTypeStudent || r.Type == repositories.SearchTypeTeacher || r.Type == repositories.SearchTypeCurator {
			t.Errorf("student must not see people in search, got type %q", r.Type)
		}
	}
}

func TestSearchService_StudentForbiddenType(t *testing.T) {
	svc, _ := newSearchTestService()

	_, err := svc.Search(context.Background(), models.RoleStudent, "поиск", "student")
	if !errors.Is(err, ErrSearchValidation) {
		t.Fatalf("expected validation error for forbidden type, got %v", err)
	}
}

func TestSearchService_TypeFilter(t *testing.T) {
	svc, repo := newSearchTestService()
	repo.groups = []repositories.SearchResult{{Type: repositories.SearchTypeGroup, ID: "g-1", Title: "ПО-23"}}
	repo.subjects = []repositories.SearchResult{{Type: repositories.SearchTypeSubject, ID: "sub-1", Title: "Базы данных"}}

	results, err := svc.Search(context.Background(), models.RoleAdmin, "поиск", "group")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Type != repositories.SearchTypeGroup {
		t.Errorf("expected only groups, got %+v", results)
	}
}

func TestSearchService_UnknownRoleEmptyResult(t *testing.T) {
	svc, _ := newSearchTestService()

	results, err := svc.Search(context.Background(), "hacker", "поиск", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty result for unknown role, got %d", len(results))
	}
}

// Тест экранирования LIKE-паттерна живёт в пакете repositories
// (см. repositories/search_repository_test.go).
