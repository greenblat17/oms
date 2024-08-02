//go:generate mockgen -source ./module.go -destination=./mocks/module.go -package=mock_module
package module

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/domain"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/dto"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/metrics"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage/transactor"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/pkg/slice"
	"go.uber.org/zap"
)

type OrderSaver interface {
	CreateOrder(ctx context.Context, order *domain.Order) error
	UpdateOrder(ctx context.Context, order *domain.Order) error
}

type OrderDeleter interface {
	DeleteOrder(ctx context.Context, orderID int64) error
	DeleteRecipientOrders(ctx context.Context) (int64, error)
}

type OrderProvider interface {
	FindReturnedOrdersWithPagination(ctx context.Context, limit, offset int32) ([]*domain.Order, error)
	FindOrdersByRecipientID(ctx context.Context, recipientID int64) ([]*domain.Order, error)
	FindOrderByID(ctx context.Context, id int64) (*domain.Order, error)
	FindOrderByIDs(ctx context.Context, ids []int64) ([]*domain.Order, error)
}

type TransactionManager interface {
	RunTransactionalQuery(ctx context.Context, isoLevel transactor.TxIsoLevel, accessMode transactor.TxAccessMode,
		queryFunc transactor.QueryFunc) error
}

type Cache interface {
	Set(ctx context.Context, key int64, value *domain.Order)
	Get(ctx context.Context, key int64) (*domain.Order, bool)
}

// Transaction isolation levels
const (
	serializable   transactor.TxIsoLevel = "serializable"
	repeatableRead transactor.TxIsoLevel = "repeatable read"
	readCommitted  transactor.TxIsoLevel = "read committed"
)

// Transaction access modes
const (
	readWrite transactor.TxAccessMode = "read write"
	readOnly  transactor.TxAccessMode = "read only"
)

var (
	ErrRecipientNotFound       = errors.New("recipient with order not found")
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderExists             = errors.New("order already exists")
	ErrOrderStorageTimeExpired = errors.New("storage time is in the past")
	ErrOrderNotExpiredOrIssued = errors.New("order has not expired or has been issued to the client")
	ErrOrderNotIssuedOrExpired = errors.New("order was not issued or more than 2 days passed")
	ErrOrdersDifferentClients  = errors.New("orders belong to different clients")
)

type Module struct {
	orderProvider      OrderProvider
	orderDeleter       OrderDeleter
	orderSaver         OrderSaver
	transactionManager TransactionManager
	cache              Cache
	logger             *zap.Logger
}

// New - конструктор для создания Module
func New(
	orderProvider OrderProvider,
	orderDeleter OrderDeleter,
	orderSaver OrderSaver,
	transactionManager TransactionManager,
	cache Cache,
	logger *zap.Logger,
) *Module {
	return &Module{
		orderProvider:      orderProvider,
		orderSaver:         orderSaver,
		orderDeleter:       orderDeleter,
		transactionManager: transactionManager,
		cache:              cache,
		logger:             logger,
	}
}

// AcceptOrderCourier позволяет принять заказ от курьера
func (m *Module) AcceptOrderCourier(ctx context.Context, order *dto.Order) error {
	const op = "module.Module.AcceptOrderCourier"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	span.LogKV(
		"event", "start_accept_order",
		"order_id", order.OrderID,
		"recipient_id", order.RecipientID,
	)

	if order.StorageUntil.Before(time.Now()) {
		span.SetTag("error", true)
		span.LogKV(
			"event", "storage_time_expired",
			"order_id", order.OrderID,
			"storage_until", order.StorageUntil,
		)

		m.logger.Error("storage time is in the past")

		return fmt.Errorf("%s: %w", op, ErrOrderStorageTimeExpired)
	}

	packageType, err := domain.NewPackageType(order.PackageType)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV(
			"event", "invalid_package_type",
			"order_id", order.OrderID,
			"package_type", order.PackageType,
			"error", err.Error(),
		)

		m.logger.Error("package type not valid", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	acceptedOrder, err := domain.NewOrder(order, packageType)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV(
			"event", "order_creation_error",
			"order_id", acceptedOrder.ID,
			"error", err.Error(),
		)
		m.logger.Error("error creating order", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	err = m.orderSaver.CreateOrder(ctx, acceptedOrder)
	if err != nil {
		if errors.Is(err, storage.ErrOrderExists) {
			span.SetTag("error", true)
			span.LogKV(
				"event", "order_already_exists",
				"order_id", acceptedOrder.ID)
			m.logger.Error("error creating order")

			return fmt.Errorf("%s: %w", op, ErrOrderExists)
		}

		span.SetTag("error", true)
		span.LogKV(
			"event", "order_save_error",
			"order_id", acceptedOrder.ID,
			"error", err.Error(),
		)
		m.logger.Error("error while saving order with transaction", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	span.LogKV(
		"event", "order_saved_to_cache",
		"order_id", acceptedOrder.ID,
	)
	m.cache.Set(ctx, acceptedOrder.ID, acceptedOrder)

	span.LogKV(
		"event", "order_accepted",
		"order_id", acceptedOrder.ID,
	)
	m.logger.Info("accept order from courier was successfully")
	metrics.AddAcceptedOrders()

	return nil
}

// ReturnOrderCourier возвращает заказ курьеру
func (m *Module) ReturnOrderCourier(ctx context.Context, orderID int64) error {
	const op = "module.Module.ReturnOrderCourier"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("order_id", orderID)

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	span.LogKV("event", "start_return_order", "order_id", orderID)

	// Проверяем наличие заказа в кэше
	order, found := m.cache.Get(ctx, orderID)
	if !found {
		var err error
		order, err = m.orderProvider.FindOrderByID(ctx, orderID)
		if err != nil {
			if errors.Is(err, storage.ErrOrderNotFound) {
				span.SetTag("error", true)
				span.LogKV("event", "order_not_found", "order_id", orderID)

				m.logger.Error("order not found")

				return fmt.Errorf("%s: %w", op, ErrOrderNotFound)
			}

			span.SetTag("error", true)
			span.LogKV(
				"event", "find_order_error",
				"order_id", orderID,
				"error", err.Error(),
			)

			m.logger.Error("error while check if order with id exists", zap.Error(err))

			return fmt.Errorf("%s: %w", op, err)
		}

		// Добавляем заказ в кэш
		span.LogKV("event", "order_cached", "order_id", orderID)
		m.cache.Set(ctx, orderID, order)
	}

	returnedOrder := domain.ToDomain(order)

	if returnedOrder.StorageUntil.After(time.Now()) || returnedOrder.IsIssued() {
		span.SetTag("error", true)
		span.LogKV("event", "order_not_returnable", "order_id", orderID)

		m.logger.Error("order already issued or not expired for returned")

		return fmt.Errorf("%s: %w", op, ErrOrderNotExpiredOrIssued)
	}

	span.LogKV("event", "order_returned", "order_id", orderID)
	m.logger.Info("return order to courier was successfully")
	metrics.AddReturnedOrders()

	err := m.orderDeleter.DeleteOrder(ctx, orderID)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV(
			"event", "order_deletion_error",
			"order_id", orderID,
			"error", err.Error(),
		)

		m.logger.Error("error deleting order", zap.Error(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// IssueOrderClient выдает заказ клиенту
func (m *Module) IssueOrderClient(ctx context.Context, orderIDs []int64) error {
	const op = "module.Module.IssueOrderClient"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("order_ids", orderIDs)
	span.LogKV("event", "start_issue_order", "order_ids", orderIDs)

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	// Кэширование: сначала пытаемся получить заказы из кэша
	orders := make([]*domain.Order, 0, len(orderIDs))
	missingOrderIDs := make([]int64, 0, len(orderIDs))

	for _, id := range orderIDs {
		if order, found := m.cache.Get(ctx, id); found {
			orders = append(orders, order)
		} else {
			missingOrderIDs = append(missingOrderIDs, id)
		}
	}

	if len(missingOrderIDs) > 0 {
		// Если есть отсутствующие заказы, получаем их из источника и обновляем кэш
		fetchedOrders, err := m.orderProvider.FindOrderByIDs(ctx, missingOrderIDs)
		if err != nil {
			span.SetTag("error", true)
			span.LogKV(
				"event", "find_orders_error",
				"missing_order_ids", missingOrderIDs,
				"error", err.Error(),
			)

			m.logger.Error("error while checking if orders exist", zap.Error(err))

			return fmt.Errorf("%s: %w", op, err)
		}

		for _, order := range fetchedOrders {
			orders = append(orders, order)

			span.LogKV("event", "order_cached", "order_id", order.ID)
			m.cache.Set(ctx, order.ID, order)
		}
	}

	recipientID := orders[0].RecipientID
	for _, order := range orders {
		if order.RecipientID != recipientID {
			span.SetTag("error", true)
			span.LogKV(
				"event", "different_recipients_error",
				"order_id", order.ID,
				"expected_recipient_id", recipientID,
				"actual_recipient_id", order.RecipientID,
			)

			m.logger.Error("order has the different recipients")

			return fmt.Errorf("%s: %w", op, ErrOrdersDifferentClients)
		}
	}

	m.logger.Info("start transactional")

	err := m.transactionManager.RunTransactionalQuery(ctx, repeatableRead, readWrite, func(ctxTX context.Context) error {
		var errs []error

		for _, order := range orders {
			order.ReturnedAt = sql.NullTime{}
			order.IssuedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}

			err := m.orderSaver.UpdateOrder(ctxTX, order)
			if err != nil {
				span.SetTag("error", true)
				span.LogKV(
					"event", "update_order_error",
					"order_id", order.ID,
					"error", err.Error(),
				)

				m.logger.Error("error while updating order with transaction", zap.Error(err))

				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			span.SetTag("error", true)
			span.LogKV("event", "processing_errors", "errors_count", len(errs))

			metrics.AddOrdersProcessedError(len(errs))

			return fmt.Errorf("%s: %w", op, errors.Join(errs...))
		}

		metrics.AddOrdersProcessedSuccess(len(orders) - len(errs))
		return nil
	})

	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "transaction_error", "error", err.Error())

		return err
	}

	span.LogKV("event", "orders_issued_successfully", "order_ids", orderIDs)
	m.logger.Info("orders issued successfully")
	metrics.AddIssuedOrders()

	return nil
}

// AcceptReturnClient принимает возврат заказа от клиента
func (m *Module) AcceptReturnClient(ctx context.Context, order *dto.Order) error {
	const op = "module.Module.AcceptReturnClient"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("order_id", order.OrderID)
	span.SetTag("recipient_id", order.RecipientID)

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	existedOrder, found := m.cache.Get(ctx, order.OrderID)
	if !found {
		// Если в кэше нет, ищем в источнике и обновляем кэш
		var err error
		existedOrder, err = m.orderProvider.FindOrderByID(ctx, order.OrderID)
		if err != nil {
			if errors.Is(err, storage.ErrOrderNotFound) {
				span.SetTag("error", true)
				span.LogKV("event", "order_not_found", "order_id", order.OrderID)

				m.logger.Error("order not found")

				return fmt.Errorf("%s: %w", op, ErrOrderNotFound)
			}

			span.SetTag("error", true)
			span.LogKV("event", "find_order_error", "order_id", order.OrderID, "error", err.Error())

			m.logger.Error("error while checking if order exists", zap.Error(err))

			return fmt.Errorf("%s: %w", op, err)
		}

		span.LogKV("event", "order_cached", "order_id", order.OrderID)
		m.cache.Set(ctx, order.OrderID, existedOrder)
	}

	if existedOrder.RecipientID != order.RecipientID {
		span.SetTag("error", true)
		span.LogKV(
			"event", "different_recipient_error",
			"order_id", order.OrderID,
			"expected_recipient_id", existedOrder.RecipientID,
			"actual_recipient_id", order.RecipientID,
		)

		m.logger.Error("existed order has different recipient")

		return fmt.Errorf("%s: %w", op, ErrRecipientNotFound)
	}

	const validIssuedHours = 48
	if !existedOrder.IssuedAt.Valid || time.Since(existedOrder.IssuedAt.Time).Hours() > validIssuedHours {
		span.SetTag("error", true)
		span.LogKV(
			"event", "issued_or_expired_error",
			"order_id", order.OrderID,
			"issued_at", existedOrder.IssuedAt.Time,
		)

		m.logger.Error("order was not issued or more than 2 days passed")

		return fmt.Errorf("%s: %w", op, ErrOrderNotIssuedOrExpired)
	}

	existedOrder.IssuedAt = sql.NullTime{}
	existedOrder.ReturnedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}

	err := m.orderSaver.UpdateOrder(ctx, existedOrder)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV(
			"event", "update_order_error",
			"order_id", order.OrderID,
			"error", err.Error(),
		)

		return fmt.Errorf("%s: %w", op, err)
	}

	m.cache.Set(ctx, existedOrder.ID, existedOrder)
	span.LogKV("event", "order_updated_and_cached", "order_id", existedOrder.ID)

	m.logger.Info("return order from client was successfully")
	metrics.AddAcceptedReturns()

	return nil
}

// ListOrders возвращает список всех заказов со склада ПВЗ
func (m *Module) ListOrders(ctx context.Context, recipientID int64, limit int32) ([]*dto.Order, error) {
	const op = "module.Module.ListOrders"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	span.SetTag("recipient_id", recipientID)
	span.SetTag("limit", limit)

	orders, err := m.orderProvider.FindOrdersByRecipientID(ctx, recipientID)
	if err != nil {
		if errors.Is(err, storage.ErrOrderNotFound) {
			span.SetTag("error", true)
			span.LogKV("event", "orders_not_found", "recipient_id", recipientID)

			return nil, ErrOrderNotFound
		}

		span.SetTag("error", true)
		span.LogKV(
			"event", "find_orders_error",
			"recipient_id", recipientID,
			"error", err.Error(),
		)

		m.logger.Error("error while finding orders for recipient", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	size := slice.MinSize(int(limit), len(orders))

	var recipientOrders []*dto.Order

	for i := len(orders) - 1; i >= 0 && size > 0; i-- {
		order := domain.ToDomain(orders[i])
		if order.RecipientID == recipientID && !order.IsIssued() {
			recipientOrders = append(recipientOrders, order)
			size--
		}
	}

	if len(recipientOrders) == 0 {
		span.SetTag("error", true)
		span.LogKV("event", "no_orders_in_pvz", "recipient_id", recipientID)

		m.logger.Error("no one orders in pvz")

		return nil, fmt.Errorf("%s: %w", op, ErrOrderNotFound)
	}

	span.LogKV("event", "orders_listed", "recipient_id", recipientID, "orders_count", len(recipientOrders))

	return recipientOrders, nil
}

// ListReturnOrders возвращает список всех заказов со склада, которые вернули
func (m *Module) ListReturnOrders(ctx context.Context, page, limit int32) ([]*dto.Order, error) {
	const op = "module.Module.ListReturnOrders"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	offset := (page - 1) * limit

	span.SetTag("page", page)
	span.SetTag("limit", limit)
	span.SetTag("offset", offset)

	orders, err := m.orderProvider.FindReturnedOrdersWithPagination(ctx, limit, offset)

	if err != nil {
		span.SetTag("error", true)
		span.LogKV(
			"event", "find_returned_orders_error", "error", err.Error())

		m.logger.Error("errors while finding returned orders", zap.Error(err))

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	var returnedOrders []*dto.Order
	for _, order := range orders {
		returnedOrders = append(returnedOrders, domain.ToDomain(order))
	}

	span.LogKV(
		"event", "returned_orders_listed",
		"limit", limit,
		"orders_count", len(returnedOrders),
	)

	return returnedOrders, nil
}

// DeleteIssuedOrders удаляет заказы из БД, которые забрал клиента больше двух дней назад
func (m *Module) DeleteIssuedOrders(ctx context.Context) (int64, error) {
	const op = "module.Module.DeleteIssuedOrders"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	count, err := m.orderDeleter.DeleteRecipientOrders(ctx)

	if err != nil {
		m.logger.Error("failed to delete old orders", zap.Error(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	m.logger.Info("old orders deleted successfully")

	return count, nil
}
