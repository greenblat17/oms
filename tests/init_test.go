//go:build integration

package tests

import (
	"log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/tests/postgres"
)

var (
	db *postgres.TDB
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("error loading .env file, %s", err)
	}

	configPath := config.GetValue("TEST_CONFIG_PATH", "./test.yml")
	password := config.GetValue("TEST_DB_PASSWORD", "")

	var cfg config.Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	cfg.DB.Password = password

	db = postgres.NewFromEnv(cfg.DB)
}
