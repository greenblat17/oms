-- +goose Up
-- +goose StatementBegin
CREATE TABLE orders (
    id INT PRIMARY KEY,
    recipient_id INT NOT NULL,
    storage_until TIMESTAMP NOT NULL,
    issued_at TIMESTAMP,
    returned_at TIMESTAMP,
    hash TEXT NOT NULL UNIQUE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE orders;
-- +goose StatementEnd
