-- Phase 4 (Schedule): schedule_templates — регулярное занятие (раздел 20, 34.1
-- спецификации). Конкретные занятия на день/неделю/месяц вычисляются "на
-- лету" из шаблона в коде приложения, а не хранятся построчно на каждый день
-- (решение из раздела 20: избежать раздувания таблицы и рассинхронизации).
--
-- schedule_changes (точечные замены/отмены/переносы) вводится в Phase 6.

CREATE TABLE schedule_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id),
    teacher_id UUID NOT NULL REFERENCES teachers(id),
    room_id UUID NOT NULL REFERENCES rooms(id),
    day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL CHECK (end_time > start_time),
    lesson_type VARCHAR(20) NOT NULL DEFAULT 'lecture'
        CHECK (lesson_type IN ('lecture', 'practice', 'lab', 'seminar', 'other')),
    week_parity VARCHAR(10) NOT NULL DEFAULT 'all'
        CHECK (week_parity IN ('all', 'odd', 'even')),
    valid_from DATE NOT NULL,
    valid_to DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,

    CHECK (valid_to IS NULL OR valid_to >= valid_from)
);

-- Индексы для быстрого построения расписания и проверки конфликтов
-- (раздел 34.2 спецификации).
CREATE INDEX idx_schedule_templates_group_day ON schedule_templates (group_id, day_of_week);
CREATE INDEX idx_schedule_templates_teacher_id ON schedule_templates (teacher_id);
CREATE INDEX idx_schedule_templates_room_id ON schedule_templates (room_id);
