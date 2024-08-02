-- +goose Up
-- +goose StatementBegin
UPDATE orders
SET weight       = 0,
    order_cost   = 0,
    package_cost = 0,
    package_type = 'without package'
WHERE weight IS NULL
   OR order_cost IS NULL
   OR package_cost IS NULL
   OR package_type IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
