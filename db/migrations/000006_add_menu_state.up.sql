CREATE TABLE bot_menu_photo (
    bot_id BIGINT NOT NULL,
    path TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    file_id TEXT NOT NULL,
    PRIMARY KEY (bot_id, path)
);

-- Only this bot's menu pins are tracked; unrelated chat pins are never removed.
-- Retaining pending cleanup rows makes unpin failures retryable after restart.
CREATE TABLE bot_menu_pin (
    bot_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    message_id INTEGER NOT NULL CHECK (message_id > 0),
    PRIMARY KEY (bot_id, chat_id, message_id)
);
