package main

import (
	"context"
	"fmt"
	"log"

	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/module"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage/postgres"
	"go.uber.org/zap"
)

// Удаление записей, у которых прошло два дня с момента выдачи клиенту
func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Can not initialize zap logger: %v", err)
	}
	defer logger.Sync()

	cfg := config.MustLoad()

	storage, err := postgres.New(cfg.DB)
	if err != nil {
		log.Fatal(err)
	}

	orderService := module.New(storage, storage, storage, storage, logger)

	count, err := orderService.DeleteIssuedOrders(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("deleted %d orders", count)
}
