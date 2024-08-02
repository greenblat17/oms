package postgres

import (
	"context"
	"errors"
	"log"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/domain"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage"
)

const (
	ordersTable      = "orders"
	uniqueConstraint = "23505"
)

var (
	ordersColumns = []string{"id", "recipient_id", "storage_until",
		"issued_at", "returned_at", "hash", "weight", "order_cost", "package_cost", "package_type"}
)

func (s *Storage) CreateOrder(ctx context.Context, order *domain.Order) error {
	const op = "storage.postgres.Storage.CreateOrder"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("table", ordersTable)

	db := s.QueryEngineProvider.GetQueryEngine(ctx)

	query := sq.Insert(ordersTable).
		Columns(ordersColumns...).
		Values(getValues(order)...).
		PlaceholderFormat(sq.Dollar)

	rowQuery, args, err := query.ToSql()
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "query_build_error", "error", err.Error())

		log.Printf("%s: %v", op, err)
		return err
	}

	commandTag, err := db.Exec(ctx, rowQuery, args...)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueConstraint {
			span.LogKV("event", "unique_constraint_error", "error", storage.ErrOrderExists.Error())

			log.Printf("%s: %v", op, storage.ErrOrderExists)
			return storage.ErrOrderExists
		}

		span.LogKV("event", "db_exec_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return err
	}

	if commandTag.RowsAffected() == 0 {
		span.SetTag("error", true)
		span.LogKV("event", "order_not_created")

		return storage.ErrOrderNotCreated
	}

	span.LogKV("event", "order_created", "order_id", order.ID)

	return nil
}

func (s *Storage) DeleteOrder(ctx context.Context, id int64) error {
	const op = "storage.postgres.Storage.DeleteOrder"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("order_id", id)
	span.SetTag("table", ordersTable)

	db := s.QueryEngineProvider.GetQueryEngine(ctx)

	query := sq.Delete(ordersTable).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	rowQuery, args, err := query.ToSql()
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "query_build_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return err
	}

	commandTag, err := db.Exec(ctx, rowQuery, args...)
	if err != nil {
		span.SetTag("error", true)

		if errors.Is(err, pgx.ErrNoRows) {
			span.LogKV("event", "order_not_found")

			log.Printf("%s: %v", op, storage.ErrOrderNotFound)

			return storage.ErrOrderNotFound
		}

		span.LogKV("event", "db_exec_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return err
	}

	rowsDeleted := commandTag.RowsAffected()

	if rowsDeleted == 0 {
		span.SetTag("error", true)
		span.LogKV("event", "order_not_found")

		return storage.ErrOrderNotFound
	}

	span.LogKV("event", "order_deleted", "rows_deleted", rowsDeleted)

	return nil
}

func (s *Storage) FindOrdersByRecipientID(ctx context.Context, recipientID int64) ([]*domain.Order, error) {
	const op = "storage.postgres.Storage.FindOrdersByRecipientID"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("recipient_id", recipientID)
	span.SetTag("table", ordersTable)

	db := s.QueryEngineProvider.GetQueryEngine(ctx)

	query := sq.Select(ordersColumns...).
		From(ordersTable).
		OrderBy("storage_until DESC").
		Where(sq.Eq{"recipient_id": recipientID}).
		PlaceholderFormat(sq.Dollar)

	rowQuery, args, err := query.ToSql()
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "query_build_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return nil, err
	}

	var orders []*domain.Order

	err = pgxscan.Select(ctx, db, &orders, rowQuery, args...)

	if err != nil {
		span.SetTag("error", true)

		if errors.Is(err, pgx.ErrNoRows) {
			span.LogKV("event", "orders_not_found")
			log.Printf("%s: %v", op, storage.ErrOrderNotFound)
			return nil, storage.ErrOrderNotFound
		}
		span.LogKV("event", "db_select_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return nil, err
	}

	span.LogKV("event", "orders_fetched", "count", len(orders))

	return orders, nil
}

func (s *Storage) FindReturnedOrdersWithPagination(ctx context.Context, limit, offset int32) ([]*domain.Order, error) {
	const op = "storage.postgres.Storage.FindReturnedOrdersWithPagination"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("table", ordersTable)

	db := s.QueryEngineProvider.GetQueryEngine(ctx)

	query := sq.Select(ordersColumns...).
		From(ordersTable).
		Where(sq.NotEq{"returned_at": nil}).
		OrderBy("storage_until DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		PlaceholderFormat(sq.Dollar)

	rowQuery, args, err := query.ToSql()
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "query_build_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return nil, err
	}

	var orders []*domain.Order

	err = pgxscan.Select(ctx, db, &orders, rowQuery, args...)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "db_select_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return nil, err
	}

	span.LogKV("event", "orders_fetched", "count", len(orders))

	return orders, nil
}

func (s *Storage) FindOrderByIDs(ctx context.Context, ids []int64) ([]*domain.Order, error) {
	const op = "storage.postgres.Storage.FindOrderByIDs"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("ids", ids)
	span.SetTag("table", ordersTable)

	db := s.QueryEngineProvider.GetQueryEngine(ctx)

	query := sq.Select(ordersColumns...).
		From(ordersTable).
		Where(sq.Eq{"id": ids}).
		PlaceholderFormat(sq.Dollar)

	rowQuery, args, err := query.ToSql()
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "query_build_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return nil, err
	}

	var orders []*domain.Order
	err = pgxscan.Select(ctx, db, &orders, rowQuery, args...)

	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "db_select_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return nil, err
	}

	span.LogKV("event", "orders_fetched", "count", len(orders))

	return orders, nil
}

func (s *Storage) FindOrderByID(ctx context.Context, id int64) (*domain.Order, error) {
	const op = "storage.postgres.Storage.FindOrderByID"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("id", id)
	span.SetTag("table", ordersTable)

	db := s.QueryEngineProvider.GetQueryEngine(ctx)

	query := sq.Select(ordersColumns...).
		From(ordersTable).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar)

	rowQuery, args, err := query.ToSql()
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "query_build_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return nil, err
	}

	var order domain.Order
	err = pgxscan.Get(ctx, db, &order, rowQuery, args...)

	if err != nil {
		span.SetTag("error", true)

		if errors.Is(err, pgx.ErrNoRows) {
			span.LogKV("event", "order_not_found", "error", storage.ErrOrderNotFound.Error())

			log.Printf("%s: %v", op, storage.ErrOrderNotFound)

			return nil, storage.ErrOrderNotFound
		}

		span.LogKV("event", "db_select_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return nil, err
	}

	span.LogKV("event", "order_fetched", "order_id", id)

	return &order, nil
}

func (s *Storage) UpdateOrder(ctx context.Context, order *domain.Order) error {
	const op = "storage.postgres.Storage.UpdateOrder"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	span.SetTag("order_id", order.ID)
	span.SetTag("table", ordersTable)

	db := s.QueryEngineProvider.GetQueryEngine(ctx)

	query := sq.Update(ordersTable).
		Set("recipient_id", order.RecipientID).
		Set("storage_until", order.StorageUntil).
		Set("returned_at", order.ReturnedAt).
		Set("issued_at", order.IssuedAt).
		Where(sq.Eq{"id": order.ID}).
		PlaceholderFormat(sq.Dollar)

	rowQuery, args, err := query.ToSql()
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "query_build_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return err
	}

	commandTag, err := db.Exec(ctx, rowQuery, args...)
	if err != nil {
		span.SetTag("error", true)

		if errors.Is(err, pgx.ErrNoRows) {
			span.LogKV("event", "order_not_found", "error", storage.ErrOrderNotFound.Error())

			log.Printf("%s: %v", op, storage.ErrOrderNotFound)

			return storage.ErrOrderNotFound
		}

		span.LogKV("event", "db_exec_error", "error", err.Error())

		log.Printf("%s: %v", op, err)

		return err
	}

	rowsUpdated := commandTag.RowsAffected()

	if rowsUpdated == 0 {
		span.SetTag("error", true)
		span.LogKV("event", "no_rows_updated", "error", storage.ErrOrderNotFound.Error())

		return storage.ErrOrderNotFound
	}

	span.LogKV("event", "order_updated", "rows_updated", rowsUpdated)

	return nil
}

func (s *Storage) DeleteRecipientOrders(ctx context.Context) (int64, error) {
	const op = "storage.postgres.Storage.DeleteRecipientOrders"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	db := s.QueryEngineProvider.GetQueryEngine(ctx)

	query := sq.Delete(ordersTable).
		Where(sq.Expr("returned_at IS NOT NULL")).
		Where(sq.Expr("returned_at < NOW() - INTERVAL '2 days'")).
		PlaceholderFormat(sq.Dollar)

	rowQuery, args, err := query.ToSql()
	if err != nil {
		log.Printf("%s: %v", op, err)
		return 0, err
	}

	commandTag, err := db.Exec(ctx, rowQuery, args...)
	if err != nil {
		log.Printf("%s: %v", op, err)
		return 0, err
	}

	rowsDeleted := commandTag.RowsAffected()

	return rowsDeleted, nil
}

func getValues(order *domain.Order) []any {
	return []any{
		order.ID,
		order.RecipientID,
		order.StorageUntil,
		order.IssuedAt,
		order.ReturnedAt,
		order.Hash,
		order.Weight,
		order.Cost,
		order.PackageCost,
		order.PackageType.Type(),
	}
}
