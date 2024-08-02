-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders
    ADD COLUMN weight       DOUBLE PRECISION,
    ADD COLUMN order_cost   DOUBLE PRECISION,
    ADD COLUMN package_cost DOUBLE PRECISION,
    ADD COLUMN package_type VARCHAR(255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders
    DROP COLUMN weight,
    DROP COLUMN order_cost,
    DROP COLUMN package_cost,
    DROP COLUMN package_type;
-- +goose StatementEnd
