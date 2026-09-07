-- Phase 8 (Notifications): notifications + notification_recipients
-- (разделы 19, 34.1 спецификации). Уведомление разворачивается на
-- конкретных получателей (fan-out) — каждый получатель имеет собственный
-- статус прочтения.

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(30) NOT NULL
        CHECK (type IN ('schedule_change', 'replacement', 'cancellation', 'new_material', 'system', 'session')),
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'normal'
        CHECK (priority IN ('low', 'normal', 'high', 'critical')),
    related_entity_type VARCHAR(50),
    related_entity_id UUID,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_related ON notifications(related_entity_type, related_entity_id)
    WHERE related_entity_id IS NOT NULL;

CREATE TABLE notification_recipients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Быстрая выборка непрочитанных уведомлений пользователя (раздел 34.2).
CREATE INDEX idx_notification_recipients_user_unread
    ON notification_recipients(user_id, is_read);

-- Защита от дублирования: один пользователь — одна запись на уведомление.
CREATE UNIQUE INDEX uniq_notification_recipient
    ON notification_recipients(notification_id, user_id);

-- Прочитанное уведомление не может иметь пустой read_at.
ALTER TABLE notification_recipients
    ADD CONSTRAINT chk_read_at_present CHECK (NOT is_read OR read_at IS NOT NULL);

COMMENT ON TABLE notifications IS 'Уведомления: замены расписания, новые материалы, системные, сессия (Phase 8)';
COMMENT ON TABLE notification_recipients IS 'Разворачивание уведомлений на конкретных пользователей с индивидуальным статусом прочтения';
