package services

import (
	"context"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// CreateTeacherInput — входные данные для создания преподавателя
// администратором (POST /api/v1/teachers, раздел 35 API Plan). Создаёт
// одновременно учётную запись пользователя (роль teacher) и связанную
// запись в teachers (1:1, раздел 34.1 спецификации).
type CreateTeacherInput struct {
	Username string
	Password string
	FullName string
	Email    *string
	Phone    *string
}

// TeacherAdminService реализует бизнес-логику управления преподавателями
// для Admin Panel (Phase 5 спецификации): оркестрирует создание
// пользователя (роль teacher) и связанной записи в teachers, а также
// назначение преподавателя на предмет/группу (teacher_subjects).
type TeacherAdminService struct {
	users    *UserAdminService
	teachers repositories.TeacherRepository
}

// NewTeacherAdminService создаёт TeacherAdminService с внедрёнными зависимостями.
func NewTeacherAdminService(users *UserAdminService, teachers repositories.TeacherRepository) *TeacherAdminService {
	return &TeacherAdminService{users: users, teachers: teachers}
}

// List возвращает список преподавателей.
func (s *TeacherAdminService) List(ctx context.Context) ([]*models.Teacher, error) {
	return s.teachers.List(ctx)
}

// Get возвращает преподавателя по ID.
func (s *TeacherAdminService) Get(ctx context.Context, id string) (*models.Teacher, error) {
	return s.teachers.FindByID(ctx, id)
}

// Create создаёт учётную запись пользователя с ролью teacher и связанную
// запись в teachers.
func (s *TeacherAdminService) Create(ctx context.Context, actorID string, in CreateTeacherInput) (*models.Teacher, error) {
	user, err := s.users.Create(ctx, actorID, CreateUserInput{
		Username: in.Username,
		Password: in.Password,
		FullName: in.FullName,
		Role:     models.RoleTeacher,
		Email:    in.Email,
		Phone:    in.Phone,
	})
	if err != nil {
		return nil, err
	}

	teacherID, err := s.teachers.Create(ctx, user.ID, user.CollegeID)
	if err != nil {
		return nil, err
	}

	return &models.Teacher{
		ID:        teacherID,
		UserID:    user.ID,
		CollegeID: user.CollegeID,
		FullName:  user.FullName,
		Email:     user.Email,
		Phone:     user.Phone,
		AvatarURL: user.AvatarURL,
	}, nil
}

// AssignSubject назначает преподавателя на предмет в конкретной группе
// (раздел 34.1: teacher_subjects).
func (s *TeacherAdminService) AssignSubject(ctx context.Context, ts models.TeacherSubject) error {
	return s.teachers.AssignSubject(ctx, ts)
}

// UnassignSubject снимает назначение преподавателя с предмета/группы.
func (s *TeacherAdminService) UnassignSubject(ctx context.Context, ts models.TeacherSubject) error {
	return s.teachers.UnassignSubject(ctx, ts)
}
