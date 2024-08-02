package main

import (
	"context"
	"log"
	"sync"

	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/grpc"
	infra "gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/infrastructure/kafka"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/kafka"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/module"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage/cache"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage/postgres"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/tracer"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.MustLoad()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Can not initialize zap logger: %v", err)
	}
	defer logger.Sync()

	tracer.MustSetup(ctx, cfg.Name)

	storage, err := postgres.New(cfg.DB)
	if err != nil {
		logger.Fatal("Database initializing error", zap.Error(err))
	}
	defer storage.Close()

	orderCache := cache.NewOrderCache(cfg.CacheConfig.Capacity, cfg.CacheConfig.Type, cfg.CacheConfig.TTL)

	orderService := module.New(storage, storage, storage, storage, orderCache, logger)

	kafkaProducer, err := infra.NewProducer(cfg.Kafka.Brokers)
	if err != nil {
		log.Fatal(err)
	}
	defer kafkaProducer.Close()

	var sender *kafka.Sender
	var receiver *kafka.Receiver

	sender = kafka.NewKafkaSender(kafkaProducer, cfg.Kafka.Topic)
	kafkaConsumer, err := infra.NewConsumer(cfg.Kafka.Brokers)
	if err != nil {
		log.Fatal(err)
	}

	receiver = kafka.NewReceiver(kafkaConsumer, map[string]kafka.HandleFunc{
		cfg.Kafka.Topic: receiver.HandleKafkaMessage,
	})
	receiver.Subscribe(cfg.Kafka.Topic)

	server := grpc.NewGRPCServer(orderService, sender)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		server.RunGRPCServer(cfg)
	}()

	go func() {
		defer wg.Done()
		server.RunProxyServer(cfg)
	}()

	wg.Wait()
}
