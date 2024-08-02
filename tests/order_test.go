//go:build integration

package tests

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/domain"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage"
)

var (
	counter int64
)

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()
		id  = nextID()
	)

	// arrange
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

	defer deleteOrders(ctx, id)

	// act
	err := db.Storage.CreateOrder(ctx, order)

	// assert
	require.NoError(t, err)

	savedOrder, err := db.Storage.FindOrderByID(ctx, id)
	assert.NoError(t, err)
	assert.ObjectsAreEqual(order, savedOrder)
}

func TestDeleteOrder(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()
		id  = nextID()
	)

	// arrange
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
	fillDb(ctx, order)

	defer deleteOrders(ctx, id)

	// act
	err := db.Storage.DeleteOrder(ctx, id)

	// assert
	require.NoError(t, err)

	deletedOrder, err := db.Storage.FindOrderByID(ctx, id)
	assert.ErrorIs(t, err, storage.ErrOrderNotFound)
	assert.Nil(t, deletedOrder)
}

func TestFindOrdersByRecipientID(t *testing.T) {
	t.Parallel()

	var (
		ctx         = context.Background()
		id1         = nextID()
		id2         = nextID()
		recipientID = 123
	)

	defer deleteOrders(ctx, id1, id2)

	// arrange
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
		fillDb(ctx, order)
	}

	// act
	foundOrders, err := db.Storage.FindOrdersByRecipientID(ctx, recipientID)

	// assert
	require.NoError(t, err)
	require.Len(t, foundOrders, len(orders))

}

func TestFindReturnedOrdersWithPagination(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()
		id1 = nextID()
		id2 = nextID()
		id3 = nextID()
		id4 = nextID()
	)

	// arrange
	orders := []*domain.Order{
		{
			ID:           id1,
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
			ID:           id2,
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
			ID:           id3,
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
			ID:           id4,
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
		fillDb(ctx, order)
	}

	defer deleteOrders(ctx, id1, id2, id3, id4)

	// act
	limit := 3
	offset := 0
	foundOrders, err := db.Storage.FindReturnedOrdersWithPagination(ctx, limit, offset)

	// assert
	require.NoError(t, err)
	require.Len(t, foundOrders, limit)
}

func TestFindOrderByIDs(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()
		id1 = nextID()
		id2 = nextID()
		id3 = nextID()
	)

	// arrange
	orders := []*domain.Order{
		{
			ID:           id1,
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
			ID:           id2,
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
			ID:           id3,
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
		fillDb(ctx, order)
	}

	defer deleteOrders(ctx, id1, id2, id3)

	// act
	ids := []int{id1, id3}
	foundOrders, err := db.Storage.FindOrderByIDs(ctx, ids)

	// assert
	require.NoError(t, err)
	require.Len(t, foundOrders, len(ids))
}

func TestFindOrderByID_ReturnOrder(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()
		id  = nextID()
	)

	// arrange
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

	fillDb(ctx, order)
	defer deleteOrders(ctx, id)

	// act
	foundOrder, err := db.Storage.FindOrderByID(ctx, order.ID)

	// assert
	require.NoError(t, err)
	require.NotNil(t, foundOrder)
	assert.ObjectsAreEqual(order, foundOrder)
}

func TestFindOrderByID_ReturnErrorIfOrderNotFound(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()
		id  = nextID()
	)

	// arrange
	nonExistentOrderID := id

	// act
	_, err := db.Storage.FindOrderByID(ctx, nonExistentOrderID)

	// assert
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrOrderNotFound)
}

func TestUpdateOrder_Success(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()
		id  = nextID()
	)

	// arrange
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

	fillDb(ctx, order)
	defer deleteOrders(ctx, id)

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
	err := db.Storage.UpdateOrder(ctx, updatedOrder)

	// assert
	require.NoError(t, err)

	foundOrder, err := db.Storage.FindOrderByID(ctx, updatedOrder.ID)
	require.NoError(t, err)
	require.NotNil(t, foundOrder)
	assert.ObjectsAreEqual(updatedOrder, foundOrder)
}

func TestUpdateOrder_ReturnErrorIfOrderNotFound(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()
		id  = nextID()
	)

	// arrange
	nonExistentOrder := &domain.Order{
		ID:          id,
		RecipientID: 3,
	}

	// act
	err := db.Storage.UpdateOrder(ctx, nonExistentOrder)

	// assert
	require.Error(t, err)
	assert.ErrorIs(t, err, storage.ErrOrderNotFound)
}

func TestDeleteRecipientOrders(t *testing.T) {
	t.Parallel()

	var (
		ctx                      = context.Background()
		id1                      = nextID()
		id2                      = nextID()
		id3                      = nextID()
		countDeletedOrders int64 = 1
	)

	// arrange
	orders := []*domain.Order{
		{
			ID:           id1,
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
			ID:           id2,
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
			ID:           id3,
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
		fillDb(ctx, order)
	}

	defer deleteOrders(ctx, id1, id2, id3)

	// act
	rowsDeleted, err := db.Storage.DeleteRecipientOrders(ctx)

	// assert
	require.NoError(t, err)
	assert.Equal(t, countDeletedOrders, rowsDeleted)
}

func fillDb(ctx context.Context, order *domain.Order) {
	query := sq.Insert("orders").
		Columns("id", "recipient_id", "storage_until", "issued_at", "returned_at", "hash", "weight", "order_cost", "package_cost", "package_type").
		Values(order.ID, order.RecipientID, order.StorageUntil, order.IssuedAt, order.ReturnedAt, order.Hash, order.Weight, order.Cost, order.PackageCost, order.PackageType.Type()).
		PlaceholderFormat(sq.Dollar)

	rowQuery, args, err := query.ToSql()
	if err != nil {
		log.Printf("Error generating SQL query: %v", err)
		return
	}

	provider := db.Storage.GetQueryEngine(ctx)

	_, err = provider.Exec(ctx, rowQuery, args...)
	if err != nil {
		log.Printf("Error executing SQL query: %v", err)
		return
	}
}

func deleteOrders(ctx context.Context, ids ...int) {
	provider := db.Storage.GetQueryEngine(ctx)

	if len(ids) == 0 {
		return
	}

	placeholders := make([]string, len(ids))
	params := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		params[i] = id
	}

	q := fmt.Sprintf("DELETE FROM orders WHERE id IN (%s)", strings.Join(placeholders, ", "))

	if _, err := provider.Exec(ctx, q, params...); err != nil {
		panic(err)
	}
}

func nextID() int {
	return int(atomic.AddInt64(&counter, 1))
}
