package postgres

import (
	"testing"

	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage/postgres"
)

type TDB struct {
	Storage *postgres.Storage
}

func NewFromEnv(cfg config.DBConfig) *TDB {
	db, err := postgres.New(cfg)
	if err != nil {
		panic(err)
	}

	return &TDB{Storage: db}
}

func (d *TDB) SetUp(t *testing.T) {
	t.Helper()
}

func (d *TDB) TearDown(t *testing.T) {
	t.Helper()
}
