package transactor

import (
	"context"
	"fmt"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type QueryEngine interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

type QueryFunc = func(context.Context) error

type QueryEngineProvider interface {
	GetQueryEngine(ctx context.Context) QueryEngine
	RunTransactionalQuery(ctx context.Context, isoLevel TxIsoLevel, accessMode TxAccessMode, queryFunc QueryFunc) error
	Close()
}

type TransactionManager struct {
	pool *pgxpool.Pool
}

// TxIsoLevel is the transaction isolation level (serializable, repeatable read, read committed or read uncommitted)
type TxIsoLevel string

// TxAccessMode is the transaction access mode (read write or read only)
type TxAccessMode string

type txKey struct{}

func New(connString string) (QueryEngineProvider, error) {
	const op = "storage.transactor.New"

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &TransactionManager{pool: pool}, nil
}

func (tm *TransactionManager) Close() {
	tm.pool.Close()
}

func (tm *TransactionManager) RunTransactionalQuery(
	ctx context.Context,
	isoLevel TxIsoLevel,
	accessMode TxAccessMode,
	queryFunc QueryFunc,
) error {
	tx, err := tm.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.TxIsoLevel(isoLevel),
		AccessMode: pgx.TxAccessMode(accessMode),
	})
	if err != nil {
		return err
	}

	if err := queryFunc(context.WithValue(ctx, txKey{}, tx)); err != nil {
		errRollback := tx.Rollback(ctx)
		if errRollback != nil {
			return fmt.Errorf("%w: %w", errRollback, err)
		}

		return err
	}

	if err := tx.Commit(ctx); err != nil {
		errRollback := tx.Rollback(ctx)
		if errRollback != nil {
			return fmt.Errorf("%w: %w", errRollback, err)
		}

		return err
	}

	return nil
}

func (tm *TransactionManager) GetQueryEngine(ctx context.Context) QueryEngine {
	tx, ok := ctx.Value(txKey{}).(QueryEngine)
	if ok && tx != nil {
		return tx
	}

	return tm.pool
}
