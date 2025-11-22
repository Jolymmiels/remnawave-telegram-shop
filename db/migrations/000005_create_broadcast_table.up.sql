-- +goose Up
-- (если не используешь goose, просто игнорируй эти комментарии)

CREATE TABLE IF NOT EXISTS broadcast (
                                         id         BIGSERIAL PRIMARY KEY,
                                         content    TEXT        NOT NULL,
                                         type       TEXT        NOT NULL,
                                         language   TEXT,
                                         created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT broadcast_type_chk
    CHECK (type IN ('all','active','inactive')),
    CONSTRAINT broadcast_language_lower_chk
    CHECK (language IS NULL OR language = LOWER(language))
    );

-- Индекс для частых сортировок по дате (DESC)
CREATE INDEX IF NOT EXISTS idx_broadcast_created_at
    ON broadcast (created_at DESC);

-- Фильтрация по типу + свежие сверху
CREATE INDEX IF NOT EXISTS idx_broadcast_type_created_at
    ON broadcast (type, created_at DESC);

-- Фильтрация по языку + свежие сверху
CREATE INDEX IF NOT EXISTS idx_broadcast_language_created_at
    ON broadcast (language, created_at DESC);
