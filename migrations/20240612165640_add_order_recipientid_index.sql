-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY idx_recipient_id ON public.orders USING BTREE (recipient_id);
-- +goose StatementEnd

-- +goose NO TRANSACTION
-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY idx_recipient_id;
-- +goose StatementEnd
