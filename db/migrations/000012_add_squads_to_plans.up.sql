-- +goose Up

ALTER TABLE plan ADD COLUMN internal_squads TEXT DEFAULT '';
ALTER TABLE plan ADD COLUMN external_squad_uuid TEXT DEFAULT '';
ALTER TABLE plan ADD COLUMN remnawave_tag TEXT DEFAULT '';

-- Миграция значений из settings в базовый тариф
UPDATE plan SET 
    internal_squads = COALESCE((SELECT value FROM settings WHERE key = 'squad_uuids'), ''),
    external_squad_uuid = COALESCE((SELECT value FROM settings WHERE key = 'external_squad_uuid'), ''),
    remnawave_tag = COALESCE((SELECT value FROM settings WHERE key = 'remnawave_tag'), '')
WHERE is_default = TRUE;
