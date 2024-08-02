package kafka

import (
	"fmt"

	"github.com/IBM/sarama"
	"github.com/go-faster/errors"
)

type Producer struct {
	brokers      []string
	SyncProducer sarama.SyncProducer
}

func newSyncProducer(brokers []string) (sarama.SyncProducer, error) {
	syncProducerConfig := sarama.NewConfig()

	syncProducerConfig.Producer.Partitioner = sarama.NewRoundRobinPartitioner

	syncProducerConfig.Producer.RequiredAcks = sarama.WaitForLocal

	syncProducerConfig.Producer.CompressionLevel = sarama.CompressionLevelDefault

	syncProducerConfig.Producer.Return.Successes = true
	syncProducerConfig.Producer.Return.Errors = true

	syncProducerConfig.Producer.Compression = sarama.CompressionGZIP

	syncProducer, err := sarama.NewSyncProducer(brokers, syncProducerConfig)

	if err != nil {
		return nil, errors.Wrap(err, "error with sync kafka-producer")
	}

	return syncProducer, nil
}

func NewProducer(brokers []string) (*Producer, error) {
	syncProducer, err := newSyncProducer(brokers)
	if err != nil {
		return nil, errors.Wrap(err, "error with sync kafka-producer")
	}

	producer := &Producer{
		brokers:      brokers,
		SyncProducer: syncProducer,
	}

	return producer, nil
}

func (k *Producer) SendSyncMessage(message *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	return k.SyncProducer.SendMessage(message)
}

func (k *Producer) SendSyncMessages(messages []*sarama.ProducerMessage) error {
	err := k.SyncProducer.SendMessages(messages)
	if err != nil {
		fmt.Println("kafka.Connector.SendMessages error", err)
	}

	return err
}

func (k *Producer) Close() error {
	err := k.SyncProducer.Close()
	if err != nil {
		return errors.Wrap(err, "kafka.Connector.Close")
	}

	return nil
}
