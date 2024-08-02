package kafka

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/infrastructure/kafka"
)

func TestKafkaReceiver_Subscribe_Success(t *testing.T) {
	t.Parallel()

	var (
		topic = "test_topic"
	)

	// arrange
	mockConsumer := mocks.NewConsumer(t, nil)
	kafkaConsumer := &kafka.Consumer{SingleConsumer: mockConsumer}

	handledMessage := false
	handlers := map[string]HandleFunc{
		topic: func(message *sarama.ConsumerMessage) {
			handledMessage = true
		},
	}

	mockConsumer.SetTopicMetadata(map[string][]int32{topic: {0}})
	mockPartitionConsumer := mockConsumer.ExpectConsumePartition(topic, 0, sarama.OffsetOldest)
	mockPartitionConsumer.YieldMessage(&sarama.ConsumerMessage{
		Topic: topic,
		Value: []byte("test message"),
	})

	// act
	receiver := NewReceiver(kafkaConsumer, handlers)
	err := receiver.Subscribe(topic)

	// assert
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	assert.True(t, handledMessage)
}

func TestKafkaReceiver_Subscribe_NoHandler(t *testing.T) {
	t.Parallel()

	var (
		topic = "non_existent_topic"
	)

	// arrange
	mockConsumer := mocks.NewConsumer(t, nil)
	kafkaConsumer := &kafka.Consumer{SingleConsumer: mockConsumer}
	handlers := make(map[string]HandleFunc)

	// act
	receiver := NewReceiver(kafkaConsumer, handlers)
	err := receiver.Subscribe(topic)

	// assert
	assert.EqualError(t, err, "can not find handler")
}

func TestKafkaReceiver_Subscribe_PartitionsError(t *testing.T) {
	t.Parallel()

	var (
		topic = "test_topic"
	)

	// arrange
	mockConsumer := mocks.NewConsumer(t, nil)
	mockConsumer.SetTopicMetadata(map[string][]int32{})

	kafkaConsumer := &kafka.Consumer{SingleConsumer: mockConsumer}
	handlers := map[string]HandleFunc{
		topic: func(message *sarama.ConsumerMessage) {},
	}

	// act
	receiver := NewReceiver(kafkaConsumer, handlers)
	err := receiver.Subscribe(topic)

	// assert
	require.Error(t, err)
}

func TestKafkaReceiver_Subscribe_ConsumePartitionError(t *testing.T) {
	t.Parallel()

	var (
		topic = "test_topic"
	)

	mockConsumer := mocks.NewConsumer(t, nil)
	mockConsumer.SetTopicMetadata(map[string][]int32{topic: {0}})

	mockConsumer.ExpectConsumePartition(topic, 0, sarama.OffsetOldest).YieldError(sarama.ErrInvalidMessage)

	kafkaConsumer := &kafka.Consumer{SingleConsumer: mockConsumer}
	handlers := map[string]HandleFunc{
		topic: func(message *sarama.ConsumerMessage) {},
	}

	receiver := NewReceiver(kafkaConsumer, handlers)
	err := receiver.Subscribe(topic)

	if err != nil {
		t.Logf("Test error: %v", err)
	} else {
		t.Logf("No error received")
	}
	assert.EqualError(t, err, "consume partition error")
}
