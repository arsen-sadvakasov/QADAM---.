-- 000004_schedule_changes.up.sql
-- Phase 6: Schedule Changes (Замены) — точечные отклонения от шаблона на конкретную дату

CREATE TABLE IF NOT EXISTS schedule_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_template_id UUID NOT NULL REFERENCES schedule_templates(id) ON DELETE CASCADE,
    change_date DATE NOT NULL,
    change_type VARCHAR(50) NOT NULL CHECK (change_type IN ('replace_teacher', 'replace_room', 'reschedule_time', 'cancel', 'move')),
    
    -- Новые значения (nullable, зависят от типа замены)
    new_teacher_id UUID REFERENCES teachers(id) ON DELETE SET NULL,
    new_room_id UUID REFERENCES rooms(id) ON DELETE SET NULL,
    new_start_time TIME,
    new_end_time TIME,
    new_date DATE,
    
    reason TEXT,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индекс для быстрого поиска замен по шаблону и дате
CREATE INDEX idx_schedule_changes_template_date ON schedule_changes(schedule_template_id, change_date);

-- Индекс для поиска замен по дате (для получения всех замен на конкретный день)
CREATE INDEX idx_schedule_changes_date ON schedule_changes(change_date);

-- Индекс для поиска замен конкретного преподавателя
CREATE INDEX idx_schedule_changes_teacher ON schedule_changes(new_teacher_id) WHERE new_teacher_id IS NOT NULL;

-- Индекс для поиска замен по кабинету
CREATE INDEX idx_schedule_changes_room ON schedule_changes(new_room_id) WHERE new_room_id IS NOT NULL;

-- Уникальное ограничение: одна замена на комбинацию шаблона+дата
-- Это предотвращает создание нескольких противоречивых замен для одного занятия в один день
CREATE UNIQUE INDEX uniq_schedule_change_per_template_date ON schedule_changes(schedule_template_id, change_date);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_schedule_changes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_schedule_changes_updated_at
    BEFORE UPDATE ON schedule_changes
    FOR EACH ROW
    EXECUTE FUNCTION update_schedule_changes_updated_at();

-- Комментарии для документации схемы
COMMENT ON TABLE schedule_changes IS 'Точечные замены/отмены/переносы занятий на конкретную дату (Phase 6)';
COMMENT ON COLUMN schedule_changes.change_type IS 'Тип замены: replace_teacher (замена преподавателя), replace_room (замена кабинета), reschedule_time (изменение времени), cancel (отмена), move (перенос на другой день)';
COMMENT ON COLUMN schedule_changes.reason IS 'Причина замены (например: "Преподаватель на больничном", "Кабинет занят для мероприятия")';
