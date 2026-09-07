package services

import (
	"context"
	"errors"
	"testing"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// fakeStudentRepository — in-memory реализация
// repositories.StudentRepository для unit-тестов.
type fakeStudentRepository struct {
	byID   map[string]*models.Student
	nextID int
}

func newFakeStudentRepository() *fakeStudentRepository {
	return &fakeStudentRepository{byID: make(map[string]*models.Student)}
}

func (f *fakeStudentRepository) FindByID(_ context.Context, id string) (*models.Student, error) {
	if s, ok := f.byID[id]; ok {
		return s, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeStudentRepository) ListByGroup(_ context.Context, groupID string) ([]*models.Student, error) {
	var result []*models.Student
	for _, s := range f.byID {
		if s.GroupID == groupID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (f *fakeStudentRepository) List(_ context.Context, groupID, status string) ([]*models.Student, error) {
	var result []*models.Student
	for _, s := range f.byID {
		if groupID != "" && s.GroupID != groupID {
			continue
		}
		if status != "" && string(s.Status) != status {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func (f *fakeStudentRepository) Create(_ context.Context, s *models.Student) (string, error) {
	f.nextID++
	s.ID = string(rune('0' + f.nextID))
	clone := *s
	f.byID[s.ID] = &clone
	return s.ID, nil
}

func (f *fakeStudentRepository) Update(_ context.Context, s *models.Student) error {
	if _, ok := f.byID[s.ID]; !ok {
		return repositories.ErrNotFound
	}
	clone := *s
	f.byID[s.ID] = &clone
	return nil
}

func (f *fakeStudentRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

// fakeGroupRepository — in-memory реализация GroupRepository (только то,
// что нужно CuratorService).
type fakeGroupRepository struct {
	byID      map[string]*models.Group
	byCurator map[string][]string // curatorID -> group IDs
}

func newFakeGroupRepository() *fakeGroupRepository {
	return &fakeGroupRepository{
		byID:      make(map[string]*models.Group),
		byCurator: make(map[string][]string),
	}
}

func (f *fakeGroupRepository) addGroup(id string, curatorID *string) {
	g := &models.Group{ID: id, CuratorID: curatorID}
	f.byID[id] = g
	if curatorID != nil {
		f.byCurator[*curatorID] = append(f.byCurator[*curatorID], id)
	}
}

func (f *fakeGroupRepository) FindByID(_ context.Context, id string) (*models.Group, error) {
	if g, ok := f.byID[id]; ok {
		return g, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeGroupRepository) List(_ context.Context) ([]*models.Group, error) {
	var result []*models.Group
	for _, g := range f.byID {
		result = append(result, g)
	}
	return result, nil
}

func (f *fakeGroupRepository) ListByCurator(_ context.Context, curatorID string) ([]*models.Group, error) {
	var result []*models.Group
	for _, id := range f.byCurator[curatorID] {
		if g, ok := f.byID[id]; ok {
			result = append(result, g)
		}
	}
	return result, nil
}

func (f *fakeGroupRepository) Create(_ context.Context, g *models.Group) (string, error) {
	f.addGroup(g.ID, g.CuratorID)
	return g.ID, nil
}

func (f *fakeGroupRepository) Update(_ context.Context, g *models.Group) error {
	f.byID[g.ID] = g
	return nil
}

func (f *fakeGroupRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

func newCuratorTestService() (*CuratorService, *fakeStudentRepository, *fakeGroupRepository) {
	students := newFakeStudentRepository()
	groups := newFakeGroupRepository()
	curatorID := "curator-1"
	groups.addGroup("group-1", &curatorID) // своя группа куратора
	groups.addGroup("group-2", nil)        // группа без куратора
	return NewCuratorService(students, groups), students, groups
}

func TestCuratorService_CreateStudent_Success(t *testing.T) {
	svc, _, _ := newCuratorTestService()

	student, err := svc.CreateStudent(context.Background(), "curator-1", false, StudentInput{
		GroupID: "group-1",
		Status:  models.StudentStatusActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if student.ID == "" || student.GroupID != "group-1" {
		t.Errorf("unexpected student: %+v", student)
	}
}

func TestCuratorService_CreateStudent_DefaultStatus(t *testing.T) {
	svc, _, _ := newCuratorTestService()

	student, err := svc.CreateStudent(context.Background(), "curator-1", false, StudentInput{
		GroupID: "group-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if student.Status != models.StudentStatusActive {
		t.Errorf("expected default status active, got %q", student.Status)
	}
}

func TestCuratorService_CreateStudent_ForbiddenForeignGroup(t *testing.T) {
	svc, _, _ := newCuratorTestService()

	// Куратор пытается добавить студента в чужую группу (без куратора)
	_, err := svc.CreateStudent(context.Background(), "curator-1", false, StudentInput{
		GroupID: "group-2",
	})
	if !errors.Is(err, ErrCuratorForbidden) {
		t.Fatalf("expected ErrCuratorForbidden, got %v", err)
	}
}

func TestCuratorService_CreateStudent_Validation(t *testing.T) {
	svc, _, _ := newCuratorTestService()
	ctx := context.Background()

	// без group_id
	_, err := svc.CreateStudent(ctx, "curator-1", false, StudentInput{})
	if !errors.Is(err, ErrCuratorValidation) {
		t.Errorf("expected validation error without group_id, got %v", err)
	}

	// неизвестный статус
	_, err = svc.CreateStudent(ctx, "curator-1", false, StudentInput{
		GroupID: "group-1",
		Status:  "bogus",
	})
	if !errors.Is(err, ErrCuratorValidation) {
		t.Errorf("expected validation error for unknown status, got %v", err)
	}
}

func TestCuratorService_GetStudent_ForbiddenForeignGroup(t *testing.T) {
	svc, students, _ := newCuratorTestService()

	// студент в чужой группе
	students.byID["s-2"] = &models.Student{ID: "s-2", GroupID: "group-2", Status: models.StudentStatusActive}

	_, err := svc.GetStudent(context.Background(), "curator-1", false, "s-2")
	if !errors.Is(err, ErrCuratorForbidden) {
		t.Fatalf("expected ErrCuratorForbidden, got %v", err)
	}
}

func TestCuratorService_ListStudents_CuratorOwnGroupsOnly(t *testing.T) {
	svc, students, _ := newCuratorTestService()

	students.byID["s-1"] = &models.Student{ID: "s-1", GroupID: "group-1", Status: models.StudentStatusActive}
	students.byID["s-2"] = &models.Student{ID: "s-2", GroupID: "group-2", Status: models.StudentStatusActive}

	// Без фильтра — только своя группа
	list, err := svc.ListStudents(context.Background(), "curator-1", false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].GroupID != "group-1" {
		t.Errorf("expected only own group students, got %+v", list)
	}

	// Явный фильтр по чужой группе — 403
	_, err = svc.ListStudents(context.Background(), "curator-1", false, "group-2", "")
	if !errors.Is(err, ErrCuratorForbidden) {
		t.Fatalf("expected ErrCuratorForbidden for foreign group filter, got %v", err)
	}
}

func TestCuratorService_ListStudents_AdminSeesAll(t *testing.T) {
	svc, students, _ := newCuratorTestService()

	students.byID["s-1"] = &models.Student{ID: "s-1", GroupID: "group-1", Status: models.StudentStatusActive}
	students.byID["s-2"] = &models.Student{ID: "s-2", GroupID: "group-2", Status: models.StudentStatusExpelled}

	list, err := svc.ListStudents(context.Background(), "admin-1", true, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected all students for admin, got %d", len(list))
	}

	// фильтр по статусу
	expelled, err := svc.ListStudents(context.Background(), "admin-1", true, "", "expelled")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expelled) != 1 || expelled[0].Status != models.StudentStatusExpelled {
		t.Errorf("expected only expelled, got %+v", expelled)
	}
}

func TestCuratorService_UpdateStudent_ChangeStatusAndTransfer(t *testing.T) {
	svc, students, groups := newCuratorTestService()

	curator2 := "curator-2"
	groups.addGroup("group-3", &curator2)
	students.byID["s-1"] = &models.Student{ID: "s-1", GroupID: "group-1", Status: models.StudentStatusActive}
	ctx := context.Background()

	// Смена статуса — своей группы
	updated, err := svc.UpdateStudent(ctx, "curator-1", false, "s-1", StudentInput{
		GroupID: "group-1",
		Status:  models.StudentStatusAcademicLeave,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != models.StudentStatusAcademicLeave {
		t.Errorf("expected academic_leave, got %q", updated.Status)
	}

	// Перевод в чужую группу — 403
	_, err = svc.UpdateStudent(ctx, "curator-1", false, "s-1", StudentInput{
		GroupID: "group-3",
	})
	if !errors.Is(err, ErrCuratorForbidden) {
		t.Fatalf("expected ErrCuratorForbidden for transfer to foreign group, got %v", err)
	}
}

func TestCuratorService_DeleteStudent(t *testing.T) {
	svc, students, _ := newCuratorTestService()
	ctx := context.Background()

	// чужая группа — 403
	students.byID["s-foreign"] = &models.Student{ID: "s-foreign", GroupID: "group-2", Status: models.StudentStatusActive}
	if err := svc.DeleteStudent(ctx, "curator-1", false, "s-foreign"); !errors.Is(err, ErrCuratorForbidden) {
		t.Fatalf("expected ErrCuratorForbidden for foreign group, got %v", err)
	}

	// своя группа — успех
	students.byID["s-own"] = &models.Student{ID: "s-own", GroupID: "group-1", Status: models.StudentStatusActive}
	if err := svc.DeleteStudent(ctx, "curator-1", false, "s-own"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := students.byID["s-own"]; ok {
		t.Error("expected student to be deleted")
	}
}
