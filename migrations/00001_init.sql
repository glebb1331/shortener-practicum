-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS urls (
    id           VARCHAR(255) PRIMARY KEY,
    original_url VARCHAR(2048) NOT NULL,
    user_id      VARCHAR(255) NOT NULL DEFAULT '',
    is_deleted   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                               );

CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_original_url ON urls(original_url);
CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS urls;
-- +goose StatementEnd