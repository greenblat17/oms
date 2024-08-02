package postgres

import (
	"fmt"

	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage/transactor"
)

type Storage struct {
	transactor.QueryEngineProvider
}

func New(cfg config.DBConfig) (*Storage, error) {
	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName)

	provider, err := transactor.New(connString)
	if err != nil {
		return nil, err
	}

	return &Storage{provider}, nil
}

func (s *Storage) Close() {
	s.QueryEngineProvider.Close()
}
