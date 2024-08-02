package outbox

import (
	"context"
	"log"
	"time"

	"github.com/IBM/sarama"
	infrakafka "gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/infrastructure/kafka"
)

func (o *OutboxRepo) OutboxProcessor(ctx context.Context, producer *infrakafka.Producer) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			tx, err := o.db.Begin(ctx)
			if err != nil {
				log.Println("Failed to begin transaction:", err)
				continue
			}

			messages, err := o.GetUnprocessedMessages(ctx)
			if err != nil {
				log.Println("Failed to get unprocessed messages:", err)
				tx.Rollback(ctx)
				continue
			}

			for _, msg := range messages {
				kafkaMsg := &sarama.ProducerMessage{
					Topic: msg.Topic,
					Value: sarama.ByteEncoder(msg.Payload),
				}

				_, _, err := producer.SendSyncMessage(kafkaMsg)
				if err != nil {
					log.Println("Failed to send message:", err)
					o.IncrementRetryCount(ctx, msg.ID)
					tx.Rollback(ctx)
					continue
				}

				err = o.MarkMessageProcessed(ctx, msg.ID)
				if err != nil {
					log.Println("Failed to mark message as processed:", err)
					tx.Rollback(ctx)
					continue
				}
			}

			err = tx.Commit(ctx)
			if err != nil {
				log.Println("Failed to commit transaction:", err)
				tx.Rollback(ctx)
			}

		}
	}
}
