-- Phase 14 (Security Hardening): audit_logs — журнал действий
-- администраторов (раздел 29 спецификации). Запись создаётся backend-слоем
-- при мутирующих операциях; через API — только чтение с фильтрами.

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,          -- create/update/delete/block/restore
    entity_type VARCHAR(50) NOT NULL,     -- user/group/schedule/schedule_change/material/notification...
    entity_id UUID,                       -- ID затронутой сущности (если есть)
    description TEXT NOT NULL,            -- человекочитаемое "было → стало"
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Фильтры просмотра: по пользователю, дате, сущности (раздел 29).
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

COMMENT ON TABLE audit_logs IS 'Журнал аудита мутаций администраторов (Phase 14, раздел 29). Только чтение через API.';
