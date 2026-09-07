package services

import (
	"context"
	"errors"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// Ошибки, возвращаемые UserAdminService — безопасны для отображения
// администратору в сообщении об ошибке.
var (
	ErrUsernameTaken = errors.New("username is already taken")
	ErrUnknownRole   = errors.New("unknown role key")
)

// CreateUserInput — входные данные для создания пользователя администратором
// (раздел 35 API Plan: POST /api/v1/users).
type CreateUserInput struct {
	Username string
	Password string
	FullName string
	Role     models.RoleKey
	Email    *string
	Phone    *string
	Language string
	Theme    string
}

// UpdateUserInput — входные данные для редактирования пользователя
// (PATCH /api/v1/users/{id}). Указатели позволяют различать "поле не
// передано" и "поле сброшено в null/пусто".
type UpdateUserInput struct {
	FullName *string
	Role     *models.RoleKey
	Email    *string
	Phone    *string
	IsActive *bool
}

// UserAdminService реализует бизнес-логику CRUD пользователей для Admin
// Panel (Phase 5 спецификации): резолв роли по ключу, хэширование пароля,
// проверка уникальности username, блокировка (soft delete).
type UserAdminService struct {
	users repositories.UserRepository
	roles repositories.RoleRepository
}

// NewUserAdminService создаёт UserAdminService с внедрёнными зависимостями.
func NewUserAdminService(users repositories.UserRepository, roles repositories.RoleRepository) *UserAdminService {
	return &UserAdminService{users: users, roles: roles}
}

// List возвращает список пользователей с фильтрами (роль, поиск по имени/username).
func (s *UserAdminService) List(ctx context.Context, filter repositories.UserListFilter) ([]*models.User, error) {
	return s.users.List(ctx, filter)
}

// Get возвращает пользователя по ID.
func (s *UserAdminService) Get(ctx context.Context, id string) (*models.User, error) {
	return s.users.FindByID(ctx, id)
}

// Create создаёт нового пользователя с указанной ролью.
func (s *UserAdminService) Create(ctx context.Context, in CreateUserInput) (*models.User, error) {
	role, err := s.roles.FindByKey(ctx, in.Role)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrUnknownRole
		}
		return nil, err
	}

	exists, err := s.users.ExistsByUsername(ctx, in.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	passwordHash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	language := in.Language
	if language == "" {
		language = "ru"
	}
	theme := in.Theme
	if theme == "" {
		theme = "dark"
	}

	user := &models.User{
		Email:           in.Email,
		Phone:           in.Phone,
		Username:        in.Username,
		PasswordHash:    passwordHash,
		FullName:        in.FullName,
		RoleID:          role.ID,
		RoleKey:         role.Key,
		Language:        language,
		ThemePreference: theme,
		IsActive:        true,
	}

	id, err := s.users.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	user.ID = id
	return user, nil
}

// Update редактирует существующего пользователя. Только переданные (не-nil)
// поля будут изменены.
func (s *UserAdminService) Update(ctx context.Context, id string, in UpdateUserInput) (*models.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.FullName != nil {
		user.FullName = *in.FullName
	}
	if in.Email != nil {
		user.Email = in.Email
	}
	if in.Phone != nil {
		user.Phone = in.Phone
	}
	if in.IsActive != nil {
		user.IsActive = *in.IsActive
	}
	if in.Role != nil {
		role, err := s.roles.FindByKey(ctx, *in.Role)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrUnknownRole
			}
			return nil, err
		}
		user.RoleID = role.ID
		user.RoleKey = role.Key
	}

	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Block выполняет блокировку/soft-delete пользователя (DELETE /api/v1/users/{id}).
func (s *UserAdminService) Block(ctx context.Context, id string) error {
	return s.users.SoftDelete(ctx, id)
}
