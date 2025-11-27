-- +goose Down

ALTER TABLE plan DROP COLUMN IF EXISTS internal_squads;
ALTER TABLE plan DROP COLUMN IF EXISTS external_squad_uuid;
ALTER TABLE plan DROP COLUMN IF EXISTS remnawave_tag;
