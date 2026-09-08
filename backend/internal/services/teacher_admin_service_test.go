package services

import (
	"context"
	"errors"
	"testing"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// fakeTeacherRepository — in-memory реализация
// repositories.TeacherRepository для unit-тестов TeacherAdminService.
type fakeTeacherRepository struct {
	byID     map[string]*models.Teacher
	subjects []models.TeacherSubject
}

func newFakeTeacherRepository() *fakeTeacherRepository {
	return &fakeTeacherRepository{byID: make(map[string]*models.Teacher)}
}

func (f *fakeTeacherRepository) FindByID(_ context.Context, id string) (*models.Teacher, error) {
	if t, ok := f.byID[id]; ok {
		return t, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeTeacherRepository) List(_ context.Context) ([]*models.Teacher, error) {
	var result []*models.Teacher
	for _, t := range f.byID {
		result = append(result, t)
	}
	return result, nil
}

func (f *fakeTeacherRepository) Create(_ context.Context, userID string, collegeID *string) (string, error) {
	id := "teacher-" + userID
	t := &models.Teacher{ID: id, UserID: userID, CollegeID: collegeID}
	f.byID[id] = t
	return id, nil
}

func (f *fakeTeacherRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

func (f *fakeTeacherRepository) ListSubjectsByTeacher(_ context.Context, teacherID string) ([]models.TeacherSubject, error) {
	var result []models.TeacherSubject
	for _, ts := range f.subjects {
		if ts.TeacherID == teacherID {
			result = append(result, ts)
		}
	}
	return result, nil
}

func (f *fakeTeacherRepository) AssignSubject(_ context.Context, ts models.TeacherSubject) error {
	for _, existing := range f.subjects {
		if existing == ts {
			return errors.New("duplicate assignment")
		}
	}
	f.subjects = append(f.subjects, ts)
	return nil
}

func (f *fakeTeacherRepository) UnassignSubject(_ context.Context, ts models.TeacherSubject) error {
	for i, existing := range f.subjects {
		if existing == ts {
			f.subjects = append(f.subjects[:i], f.subjects[i+1:]...)
			return nil
		}
	}
	return repositories.ErrNotFound
}

func newTestTeacherAdminService() (*TeacherAdminService, *fakeTeacherRepository, *fakeAdminUserRepository) {
	users := newFakeAdminUserRepository()
	teachers := newFakeTeacherRepository()
	svc := NewTeacherAdminService(NewUserAdminService(users, fakeAdminRoleRepository{}), teachers)
	return svc, teachers, users
}

func TestTeacherAdminService_Create_Success(t *testing.T) {
	svc, teachers, _ := newTestTeacherAdminService()

	teacher, err := svc.Create(context.Background(), "admin-1", CreateTeacherInput{
		Username: "newteach",
		Password: "secret123",
		FullName: "New Teacher",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if teacher.ID == "" || teacher.UserID == "" {
		t.Errorf("unexpected teacher: %+v", teacher)
	}
	if _, ok := teachers.byID[teacher.ID]; !ok {
		t.Error("expected teacher record created")
	}
}

func TestTeacherAdminService_Create_ValidationPropagates(t *testing.T) {
	svc, _, _ := newTestTeacherAdminService()

	// Неверный username — валидация UserAdminService пробрасывается.
	_, err := svc.Create(context.Background(), "admin-1", CreateTeacherInput{
		Username: "x",
		Password: "secret123",
		FullName: "Bad",
	})
	if !errors.Is(err, ErrUsernameInvalid) {
		t.Fatalf("expected ErrUsernameInvalid, got %v", err)
	}
}

func TestTeacherAdminService_AssignUnassignSubject(t *testing.T) {
	svc, teachers, _ := newTestTeacherAdminService()

	ts := models.TeacherSubject{TeacherID: "teacher-u-1", SubjectID: "sub-1", GroupID: "g-1"}
	if err := svc.AssignSubject(context.Background(), ts); err != nil {
		t.Fatalf("assign failed: %v", err)
	}
	if len(teachers.subjects) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(teachers.subjects))
	}

	if err := svc.UnassignSubject(context.Background(), ts); err != nil {
		t.Fatalf("unassign failed: %v", err)
	}
	if len(teachers.subjects) != 0 {
		t.Errorf("expected 0 assignments after unassign, got %d", len(teachers.subjects))
	}

	// Повторный unassign — 404
	if err := svc.UnassignSubject(context.Background(), ts); !errors.Is(err, repositories.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTeacherAdminService_ListGet(t *testing.T) {
	svc, _, _ := newTestTeacherAdminService()
	ctx := context.Background()

	created, err := svc.Create(ctx, "admin-1", CreateTeacherInput{
		Username: "teach2", Password: "secret123", FullName: "Teacher Two",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 teacher, got %d", len(list))
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected %q, got %q", created.ID, got.ID)
	}

	// Несуществующий — 404
	if _, err := svc.Get(ctx, "missing"); !errors.Is(err, repositories.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
