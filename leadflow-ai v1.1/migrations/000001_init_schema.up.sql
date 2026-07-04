-- 000001_init_schema.up.sql
--
-- Первая миграция создаёт минимальную схему: таблицу leads (лиды —
-- компании, найденные и сохранённые пользователем). Полная модель данных
-- для задач поиска (search tasks) появится в v0.2, но базовая таблица
-- leads заводится уже сейчас, чтобы v0.1 можно было проверить сквозным
-- health-check запросом к БД в будущих итерациях.

-- pgcrypto даёт функцию gen_random_uuid(), используемую как значение
-- по умолчанию для первичного ключа. UUID выбран вместо SERIAL/BIGSERIAL,
-- потому что идентификаторы лидов не должны быть последовательно
-- угадываемыми (например, при передаче id во внешние API или ссылки)
-- и не создают конфликтов при будущей репликации/шардировании.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name TEXT NOT NULL,
    website TEXT,
    email TEXT,
    phone TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Индекс по website ускоряет проверку "уже сохраняли ли этот сайт",
-- которая понадобится в v0.2 при сохранении результатов поиска.
CREATE INDEX IF NOT EXISTS idx_leads_website ON leads (website);
