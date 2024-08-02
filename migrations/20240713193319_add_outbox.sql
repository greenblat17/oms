-- +goose Up
-- +goose StatementBegin
CREATE TABLE outbox
(
    id          UUID PRIMARY KEY,
    payload     BYTEA,
    topic       VARCHAR(255),
    created_at  TIMESTAMPTZ,
    processed   BOOLEAN,
    retry_count INT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE outbox;
-- +goose StatementEnd
