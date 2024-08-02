package outbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
)

type OutboxRepo struct {
	db *pgx.Conn
	mu sync.Mutex
}

type OutboxMessage struct {
	ID         uuid.UUID
	Payload    []byte
	Topic      string
	CreatedAt  time.Time
	Processed  bool
	RetryCount int
}

func NewOutboxRepo(cfg config.DBConfig) (*OutboxRepo, error) {
	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}

	return &OutboxRepo{db: conn}, nil
}

func (o *OutboxRepo) CreateMessage(ctx context.Context, msg *OutboxMessage) error {
	tx, err := o.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
			return
		}

		tx.Commit(ctx)
	}()

	commandTag, err := tx.Exec(ctx, "INSERT INTO outbox (id, payload, topic, created_at, processed, retry_count) VALUES ($1, $2, $3, $4, $5, $6)",
		msg.ID, msg.Payload, msg.Topic, msg.CreatedAt, msg.Processed, msg.RetryCount)

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("error creating msg in outbox")
	}

	return err
}

func (o *OutboxRepo) GetUnprocessedMessages(ctx context.Context) ([]OutboxMessage, error) {
	tx, err := o.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
			return
		}

		tx.Commit(ctx)
	}()

	rows, err := tx.Query(ctx, "SELECT id, payload, topic, created_at, processed, retry_count FROM outbox WHERE processed = FALSE AND retry_count < 5")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []OutboxMessage
	for rows.Next() {
		var msg OutboxMessage
		err := rows.Scan(&msg.ID, &msg.Payload, &msg.Topic, &msg.CreatedAt, &msg.Processed, &msg.RetryCount)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (o *OutboxRepo) MarkMessageProcessed(ctx context.Context, msgID uuid.UUID) error {
	tx, err := o.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
			return
		}

		tx.Commit(ctx)

	}()

	_, err = tx.Exec(ctx, "UPDATE outbox SET processed = TRUE WHERE id = $1", msgID)
	return err
}

func (o *OutboxRepo) IncrementRetryCount(ctx context.Context, msgID uuid.UUID) error {
	tx, err := o.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}

		tx.Commit(ctx)
	}()

	_, err = tx.Exec(ctx, "UPDATE outbox SET retry_count = retry_count + 1 WHERE id = $1", msgID)
	return err
}
