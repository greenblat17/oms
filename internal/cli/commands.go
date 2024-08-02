package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
)

const (
	acceptOrderCourierCommand = "accept-order"
	returnOrderCourierCommand = "return-order"
	issueOrderClientCommand   = "issue-order"
	listOrdersCommand         = "list-orders"
	acceptReturnClientCommand = "accept-return"
	returnListCommand         = "return-list"
	helpCommand               = "help"
	exitCommand               = "exit"
	workersCommand            = "workers"
)

var (
	ErrCommandNotFound       = errors.New("invalid command")
	ErrCommandEmpty          = errors.New("command is not set")
	ErrCommandReading        = errors.New("error reading command")
	ErrWorkersNegativeNumber = errors.New("number of workers must be greater than 0")
)

type command struct {
	name        string
	description string
	call        func(context.Context, []string) (string, error)
}

// initCommandList - описание для команд
func initCommandList(handler *Handler) []command {
	return []command{
		{
			name:        acceptOrderCourierCommand,
			description: "Принять заказ от курьера: использование accept-order --order_id=1 --recipient_id=2 --storage_until=26.05.2024",
			call:        handler.acceptOrderCourier,
		},
		{
			name:        returnOrderCourierCommand,
			description: "Вернуть заказ курьеру: испольщование return-order --order_id=1",
			call:        handler.returnOrderCourier,
		},
		{
			name:        issueOrderClientCommand,
			description: "Выдать заказ клиенту: использование issue-order --order_ids=1,2,4",
			call:        handler.issueOrderClient,
		},
		{
			name:        listOrdersCommand,
			description: "Получить список заказов: использование list-orders [--limit=10]",
			call:        handler.listOrders,
		},
		{
			name:        acceptReturnClientCommand,
			description: "Принять возврат от клиента: использование accept-return --order_id=1 --recipient_id=2",
			call:        handler.acceptReturnClient,
		},
		{
			name:        returnListCommand,
			description: "Получить список возвратов: использование return-list --page=1 [--limit=10]",
			call:        handler.returnList,
		},
		{
			name:        helpCommand,
			description: "Получить справку",
		},
		{
			name:        exitCommand,
			description: "Выйти из программы: использование exit или ctrl+c",
		},
		{
			name:        workersCommand,
			description: "Изменить количество горутин",
		},
	}
}

// help - справка со всеми командами
func (c *CLI) help() {
	for _, cmd := range c.commandList {
		fmt.Println("", cmd.name, cmd.description)
	}
}

// exit - остановка программы
func (c *CLI) exit() {
	c.Close()
	os.Exit(0)
}

func (c *CLI) setWorkerNum(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return ErrCommandEmpty
	}

	numWorkers, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid number of workers: %w", err)
	}

	if numWorkers <= 0 {
		return ErrWorkersNegativeNumber
	}

	c.adjustWorkerCount(ctx, numWorkers)

	return nil
}

// adjustWorkerCount устанавливает количество рабочих горутин.
func (c *CLI) adjustWorkerCount(ctx context.Context, numWorkers int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if numWorkers > c.currentWorkers {
		for i := 0; i < numWorkers-c.currentWorkers; i++ {
			workerCtx, cancel := context.WithCancel(ctx)
			c.cancelFuncs = append(c.cancelFuncs, cancel)
			c.wg.Add(1)

			go c.runWorker(workerCtx, c.currentWorkers+i)
		}
	} else {
		for i := 0; i < c.currentWorkers-numWorkers; i++ {
			c.cancelFuncs[len(c.cancelFuncs)-1]()
			c.cancelFuncs = c.cancelFuncs[:len(c.cancelFuncs)-1]
		}
	}

	c.currentWorkers = numWorkers
}

// Close закрывает все каналы и ожидает завершения всех горутин.
func (c *CLI) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		close(c.errors)
		close(c.jobs)
		close(c.notifyChan)

		c.wg.Wait()
		c.running = false
	}
}
