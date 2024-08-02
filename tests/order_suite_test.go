//go:build integration

package tests

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/domain"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/tests/postgres"
)

type OrderTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestOrderTestSuite(t *testing.T) {
	suite.Run(t, new(OrderTestSuite))
}

func (s *OrderTestSuite) SetupSuite() {
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

func (s *OrderTestSuite) SetupTest() {
	db.SetUp(s.T())
}

func (s *OrderTestSuite) TearDownTest() {
	db.TearDown(s.T())
}

func (s *OrderTestSuite) TestCreateOrder() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	id := nextID()
	order := &domain.Order{
		ID:           id,
		RecipientID:  2,
		Weight:       10.32,
		Cost:         123,
		PackageCost:  20,
		PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
		StorageUntil: time.Now().Add(12 * time.Hour),
		IssuedAt:     sql.NullTime{},
		ReturnedAt:   sql.NullTime{},
		Hash:         uuid.New().String(),
	}

	defer deleteOrders(s.ctx, id)

	// act
	fmt.Printf("%+v", order)
	err := db.Storage.CreateOrder(s.ctx, order)

	// assert
	require.NoError(s.T(), err)

	savedOrder, err := db.Storage.FindOrderByID(s.ctx, order.ID)
	require.NoError(s.T(), err)
	assert.ObjectsAreEqual(order, savedOrder)
}

func (s *OrderTestSuite) TestDeleteOrder() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	id := nextID()
	order := &domain.Order{
		ID:           id,
		RecipientID:  2,
		Weight:       10.32,
		Cost:         123,
		PackageCost:  20,
		PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
		StorageUntil: time.Now().Add(12 * time.Hour),
		IssuedAt:     sql.NullTime{},
		ReturnedAt:   sql.NullTime{},
		Hash:         uuid.New().String(),
	}

	fillDb(s.ctx, order)
	defer deleteOrders(s.ctx, id)

	// act
	err := db.Storage.DeleteOrder(s.ctx, id)

	// assert
	require.NoError(s.T(), err)

	deletedOrder, err := db.Storage.FindOrderByID(s.ctx, id)
	assert.ErrorIs(s.T(), err, storage.ErrOrderNotFound)
	assert.Nil(s.T(), deletedOrder)
}

func (s *OrderTestSuite) TestFindOrdersByRecipientID() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	recipientID := 123

	id1 := nextID()
	id2 := nextID()

	orders := []*domain.Order{
		{
			ID:           id1,
			RecipientID:  recipientID,
			Weight:       10.32,
			Cost:         123,
			PackageCost:  20,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
			StorageUntil: time.Now().Add(12 * time.Hour),
			IssuedAt:     sql.NullTime{},
			ReturnedAt:   sql.NullTime{},
			Hash:         uuid.New().String(),
		},
		{
			ID:           id2,
			RecipientID:  recipientID,
			Weight:       5.5,
			Cost:         50,
			PackageCost:  10,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Film{}},
			StorageUntil: time.Now().Add(24 * time.Hour),
			IssuedAt:     sql.NullTime{},
			ReturnedAt:   sql.NullTime{},
			Hash:         uuid.New().String(),
		},
	}

	for _, order := range orders {
		fillDb(s.ctx, order)
	}

	defer deleteOrders(s.ctx, id1, id2)

	// act
	foundOrders, err := db.Storage.FindOrdersByRecipientID(s.ctx, recipientID)

	// assert
	require.NoError(s.T(), err)
	require.Len(s.T(), foundOrders, len(orders))
}

func (s *OrderTestSuite) TestFindReturnedOrdersWithPagination() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	ids := []int{nextID(), nextID(), nextID(), nextID()}

	orders := []*domain.Order{
		{
			ID:           ids[0],
			RecipientID:  1,
			Weight:       10.32,
			Cost:         123,
			PackageCost:  20,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.StandardPackage{}},
			StorageUntil: time.Now().Add(12 * time.Hour),
			IssuedAt:     sql.NullTime{Time: time.Now(), Valid: true},
			ReturnedAt:   sql.NullTime{},
			Hash:         uuid.New().String(),
		},
		{
			ID:           ids[1],
			RecipientID:  1,
			Weight:       5.5,
			Cost:         50,
			PackageCost:  10,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Film{}},
			StorageUntil: time.Now().Add(24 * time.Hour),
			IssuedAt:     sql.NullTime{},
			ReturnedAt:   sql.NullTime{Time: time.Now(), Valid: true},
			Hash:         uuid.New().String(),
		},
		{
			ID:           ids[2],
			RecipientID:  2,
			Weight:       3.2,
			Cost:         30,
			PackageCost:  5,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
			StorageUntil: time.Now().Add(36 * time.Hour),
			IssuedAt:     sql.NullTime{},
			ReturnedAt:   sql.NullTime{Time: time.Now(), Valid: true},
			Hash:         uuid.New().String(),
		},
		{
			ID:           ids[3],
			RecipientID:  1,
			Weight:       3.2,
			Cost:         30,
			PackageCost:  5,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
			StorageUntil: time.Now().Add(36 * time.Hour),
			IssuedAt:     sql.NullTime{},
			ReturnedAt:   sql.NullTime{Time: time.Now(), Valid: true},
			Hash:         uuid.New().String(),
		},
	}

	for _, order := range orders {
		fillDb(s.ctx, order)
	}

	defer deleteOrders(s.ctx, ids...)

	// act
	limit := 3
	offset := 0
	foundOrders, err := db.Storage.FindReturnedOrdersWithPagination(s.ctx, limit, offset)

	// assert
	require.NoError(s.T(), err)
	require.Len(s.T(), foundOrders, limit)
}

func (s *OrderTestSuite) TestFindOrderByIDs() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	ids := []int{nextID(), nextID(), nextID()}

	orders := []*domain.Order{
		{
			ID:           ids[0],
			RecipientID:  1,
			Weight:       10.32,
			Cost:         123,
			PackageCost:  20,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.StandardPackage{}},
			StorageUntil: time.Now().Add(12 * time.Hour),
			IssuedAt:     sql.NullTime{},
			ReturnedAt:   sql.NullTime{Time: time.Now(), Valid: true},
			Hash:         uuid.New().String(),
		},
		{
			ID:           ids[1],
			RecipientID:  1,
			Weight:       5.5,
			Cost:         50,
			PackageCost:  10,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Film{}},
			StorageUntil: time.Now().Add(24 * time.Hour),
			IssuedAt:     sql.NullTime{Time: time.Now(), Valid: true},
			ReturnedAt:   sql.NullTime{},
			Hash:         uuid.New().String(),
		},
		{
			ID:           ids[2],
			RecipientID:  2,
			Weight:       3.2,
			Cost:         30,
			PackageCost:  5,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
			StorageUntil: time.Now().Add(36 * time.Hour),
			IssuedAt:     sql.NullTime{Time: time.Now(), Valid: true},
			ReturnedAt:   sql.NullTime{},
			Hash:         uuid.New().String(),
		},
	}

	for _, order := range orders {
		fillDb(s.ctx, order)
	}

	defer deleteOrders(s.ctx, ids...)

	// act
	resultIDs := []int{ids[0], ids[2]}
	foundOrders, err := db.Storage.FindOrderByIDs(s.ctx, resultIDs)

	// assert
	require.NoError(s.T(), err)
	require.Len(s.T(), foundOrders, len(resultIDs))
}

func (s *OrderTestSuite) TestFindOrderByID_ReturnOrder() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	id := nextID()
	order := &domain.Order{
		ID:           id,
		RecipientID:  1,
		Weight:       10.32,
		Cost:         123,
		PackageCost:  20,
		PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
		StorageUntil: time.Now().Add(12 * time.Hour),
		IssuedAt:     sql.NullTime{},
		ReturnedAt:   sql.NullTime{},
		Hash:         uuid.New().String(),
	}

	fillDb(s.ctx, order)
	defer deleteOrders(s.ctx, id)

	// act
	foundOrder, err := db.Storage.FindOrderByID(s.ctx, order.ID)

	// assert
	require.NoError(s.T(), err)
	require.NotNil(s.T(), foundOrder)
	assert.ObjectsAreEqual(order, foundOrder)

	// act
	nonExistentOrderID := nextID()
	_, err = db.Storage.FindOrderByID(s.ctx, nonExistentOrderID)

	// assert
	require.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, storage.ErrOrderNotFound)
}

func (s *OrderTestSuite) TestFindOrderByID_ReturnErrorIfOrderNotFound() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	nonExistentOrderID := nextID()

	// act
	_, err := db.Storage.FindOrderByID(s.ctx, nonExistentOrderID)

	// assert
	assert.ErrorIs(s.T(), err, storage.ErrOrderNotFound)
}

func (s *OrderTestSuite) TestUpdateOrder_Success() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	id := nextID()
	order := &domain.Order{
		ID:           id,
		RecipientID:  1,
		Weight:       10.32,
		Cost:         123,
		PackageCost:  20,
		PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
		StorageUntil: time.Now().Add(12 * time.Hour),
		IssuedAt:     sql.NullTime{},
		ReturnedAt:   sql.NullTime{},
		Hash:         uuid.New().String(),
	}

	fillDb(s.ctx, order)
	defer deleteOrders(s.ctx, id)

	updatedOrder := &domain.Order{
		ID:           id,
		RecipientID:  2,
		Weight:       15.50,
		Cost:         150,
		PackageCost:  30,
		PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
		StorageUntil: time.Now().Add(24 * time.Hour),
		IssuedAt:     sql.NullTime{},
		ReturnedAt:   sql.NullTime{},
		Hash:         uuid.New().String(),
	}

	// act
	err := db.Storage.UpdateOrder(s.ctx, updatedOrder)

	// assert
	require.NoError(s.T(), err)

	foundOrder, err := db.Storage.FindOrderByID(s.ctx, updatedOrder.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), foundOrder)
	assert.ObjectsAreEqual(updatedOrder, foundOrder)

}

func (s *OrderTestSuite) TestUpdateOrder_ReturnErrorIfOrderNotFound() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	nonExistentOrder := &domain.Order{
		ID:          nextID(),
		RecipientID: 3,
	}

	// act
	err := db.Storage.UpdateOrder(s.ctx, nonExistentOrder)

	// assert
	assert.ErrorIs(s.T(), err, storage.ErrOrderNotFound)
}

func (s *OrderTestSuite) TestDeleteRecipientOrders() {
	s.T().Parallel()

	s.ctx = context.Background()

	// arrange
	ids := []int{nextID(), nextID(), nextID()}

	orders := []*domain.Order{
		{
			ID:           ids[0],
			RecipientID:  1,
			Weight:       10.32,
			Cost:         123,
			PackageCost:  20,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
			StorageUntil: time.Now().Add(12 * time.Hour),
			IssuedAt:     sql.NullTime{},
			ReturnedAt:   sql.NullTime{Time: time.Now().Add(-1 * time.Hour), Valid: true},
			Hash:         uuid.New().String(),
		},
		{
			ID:           ids[1],
			RecipientID:  2,
			Weight:       15.50,
			Cost:         150,
			PackageCost:  30,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
			StorageUntil: time.Now().Add(24 * time.Hour),
			IssuedAt:     sql.NullTime{},
			ReturnedAt:   sql.NullTime{Time: time.Now().Add(-2 * 24 * time.Hour), Valid: true},
			Hash:         uuid.New().String(),
		},
		{
			ID:           ids[2],
			RecipientID:  1,
			Weight:       20.75,
			Cost:         200,
			PackageCost:  25,
			PackageType:  &domain.OrderPackageType{OrderPackager: &domain.Box{}},
			StorageUntil: time.Now().Add(48 * time.Hour),
			IssuedAt:     sql.NullTime{},
			ReturnedAt:   sql.NullTime{Time: time.Now().Add(-3 * 24 * time.Hour), Valid: true},
			Hash:         uuid.New().String(),
		},
	}

	for _, order := range orders {
		fillDb(s.ctx, order)
	}

	defer deleteOrders(s.ctx, ids...)

	// act
	rowsDeleted, err := db.Storage.DeleteRecipientOrders(s.ctx)

	// assert
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), rowsDeleted)
}
