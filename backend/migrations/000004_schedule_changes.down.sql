-- 000004_schedule_changes.down.sql
-- Откат миграции Phase 6: Schedule Changes

DROP TRIGGER IF EXISTS trigger_schedule_changes_updated_at ON schedule_changes;
DROP FUNCTION IF EXISTS update_schedule_changes_updated_at();
DROP TABLE IF EXISTS schedule_changes CASCADE;
