-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY idx_partial_storage_until ON orders (storage_until DESC) WHERE returned_at IS NOT NULL;
-- +goose StatementEnd

-- +goose NO TRANSACTION
-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY idx_partial_storage_until;
-- +goose StatementEnd
