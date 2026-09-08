package services

import (
	"context"
	"errors"
	"regexp"

	"github.com/qadam/backend/internal/locales"
	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// ErrUnknownLanguage возвращается при передаче неподдерживаемого кода
// языка (поддерживаются kz/ru/en — раздел 26 спецификации).
var ErrUnknownLanguage = errors.New("unsupported language (supported: kz, ru, en)")

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

// ErrUsernameInvalid возвращается, когда username не соответствует формату
// (Phase 14, раздел 28: server-side input validation).
var ErrUsernameInvalid = errors.New("username must be 3-50 chars: letters, digits, dot, dash, underscore")

// ErrPasswordTooShort возвращается, когда пароль короче MinPasswordLength.
var ErrPasswordTooShort = errors.New("password must be at least 8 characters")

// MinPasswordLength — минимальная длина пароля (Phase 14, раздел 28).
const MinPasswordLength = 8

// validUsernameRegex — буквы/цифры/точка/тире/подчёркивание, 3-50 символов.
var validUsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,50}$`)

// UpdateUserInput — входные данные для редактирования пользователя
// (PATCH /api/v1/users/{id}). Указатели позволяют различать "поле не
// передано" и "поле сброшено в null/пусто".
type UpdateUserInput struct {
	FullName *string
	Role     *models.RoleKey
	Email    *string
	Phone    *string
	IsActive *bool
	Language *string
}

// UserAdminService реализует бизнес-логику CRUD пользователей для Admin
// Panel (Phase 5 спецификации): резолв роли по ключу, хэширование пароля,
// проверка уникальности username, блокировка (soft delete).
// Phase 14: мутирующие операции журналируются в audit_logs (best-effort).
type UserAdminService struct {
	users repositories.UserRepository
	roles repositories.RoleRepository
	audit *AuditService
}

// NewUserAdminService создаёт UserAdminService с внедрёнными зависимостями.
func NewUserAdminService(users repositories.UserRepository, roles repositories.RoleRepository) *UserAdminService {
	return &UserAdminService{users: users, roles: roles}
}

// SetAudit подключает журнал аудита (опционально; вызывается из main.go).
func (s *UserAdminService) SetAudit(a *AuditService) {
	s.audit = a
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
func (s *UserAdminService) Create(ctx context.Context, actorID string, in CreateUserInput) (*models.User, error) {
	// Phase 14: server-side валидация ввода (раздел 28).
	if !validUsernameRegex.MatchString(in.Username) {
		return nil, ErrUsernameInvalid
	}
	if len(in.Password) < MinPasswordLength {
		return nil, ErrPasswordTooShort
	}

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
		language = string(locales.DefaultLanguage)
	} else if _, err := locales.Parse(language); err != nil {
		return nil, ErrUnknownLanguage
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

	// Audit: создание пользователя (best-effort, раздел 29).
	if s.audit != nil {
		s.audit.Record(ctx, actorID, "create", "user", &id,
			"создан пользователь "+in.Username+" (роль "+string(in.Role)+")")
	}
	return user, nil
}

// Update редактирует существующего пользователя. Только переданные (не-nil)
// поля будут изменены.
func (s *UserAdminService) Update(ctx context.Context, actorID, id string, in UpdateUserInput) (*models.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Для аудита: описание того, какие поля менялись (без чувствительных данных).
	changes := ""

	if in.FullName != nil {
		changes += "full_name; "
		user.FullName = *in.FullName
	}
	if in.Email != nil {
		changes += "email; "
		user.Email = in.Email
	}
	if in.Phone != nil {
		changes += "phone; "
		user.Phone = in.Phone
	}
	if in.IsActive != nil {
		changes += "is_active; "
		user.IsActive = *in.IsActive
	}
	if in.Language != nil {
		if _, err := locales.Parse(*in.Language); err != nil {
			return nil, ErrUnknownLanguage
		}
		changes += "language; "
		user.Language = *in.Language
	}
	if in.Role != nil {
		role, err := s.roles.FindByKey(ctx, *in.Role)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, ErrUnknownRole
			}
			return nil, err
		}
		changes += "role=" + string(role.Key) + "; "
		user.RoleID = role.ID
		user.RoleKey = role.Key
	}

	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	// Audit: редактирование пользователя (best-effort, раздел 29).
	if s.audit != nil && changes != "" {
		s.audit.Record(ctx, actorID, "update", "user", &id,
			"изменены поля: "+changes)
	}
	return user, nil
}

// Block выполняет блокировку/soft-delete пользователя (DELETE /api/v1/users/{id}).
func (s *UserAdminService) Block(ctx context.Context, actorID, id string) error {
	if err := s.users.SoftDelete(ctx, id); err != nil {
		return err
	}
	// Audit: блокировка пользователя (best-effort, раздел 29).
	if s.audit != nil {
		s.audit.Record(ctx, actorID, "block", "user", &id, "пользователь заблокирован")
	}
	return nil
}
