package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/cli"
)

const (
	defaultNumWorkers = 2
)

func Run(commands *cli.CLI) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	commands.SetRunning(true)
	defer commands.SetRunning(false)

	// Запуск команды с контекстом
	go func() {
		commands.Run(ctx, defaultNumWorkers)
	}()

	<-ctx.Done()
	fmt.Println("Received shutdown signal")
	commands.Close()
	fmt.Println("stopped program")
}
