package kafka

import (
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/infrastructure/kafka"
)

func TestKafkaSender_SendMessage_Success(t *testing.T) {
	// Arrange
	mockProducer := mocks.NewSyncProducer(t, nil)
	defer mockProducer.Close()

	mockProducer.ExpectSendMessageAndSucceed()

	producer := &kafka.Producer{SyncProducer: mockProducer}
	sender := NewKafkaSender(producer, "test_topic")

	eventID := uuid.New()
	message := &EventMessage{
		EventID:   eventID,
		Timestamp: time.Now(),
		Method:    "TestMethod",
		Arguments: []string{"arg1", "arg2"},
	}

	// Act
	err := sender.SendMessage(message)

	// Assert
	assert.NoError(t, err)
}

func TestKafkaSender_SendMessage_ProducerError(t *testing.T) {
	// Arrange
	mockSyncProducer := mocks.NewSyncProducer(t, nil)
	defer mockSyncProducer.Close()

	mockSyncProducer.ExpectSendMessageAndFail(errors.New("send message error"))

	producer := &kafka.Producer{SyncProducer: mockSyncProducer}
	sender := NewKafkaSender(producer, "test_topic")

	eventID := uuid.New()
	message := &EventMessage{
		EventID:   eventID,
		Timestamp: time.Now(),
		Method:    "TestMethod",
		Arguments: []string{"arg1", "arg2"},
	}

	// Act
	err := sender.SendMessage(message)

	// Assert
	require.Error(t, err)
	assert.Equal(t, "send message error", err.Error())
	mockSyncProducer.Close()
}

func TestKafkaSender_SendMessages_Success(t *testing.T) {
	t.Parallel()

	// arrange
	mockProducer := mocks.NewSyncProducer(t, nil)
	defer mockProducer.Close()

	mockProducer.ExpectSendMessageAndSucceed()
	mockProducer.ExpectSendMessageAndSucceed()

	producer := &kafka.Producer{SyncProducer: mockProducer}
	sender := NewKafkaSender(producer, "test_topic")

	messages := []EventMessage{
		{
			EventID:   uuid.New(),
			Timestamp: time.Now(),
			Method:    "TestMethod1",
			Arguments: []string{"arg1"},
		},
		{
			EventID:   uuid.New(),
			Timestamp: time.Now(),
			Method:    "TestMethod2",
			Arguments: []string{"arg2"},
		},
	}

	// act
	err := sender.SendMessages(messages)

	// assert
	assert.NoError(t, err)
}

func TestKafkaSender_SendMessages_ProducerError(t *testing.T) {
	t.Parallel()

	// arrange
	mockSyncProducer := mocks.NewSyncProducer(t, nil)
	defer mockSyncProducer.Close()

	producer := &kafka.Producer{SyncProducer: mockSyncProducer}
	sender := NewKafkaSender(producer, "test_topic")

	eventID1 := uuid.New()
	eventID2 := uuid.New()
	messages := []EventMessage{
		{
			EventID:   eventID1,
			Timestamp: time.Now(),
			Method:    "TestMethod1",
			Arguments: []string{"arg1", "arg2"},
		},
		{
			EventID:   eventID2,
			Timestamp: time.Now(),
			Method:    "TestMethod2",
			Arguments: []string{"arg3", "arg4"},
		},
	}

	mockSyncProducer.ExpectSendMessageAndFail(errors.New("send messages error"))
	mockSyncProducer.ExpectSendMessageAndSucceed()

	// act
	err := sender.SendMessages(messages)

	// assert
	require.Error(t, err)
	assert.Equal(t, "send messages error", err.Error())
	mockSyncProducer.Close()
}
