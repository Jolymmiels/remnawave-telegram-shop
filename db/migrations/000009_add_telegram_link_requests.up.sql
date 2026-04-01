CREATE TABLE telegram_link_request
(
    id          BIGSERIAL PRIMARY KEY,
    customer_id BIGINT                   NOT NULL REFERENCES customer (id) ON DELETE CASCADE,
    token       TEXT                     NOT NULL UNIQUE,
    telegram_id BIGINT,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at     TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_telegram_link_request_customer_id
    ON telegram_link_request (customer_id);

CREATE INDEX idx_telegram_link_request_expires_at
    ON telegram_link_request (expires_at);
