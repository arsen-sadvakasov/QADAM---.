package services

import (
	"context"
	"testing"

	"github.com/qadam/backend/internal/models"
	"github.com/qadam/backend/internal/repositories"
)

// fakeAuditLogRepository — in-memory реализация
// repositories.AuditLogRepository для unit-тестов.
type fakeAuditLogRepository struct {
	logs []*models.AuditLog
}

func (f *fakeAuditLogRepository) Create(_ context.Context, log *models.AuditLog) error {
	log.ID = "audit-" + string(rune('0'+len(f.logs)+1))
	f.logs = append(f.logs, log)
	return nil
}

func (f *fakeAuditLogRepository) List(_ context.Context, filter repositories.AuditLogFilter) ([]*models.AuditLog, error) {
	var result []*models.AuditLog
	for _, a := range f.logs {
		if filter.ActorID != "" && a.ActorID != filter.ActorID {
			continue
		}
		if filter.EntityType != "" && a.EntityType != filter.EntityType {
			continue
		}
		if filter.EntityID != "" && (a.EntityID == nil || *a.EntityID != filter.EntityID) {
			continue
		}
		result = append(result, a)
	}
	return result, nil
}

func TestAuditService_Record(t *testing.T) {
	repo := &fakeAuditLogRepository{}
	svc := NewAuditService(repo)

	entityID := "target-1"
	svc.Record(context.Background(), "admin-1", "update", "user", &entityID, "изменены поля: full_name;")

	if len(repo.logs) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(repo.logs))
	}
	a := repo.logs[0]
	if a.ActorID != "admin-1" || a.Action != "update" || a.EntityType != "user" {
		t.Errorf("unexpected audit record: %+v", a)
	}
	if a.EntityID == nil || *a.EntityID != "target-1" {
		t.Errorf("unexpected entity_id: %v", a.EntityID)
	}
}

func TestAuditService_List_Filters(t *testing.T) {
	repo := &fakeAuditLogRepository{}
	svc := NewAuditService(repo)

	id1 := "e-1"
	id2 := "e-2"
	svc.Record(context.Background(), "admin-1", "create", "user", &id1, "A")
	svc.Record(context.Background(), "admin-2", "create", "schedule", &id2, "B")

	// По actor
	byActor, _ := svc.List(context.Background(), repositories.AuditLogFilter{ActorID: "admin-1"})
	if len(byActor) != 1 || byActor[0].ActorID != "admin-1" {
		t.Errorf("expected only admin-1 records, got %+v", byActor)
	}

	// По типу сущности
	byType, _ := svc.List(context.Background(), repositories.AuditLogFilter{EntityType: "schedule"})
	if len(byType) != 1 || byType[0].EntityType != "schedule" {
		t.Errorf("expected only schedule records, got %+v", byType)
	}

	// Без фильтров — все
	all, _ := svc.List(context.Background(), repositories.AuditLogFilter{})
	if len(all) != 2 {
		t.Errorf("expected all records, got %d", len(all))
	}
}
