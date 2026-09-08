package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// fakeAdminUserRepository — in-memory реализация UserRepository
// для unit-тестов UserAdminService.
type fakeAdminUserRepository struct {
	byID       map[string]*models.User
	byUsername map[string]*models.User
	nextID     int
}

func newFakeAdminUserRepository() *fakeAdminUserRepository {
	return &fakeAdminUserRepository{
		byID:       make(map[string]*models.User),
		byUsername: make(map[string]*models.User),
	}
}

func (f *fakeAdminUserRepository) genID() string {
	f.nextID++
	return "u-" + string(rune('0'+f.nextID))
}

func (f *fakeAdminUserRepository) Create(_ context.Context, u *models.User) (string, error) {
	u.ID = f.genID()
	clone := *u
	f.byID[u.ID] = &clone
	f.byUsername[u.Username] = &clone
	return u.ID, nil
}

func (f *fakeAdminUserRepository) FindByID(_ context.Context, id string) (*models.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeAdminUserRepository) FindByUsername(_ context.Context, username string) (*models.User, error) {
	if u, ok := f.byUsername[username]; ok {
		return u, nil
	}
	return nil, repositories.ErrNotFound
}

func (f *fakeAdminUserRepository) ExistsByUsername(_ context.Context, username string) (bool, error) {
	_, exists := f.byUsername[username]
	return exists, nil
}

func (f *fakeAdminUserRepository) Update(_ context.Context, u *models.User) error {
	if _, ok := f.byID[u.ID]; !ok {
		return repositories.ErrNotFound
	}
	clone := *u
	f.byID[u.ID] = &clone
	f.byUsername[u.Username] = &clone
	return nil
}

func (f *fakeAdminUserRepository) SoftDelete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

func (f *fakeAdminUserRepository) UpdateLastLogin(_ context.Context, userID string) error {
	if _, ok := f.byID[userID]; !ok {
		return repositories.ErrNotFound
	}
	return nil
}

func (f *fakeAdminUserRepository) List(_ context.Context, filter repositories.UserListFilter) ([]*models.User, error) {
	var result []*models.User
	for _, u := range f.byID {
		if filter.RoleKey != nil && u.RoleKey != *filter.RoleKey {
			continue
		}
		if filter.Search != nil && !strings.Contains(u.FullName, *filter.Search) {
			continue
		}
		result = append(result, u)
	}
	return result, nil
}

// fakeAdminRoleRepository — резолвит роль по ключу.
type fakeAdminRoleRepository struct{}

func (fakeAdminRoleRepository) FindByKey(_ context.Context, key models.RoleKey) (*models.Role, error) {
	known := map[models.RoleKey]string{
		models.RoleStudent: "role-student",
		models.RoleTeacher: "role-teacher",
		models.RoleCurator: "role-curator",
		models.RoleAdmin:   "role-admin",
	}
	if id, ok := known[key]; ok {
		return &models.Role{ID: id, Key: key, Name: string(key)}, nil
	}
	return nil, repositories.ErrNotFound
}

func (fakeAdminRoleRepository) List(_ context.Context) ([]*models.Role, error) {
	return nil, nil
}

func newTestUserAdminService() (*UserAdminService, *fakeAdminUserRepository) {
	users := newFakeAdminUserRepository()
	svc := NewUserAdminService(users, fakeAdminRoleRepository{})
	return svc, users
}

func TestUserAdminService_Create_Success(t *testing.T) {
	svc, _ := newTestUserAdminService()

	user, err := svc.Create(context.Background(), "admin-1", CreateUserInput{
		Username: "newuser",
		Password: "secret123",
		FullName: "New User",
		Role:     models.RoleStudent,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID == "" || user.RoleKey != models.RoleStudent {
		t.Errorf("unexpected user: %+v", user)
	}
	// Пароль хэшируется, не хранится открытым текстом (раздел 28).
	if user.PasswordHash == "secret123" || user.PasswordHash == "" {
		t.Error("expected password to be hashed")
	}
}

func TestUserAdminService_Create_Defaults(t *testing.T) {
	svc, _ := newTestUserAdminService()

	user, err := svc.Create(context.Background(), "admin-1", CreateUserInput{
		Username: "newuser",
		Password: "secret123",
		FullName: "New User",
		Role:     models.RoleStudent,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Language != "ru" {
		t.Errorf("expected default language ru, got %q", user.Language)
	}
	if user.ThemePreference != "dark" {
		t.Errorf("expected default theme dark, got %q", user.ThemePreference)
	}
	if !user.IsActive {
		t.Error("expected new user to be active")
	}
}

func TestUserAdminService_Create_Validation(t *testing.T) {
	svc, _ := newTestUserAdminService()
	ctx := context.Background()

	// username короче 3 символов
	_, err := svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "ab", Password: "secret123", FullName: "X", Role: models.RoleStudent,
	})
	if !errors.Is(err, ErrUsernameInvalid) {
		t.Errorf("expected ErrUsernameInvalid for short username, got %v", err)
	}

	// username с недопустимыми символами
	_, err = svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "bad user!", Password: "secret123", FullName: "X", Role: models.RoleStudent,
	})
	if !errors.Is(err, ErrUsernameInvalid) {
		t.Errorf("expected ErrUsernameInvalid for invalid chars, got %v", err)
	}

	// пароль короче 8
	_, err = svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "newuser", Password: "1234567", FullName: "X", Role: models.RoleStudent,
	})
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}

	// неизвестный язык
	_, err = svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "newuser", Password: "secret123", FullName: "X", Role: models.RoleStudent, Language: "de",
	})
	if !errors.Is(err, ErrUnknownLanguage) {
		t.Errorf("expected ErrUnknownLanguage, got %v", err)
	}

	// неизвестная роль
	_, err = svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "newuser", Password: "secret123", FullName: "X", Role: "bogus",
	})
	if !errors.Is(err, ErrUnknownRole) {
		t.Errorf("expected ErrUnknownRole, got %v", err)
	}
}

func TestUserAdminService_Create_DuplicateUsername(t *testing.T) {
	svc, _ := newTestUserAdminService()
	ctx := context.Background()

	_, err := svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "dup", Password: "secret123", FullName: "First", Role: models.RoleStudent,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "dup", Password: "secret123", FullName: "Second", Role: models.RoleStudent,
	})
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}

func TestUserAdminService_Update(t *testing.T) {
	svc, users := newTestUserAdminService()
	ctx := context.Background()

	created, err := svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "target", Password: "secret123", FullName: "Target", Role: models.RoleStudent,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newRole := models.RoleTeacher
	updated, err := svc.Update(ctx, "admin-1", created.ID, UpdateUserInput{
		FullName: &[]string{"Updated Name"}[0],
		Role:     &newRole,
		Language: &[]string{"en"}[0],
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.FullName != "Updated Name" || updated.RoleKey != models.RoleTeacher || updated.Language != "en" {
		t.Errorf("unexpected updated user: %+v", updated)
	}
	if users.byID[created.ID].FullName != "Updated Name" {
		t.Error("expected update persisted")
	}

	// невалидный язык — 400
	_, err = svc.Update(ctx, "admin-1", created.ID, UpdateUserInput{
		Language: &[]string{"de"}[0],
	})
	if !errors.Is(err, ErrUnknownLanguage) {
		t.Errorf("expected ErrUnknownLanguage, got %v", err)
	}

	// несуществующий пользователь — 404
	_, err = svc.Update(ctx, "admin-1", "missing", UpdateUserInput{})
	if !errors.Is(err, repositories.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserAdminService_Block(t *testing.T) {
	svc, users := newTestUserAdminService()
	ctx := context.Background()

	created, _ := svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "victim", Password: "secret123", FullName: "V", Role: models.RoleStudent,
	})

	if err := svc.Block(ctx, "admin-1", created.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := users.byID[created.ID]; ok {
		t.Error("expected user removed after block")
	}
}

func TestUserAdminService_List_Filters(t *testing.T) {
	svc, _ := newTestUserAdminService()
	ctx := context.Background()

	_, err := svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "stud1", Password: "secret123", FullName: "Alice", Role: models.RoleStudent,
	})
	if err != nil {
		t.Fatalf("failed to create stud1: %v", err)
	}
	_, err = svc.Create(ctx, "admin-1", CreateUserInput{
		Username: "teach1", Password: "secret123", FullName: "Bob", Role: models.RoleTeacher,
	})
	if err != nil {
		t.Fatalf("failed to create teach1: %v", err)
	}

	teacher := models.RoleTeacher
	list, err := svc.List(ctx, repositories.UserListFilter{RoleKey: &teacher})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].RoleKey != models.RoleTeacher {
		t.Errorf("expected only teacher, got %+v", list)
	}

	search := "Alice"
	list, err = svc.List(ctx, repositories.UserListFilter{Search: &search})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 || list[0].FullName != "Alice" {
		t.Errorf("expected only Alice, got %+v", list)
	}
}
