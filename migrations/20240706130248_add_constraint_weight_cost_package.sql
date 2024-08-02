-- +goose Up
-- +goose StatementBegin
-- Добавление ограничения CHECK с флагом NOT VALID
ALTER TABLE orders
    ADD CONSTRAINT orders_weight_check CHECK (weight IS NOT NULL) NOT VALID,
    ADD CONSTRAINT orders_order_cost_check CHECK (order_cost IS NOT NULL) NOT VALID,
    ADD CONSTRAINT orders_package_cost_check CHECK (package_cost IS NOT NULL) NOT VALID,
    ADD CONSTRAINT orders_package_type_check CHECK (package_type IS NOT NULL) NOT VALID;

-- Валидация ограничения
ALTER TABLE orders
    VALIDATE CONSTRAINT orders_weight_check,
    VALIDATE CONSTRAINT orders_order_cost_check,
    VALIDATE CONSTRAINT orders_package_cost_check,
    VALIDATE CONSTRAINT orders_package_type_check;

-- Применение ограничения NOT NULL
ALTER TABLE orders
    ALTER COLUMN weight SET NOT NULL,
    ALTER COLUMN order_cost SET NOT NULL,
    ALTER COLUMN package_cost SET NOT NULL,
    ALTER COLUMN package_type SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders
DROP CONSTRAINT orders_weight_check,
    DROP CONSTRAINT orders_order_cost_check,
    DROP CONSTRAINT orders_package_cost_check,
    DROP CONSTRAINT orders_package_type_check;

ALTER TABLE orders
    ALTER COLUMN weight DROP NOT NULL,
    ALTER COLUMN order_cost DROP NOT NULL,
    ALTER COLUMN package_cost DROP NOT NULL,
    ALTER COLUMN package_type DROP NOT NULL;
-- +goose StatementEnd
