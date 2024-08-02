package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Notification содержит информацию о состоянии выполнения команды
type Notification struct {
	WorkerID int
	Message  string
}

type Job struct {
	CommandName string
	Args        []string
}

type CLI struct {
	handler        *Handler
	commandList    []command
	currentWorkers int
	notifyChan     chan Notification
	jobs           chan Job
	errors         chan error
	cancelFuncs    []context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.Mutex // Мьютекс для синхронизации доступа к cancelFuncs
	running        bool
}

// New создает новый экземпляр CLI.
func New(handler *Handler) *CLI {
	return &CLI{
		handler:        handler,
		commandList:    initCommandList(handler),
		currentWorkers: 0,
		cancelFuncs:    []context.CancelFunc{},
		running:        false,
	}
}

// Run запускает ввод команд от пользователя.
func (c *CLI) Run(ctx context.Context, numWorkers int) {
	c.notifyChan = make(chan Notification)
	c.jobs = make(chan Job)
	c.errors = make(chan error)

	go c.handleNotifications(ctx)
	go c.handleErrors(ctx)
	go c.inputCommands(ctx)

	c.adjustWorkerCount(ctx, numWorkers)
}

func (c *CLI) runWorker(ctx context.Context, workerID int) {
	defer c.wg.Done()

	for {
		select {
		case job, ok := <-c.jobs:
			if !ok {
				return
			}

			c.notifyChan <- Notification{WorkerID: workerID, Message: "started processing"}

			resp, err := c.handleCommands(ctx, job)
			if err != nil {
				c.recordError(err)
				c.notifyChan <- Notification{WorkerID: workerID, Message: "finished processing with error"}
				continue
			}

			fmt.Println(resp)

			c.notifyChan <- Notification{WorkerID: workerID, Message: "finished processing"}
		case <-ctx.Done():
			// Завершение горутины при получении сигнала завершения
			return
		}
	}
}

func (c *CLI) inputCommands(ctx context.Context) {
	reader := bufio.NewReader(os.Stdin)

	for {
		select {
		case <-ctx.Done():
			// Завершение горутины при получении сигнала завершения
			return
		default:
			inputCommand, err := reader.ReadString('\n')
			if err != nil {
				c.recordError(fmt.Errorf("%w: %w", ErrCommandReading, err))
				continue
			}

			inputCommand = strings.TrimSpace(inputCommand)
			if inputCommand == "" {
				c.recordError(ErrCommandEmpty)
				continue
			}

			args := strings.Split(inputCommand, " ")
			c.jobs <- Job{CommandName: args[0], Args: args[1:]}
		}
	}
}

// handleCommands направляет команды от пользователя в нужный метод.
func (c *CLI) handleCommands(ctx context.Context, job Job) (string, error) {
	commandName := job.CommandName
	args := job.Args

	switch commandName {
	case helpCommand:
		c.help()
	case exitCommand:
		c.exit()
	case workersCommand:
		return "", c.setWorkerNum(ctx, args)
	default:
		// orders command
		resp, err := c.executeCommands(ctx, commandName, args)
		if err != nil {
			return "", err
		}

		return resp, nil

	}

	return "", nil
}

func (c *CLI) executeCommands(ctx context.Context, commandName string, args []string) (string, error) {
	for _, cmd := range c.commandList {
		if cmd.name == commandName {
			return cmd.call(ctx, args)
		}
	}

	return "", ErrCommandNotFound
}

// handleNotifications обрабатывает нотификации о состоянии выполнения команд.
func (c *CLI) handleNotifications(ctx context.Context) {
	for {
		select {
		case notify := <-c.notifyChan:
			fmt.Printf("[INFO] Worker %d %s\n", notify.WorkerID, notify.Message)
		case <-ctx.Done():
			// Завершение горутины при получении сигнала завершения
			return
		}
	}
}

func (c *CLI) handleErrors(ctx context.Context) {
	for {
		select {
		case err := <-c.errors:
			fmt.Println("Error: " + err.Error())
		case <-ctx.Done():
			// Завершение горутины при получении сигнала завершения
			return
		}
	}
}

// IsRunning проверяет, запущено ли приложение.
func (c *CLI) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.running
}

// SetRunning устанавливает состояние приложения.
func (c *CLI) SetRunning(state bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = state
}

// recordError записывает ошибку в канал ошибок, если приложение запущено.
func (c *CLI) recordError(err error) {
	if c.IsRunning() {
		c.errors <- err
	}
}
