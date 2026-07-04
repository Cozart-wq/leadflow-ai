-- 000001_init_schema.down.sql
--
-- Откатывает изменения из 000001_init_schema.up.sql.
-- Каждая .up-миграция должна иметь парную .down-миграцию — это позволяет
-- безопасно откатывать изменения схемы при ошибке деплоя, не восстанавливая
-- всю БД из бэкапа.

DROP INDEX IF EXISTS idx_leads_website;
DROP TABLE IF EXISTS leads;
