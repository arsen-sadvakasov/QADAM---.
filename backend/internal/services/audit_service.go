package services

import (
	"context"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// AuditService — журнал аудита мутаций администраторов (Phase 14,
// раздел 29 спецификации). Запись создаётся сервисами при мутациях;
// чтение — только Admin с фильтрами.
type AuditService struct {
	audit repositories.AuditLogRepository
}

// NewAuditService создаёт AuditService с внедрённым репозиторием.
func NewAuditService(audit repositories.AuditLogRepository) *AuditService {
	return &AuditService{audit: audit}
}

// Record записывает действие в журнал. Ошибка записи не должна ломать
// основную операцию — вызывающий код игнорирует результат (журналирование
// best-effort), но ошибки логируются middleware Logging.
func (s *AuditService) Record(ctx context.Context, actorID, action, entityType string, entityID *string, description string) {
	_ = s.audit.Create(ctx, &models.AuditLog{
		ActorID:     actorID,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Description: description,
	})
}

// List возвращает журнал с фильтрами (GET /api/v1/admin/audit-logs, Admin).
func (s *AuditService) List(ctx context.Context, filter repositories.AuditLogFilter) ([]*models.AuditLog, error) {
	return s.audit.List(ctx, filter)
}
