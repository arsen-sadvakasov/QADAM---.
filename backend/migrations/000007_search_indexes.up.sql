-- Phase 11 (Search): GIN-индексы pg_trgm для глобального поиска
-- (разделы 25, 34.2 спецификации). Поиск на старте — ILIKE по ключевым
-- полям (ФИО, название); trgm-индексы ускоряют ILIKE '%...%' запросы.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_users_full_name_trgm ON users USING gin (full_name gin_trgm_ops);
CREATE INDEX idx_subjects_name_trgm ON subjects USING gin (name gin_trgm_ops);
CREATE INDEX idx_groups_name_trgm ON groups USING gin (name gin_trgm_ops);
CREATE INDEX idx_rooms_number_trgm ON rooms USING gin (number gin_trgm_ops);
CREATE INDEX idx_rooms_name_trgm ON rooms USING gin (name gin_trgm_ops);

COMMENT ON EXTENSION pg_trgm IS 'Trigram matching for ILIKE search acceleration (Phase 11)';
