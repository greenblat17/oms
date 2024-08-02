package main

import (
	"fmt"
	"log"

	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/app"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/cli"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/pkg/api/proto/order/v1/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.MustLoad()

	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", cfg.GRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := order.NewOrderClient(conn)

	orderHandler := cli.NewHandler(client)
	commands := cli.New(orderHandler)

	app.Run(commands)
}
