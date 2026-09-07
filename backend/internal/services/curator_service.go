package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// ErrCuratorValidation возвращается при нарушении правил валидации данных
// студента — HTTP 400.
var ErrCuratorValidation = errors.New("invalid student input")

// ErrCuratorForbidden возвращается, когда куратор пытается работать с
// чужой группой (раздел 17: students.manage — curator «своя группа») —
// HTTP 403.
var ErrCuratorForbidden = errors.New("not allowed to manage this group or student")

// CuratorService реализует бизнес-логику Curator Module (Phase 9):
// управление студентами группы с проверкой прав «куратор — только своя
// группа, admin — все» (разделы 17, 22 спецификации).
type CuratorService struct {
	students repositories.StudentRepository
	groups   repositories.GroupRepository
}

// NewCuratorService создаёт CuratorService с внедрёнными репозиториями.
func NewCuratorService(students repositories.StudentRepository, groups repositories.GroupRepository) *CuratorService {
	return &CuratorService{students: students, groups: groups}
}

// canManageGroup проверяет право пользователя управлять группой:
// admin — любая группа, curator — только группа, где он указан куратором
// (groups.curator_id). Преподаватели и студенты права не имеют.
func (s *CuratorService) canManageGroup(ctx context.Context, userID string, isAdmin bool, groupID string) error {
	if isAdmin {
		return nil
	}
	group, err := s.groups.FindByID(ctx, groupID)
	if err != nil {
		return err
	}
	if group.CuratorID == nil || *group.CuratorID != userID {
		return ErrCuratorForbidden
	}
	return nil
}

// StudentInput — входные данные для создания/редактирования студента.
type StudentInput struct {
	UserID            *string
	GroupID           string
	Status            models.StudentStatus
	AcademicStatusID  *string
	ScholarshipStatus *string
}

// CreateStudent добавляет студента в группу. Curator (своя группа), Admin.
func (s *CuratorService) CreateStudent(ctx context.Context, userID string, isAdmin bool, in StudentInput) (*models.Student, error) {
	if in.GroupID == "" {
		return nil, fmt.Errorf("%w: group_id is required", ErrCuratorValidation)
	}
	if in.Status == "" {
		in.Status = models.StudentStatusActive
	}
	if !isValidStudentStatus(in.Status) {
		return nil, fmt.Errorf("%w: unknown status %q", ErrCuratorValidation, in.Status)
	}

	if err := s.canManageGroup(ctx, userID, isAdmin, in.GroupID); err != nil {
		return nil, err
	}

	student := &models.Student{
		UserID:            in.UserID,
		GroupID:           in.GroupID,
		Status:            in.Status,
		AcademicStatusID:  in.AcademicStatusID,
		ScholarshipStatus: in.ScholarshipStatus,
	}

	id, err := s.students.Create(ctx, student)
	if err != nil {
		return nil, err
	}
	return s.students.FindByID(ctx, id)
}

// GetStudent возвращает профиль студента. Curator (своя группа), Admin.
func (s *CuratorService) GetStudent(ctx context.Context, userID string, isAdmin bool, studentID string) (*models.Student, error) {
	student, err := s.students.FindByID(ctx, studentID)
	if err != nil {
		return nil, err
	}
	if err := s.canManageGroup(ctx, userID, isAdmin, student.GroupID); err != nil {
		return nil, err
	}
	return student, nil
}

// ListStudents возвращает список студентов с фильтрами. Curator — только
// студенты своих групп (фильтр по группе обязателен и проверяется на
// владение), Admin — любые.
func (s *CuratorService) ListStudents(ctx context.Context, userID string, isAdmin bool, groupID, status string) ([]*models.Student, error) {
	if status != "" && !isValidStudentStatus(models.StudentStatus(status)) {
		return nil, fmt.Errorf("%w: unknown status %q", ErrCuratorValidation, status)
	}

	if !isAdmin {
		if groupID == "" {
			// Куратор без фильтра — все его группы; собираем студентов по ним.
			groups, err := s.groups.ListByCurator(ctx, userID)
			if err != nil {
				return nil, err
			}
			var result []*models.Student
			for _, g := range groups {
				students, err := s.students.ListByGroup(ctx, g.ID)
				if err != nil {
					return nil, err
				}
				result = append(result, students...)
			}
			return filterByStatus(result, status), nil
		}
		// Фильтр по конкретной группе — проверяем владение.
		if err := s.canManageGroup(ctx, userID, false, groupID); err != nil {
			return nil, err
		}
	}

	return s.students.List(ctx, groupID, status)
}

// UpdateStudent редактирует студента/меняет статус. Curator (своя группа),
// Admin.
func (s *CuratorService) UpdateStudent(ctx context.Context, userID string, isAdmin bool, studentID string, in StudentInput) (*models.Student, error) {
	existing, err := s.students.FindByID(ctx, studentID)
	if err != nil {
		return nil, err
	}

	// Право проверяется по текущей группе студента И по новой (если перевод).
	if err := s.canManageGroup(ctx, userID, isAdmin, existing.GroupID); err != nil {
		return nil, err
	}

	if in.GroupID != "" && in.GroupID != existing.GroupID {
		if err := s.canManageGroup(ctx, userID, isAdmin, in.GroupID); err != nil {
			return nil, err
		}
		existing.GroupID = in.GroupID
	}
	if in.Status != "" {
		if !isValidStudentStatus(in.Status) {
			return nil, fmt.Errorf("%w: unknown status %q", ErrCuratorValidation, in.Status)
		}
		existing.Status = in.Status
	}
	if in.AcademicStatusID != nil {
		existing.AcademicStatusID = in.AcademicStatusID
	}
	if in.ScholarshipStatus != nil {
		existing.ScholarshipStatus = in.ScholarshipStatus
	}

	if err := s.students.Update(ctx, existing); err != nil {
		return nil, err
	}
	return s.students.FindByID(ctx, studentID)
}

// DeleteStudent удаляет студента (soft delete). Curator (своя группа), Admin.
func (s *CuratorService) DeleteStudent(ctx context.Context, userID string, isAdmin bool, studentID string) error {
	student, err := s.students.FindByID(ctx, studentID)
	if err != nil {
		return err
	}
	if err := s.canManageGroup(ctx, userID, isAdmin, student.GroupID); err != nil {
		return err
	}
	return s.students.SoftDelete(ctx, studentID)
}

// isValidStudentStatus проверяет статус обучения (расширяемый enum,
// раздел 22 спецификации).
func isValidStudentStatus(status models.StudentStatus) bool {
	switch status {
	case models.StudentStatusActive, models.StudentStatusExpelled,
		models.StudentStatusAcademicLeave, models.StudentStatusGraduated:
		return true
	}
	return false
}

// filterByStatus фильтрует студентов по статусу (пустой статус — все).
func filterByStatus(students []*models.Student, status string) []*models.Student {
	if status == "" {
		return students
	}
	var result []*models.Student
	for _, s := range students {
		if string(s.Status) == status {
			result = append(result, s)
		}
	}
	return result
}
