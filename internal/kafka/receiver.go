package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/infrastructure/kafka"
)

type HandleFunc func(message *sarama.ConsumerMessage)

type Receiver struct {
	consumer *kafka.Consumer
	handlers map[string]HandleFunc
}

func NewReceiver(consumer *kafka.Consumer, handlers map[string]HandleFunc) *Receiver {
	return &Receiver{
		consumer: consumer,
		handlers: handlers,
	}
}

func (r *Receiver) Subscribe(topic string) error {
	handler, ok := r.handlers[topic]

	if !ok {
		return errors.New("can not find handler")
	}

	// получаем все партиции топика
	partitionList, err := r.consumer.SingleConsumer.Partitions(topic)

	if err != nil {
		return err
	}

	initialOffset := sarama.OffsetOldest

	for _, partition := range partitionList {
		pc, err := r.consumer.SingleConsumer.ConsumePartition(topic, partition, initialOffset)

		if err != nil {
			return err
		}

		go func(pc sarama.PartitionConsumer, partition int32) {
			for message := range pc.Messages() {
				handler(message)
				fmt.Println("Read Topic: ", topic, " Partition: ", partition, " Offset: ", message.Offset)
			}
		}(pc, partition)
	}

	return nil
}

func (r *Receiver) HandleKafkaMessage(message *sarama.ConsumerMessage) {
	pm := EventMessage{}
	err := json.Unmarshal(message.Value, &pm)
	if err != nil {
		fmt.Println("Consumer error", err)
		return
	}
	fmt.Println("Received Key: ", string(message.Key), " Value: ", pm)
}

func (r *Receiver) SubscribeGroup(brokers []string, topic string) error {
	keepRunning := true
	log.Println("Starting a new Sarama consumer")

	config := sarama.NewConfig()
	config.Version = sarama.MaxVersion

	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	config.Consumer.Group.ResetInvalidOffsets = true

	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second

	config.Consumer.Group.Session.Timeout = 60 * time.Second

	config.Consumer.Group.Rebalance.Timeout = 60 * time.Second

	const BalanceStrategy = "roundrobin"
	switch BalanceStrategy {
	case "sticky":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "roundrobin":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	case "range":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	default:
		log.Panicf("Unrecognized consumer group partition assignor: %s", BalanceStrategy)
	}

	consumer := kafka.NewConsumerGroup()
	group := "event"

	ctx, cancel := context.WithCancel(context.Background())
	client, err := sarama.NewConsumerGroup(brokers, group, config)
	if err != nil {
		log.Panicf("Error creating consumer group client: %v", err)
	}

	consumptionIsPaused := false
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err := client.Consume(ctx, []string{topic}, &consumer); err != nil {
				log.Panicf("Error from consumer: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	<-consumer.Ready()
	log.Println("Sarama consumer up and running!...")

	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	for keepRunning {
		select {
		case <-ctx.Done():
			log.Println("terminating: context cancelled")
			keepRunning = false
		case <-sigterm:
			log.Println("terminating: via signal")
			keepRunning = false
		case <-sigusr1:
			r.toggleConsumptionFlow(client, &consumptionIsPaused)
		}
	}

	cancel()
	wg.Wait()

	if err = client.Close(); err != nil {
		return err
	}

	return nil
}

func (r *Receiver) toggleConsumptionFlow(client sarama.ConsumerGroup, isPaused *bool) {
	if *isPaused {
		client.ResumeAll()
		log.Println("Resuming consumption")
	} else {
		client.PauseAll()
		log.Println("Pausing consumption")
	}

	*isPaused = !*isPaused
}
