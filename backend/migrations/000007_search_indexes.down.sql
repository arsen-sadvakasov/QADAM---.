-- Откат миграции Phase 11: Search indexes

DROP INDEX IF EXISTS idx_rooms_name_trgm;
DROP INDEX IF EXISTS idx_rooms_number_trgm;
DROP INDEX IF EXISTS idx_groups_name_trgm;
DROP INDEX IF EXISTS idx_subjects_name_trgm;
DROP INDEX IF EXISTS idx_users_full_name_trgm;
-- Расширение pg_trgm не удаляем: может использоваться другими объектами БД.
