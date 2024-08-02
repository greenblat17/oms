package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
)

const (
	upMigrateCommand     = "up"
	downMigrateCommand   = "down"
	statusMigrateCommand = "status"
)

func main() {
	var action string

	flag.StringVar(&action, "action", statusMigrateCommand, "The migration action to perform: up or down")
	flag.Parse()

	cfg := config.MustLoad()

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.DB.Username, cfg.DB.Password, net.JoinHostPort(cfg.DB.Host, cfg.DB.Port), cfg.DB.DBName, cfg.DB.SSLMode)

	db, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	switch action {
	case upMigrateCommand:
		if err := goose.Up(db, "migrations"); err != nil {
			log.Printf("Error running up migrations: %v", err)
		}

		log.Println("Migrations up ran successfully")
	case downMigrateCommand:
		if err := goose.Down(db, "migrations"); err != nil {
			log.Printf("Error running down migrations: %v", err)
		}

		log.Println("Migrations down ran successfully")
	case statusMigrateCommand:
		if err := goose.Status(db, "migrations"); err != nil {
			log.Printf("Error getting migrations status: %v", err)
		}
	default:
		log.Printf("Invalid action: %s. Use 'up', 'down', or 'status'", action)
	}
}
