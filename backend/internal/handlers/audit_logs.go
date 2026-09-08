package handlers

import (
	"net/http"
	"time"

	"github.com/qadam/backend/internal/repositories"
	"github.com/qadam/backend/internal/services"
)

// AuditLogsHandler содержит HTTP-хендлер журнала аудита
// (GET /api/v1/admin/audit-logs, разделы 29 и 35 спецификации). Admin only.
type AuditLogsHandler struct {
	audit *services.AuditService
}

// NewAuditLogsHandler создаёт AuditLogsHandler с внедрённым сервисом.
func NewAuditLogsHandler(audit *services.AuditService) *AuditLogsHandler {
	return &AuditLogsHandler{audit: audit}
}

type auditLogDTO struct {
	ID          string  `json:"id"`
	ActorID     string  `json:"actor_id"`
	ActorName   string  `json:"actor_name"`
	Action      string  `json:"action"`
	EntityType  string  `json:"entity_type"`
	EntityID    *string `json:"entity_id"`
	Description string  `json:"description"`
	CreatedAt   string  `json:"created_at"`
}

// List обрабатывает GET /api/v1/admin/audit-logs?actor_id=&entity_type=&entity_id=&limit=
func (h *AuditLogsHandler) List(w http.ResponseWriter, r *http.Request) {
	logs, err := h.audit.List(r.Context(), repositories.AuditLogFilter{
		ActorID:    r.URL.Query().Get("actor_id"),
		EntityType: r.URL.Query().Get("entity_type"),
		EntityID:   r.URL.Query().Get("entity_id"),
		Limit:      parsePositiveInt(r.URL.Query().Get("limit")),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	dtos := make([]auditLogDTO, 0, len(logs))
	for _, a := range logs {
		dtos = append(dtos, auditLogDTO{
			ID:          a.ID,
			ActorID:     a.ActorID,
			ActorName:   a.ActorName,
			Action:      a.Action,
			EntityType:  a.EntityType,
			EntityID:    a.EntityID,
			Description: a.Description,
			CreatedAt:   a.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit_logs": dtos})
}

func parsePositiveInt(s string) int {
	if s == "" {
		return 0
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
		if n > 1_000_000 {
			return 1_000_000
		}
	}
	return n
}

