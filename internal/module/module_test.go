package module

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/domain"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/dto"
	mock_module "gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/module/mocks"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/storage/transactor"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/testutils"
	"go.uber.org/zap"
)

type fixture struct {
	t                      *testing.T
	ctrl                   *gomock.Controller
	mockOrderSaver         *mock_module.MockOrderSaver
	mockOrderProvider      *mock_module.MockOrderProvider
	mockOrderDeleter       *mock_module.MockOrderDeleter
	mockTransactionManager *mock_module.MockTransactionManager
	mockCache              *mock_module.MockCache
	module                 *Module
	logger                 *zap.Logger
	assert                 *assert.Assertions
	require                *require.Assertions
}

func newFixture(t *testing.T) *fixture {
	ctrl := gomock.NewController(t)

	mockOrderSaver := mock_module.NewMockOrderSaver(ctrl)
	mockOrderProvider := mock_module.NewMockOrderProvider(ctrl)
	mockOrderDeleter := mock_module.NewMockOrderDeleter(ctrl)
	mockTransactionManager := mock_module.NewMockTransactionManager(ctrl)
	mockCache := mock_module.NewMockCache(ctrl)

	logger := zap.NewNop()

	orderModule := New(mockOrderProvider, mockOrderDeleter, mockOrderSaver, mockTransactionManager, mockCache, logger)

	assertions := assert.New(t)
	reqAssertions := require.New(t)

	return &fixture{
		t:                      t,
		ctrl:                   ctrl,
		mockOrderSaver:         mockOrderSaver,
		mockOrderProvider:      mockOrderProvider,
		mockOrderDeleter:       mockOrderDeleter,
		mockTransactionManager: mockTransactionManager,
		mockCache:              mockCache,
		module:                 orderModule,
		logger:                 logger,
		assert:                 assertions,
		require:                reqAssertions,
	}
}

// TODO: обновить тесты с учетом кэша
func TestModule_AcceptOrderCourier(t *testing.T) {
	var (
		ctx = context.Background()
	)

	// happy path
	t.Run("should accept order successfully", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		packageType, err := domain.NewPackageType("box")
		require.NoError(t, err)

		order := &dto.Order{
			OrderID:      10,
			RecipientID:  1,
			StorageUntil: time.Now().Add(time.Hour),
			IssuedAt:     time.Time{},
			ReturnAt:     time.Time{},
			Weight:       5.0,
			Cost:         120,
			PackageType:  "box",
		}

		orderEntity, err := domain.NewOrder(order, packageType)
		require.NoError(t, err)

		fx.mockOrderSaver.EXPECT().CreateOrder(gomock.Any(), testutils.OrderEq(orderEntity)).Return(nil).Times(1)

		// act
		err = fx.module.AcceptOrderCourier(ctx, order)

		// assert
		fx.require.NoError(err)
	})
	t.Run("should return error when storage time is in the past", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		order := &dto.Order{
			OrderID:      10,
			RecipientID:  1,
			StorageUntil: time.Now().Add(-time.Hour),
			IssuedAt:     time.Time{},
			ReturnAt:     time.Time{},
			Weight:       5.0,
			Cost:         120,
			PackageType:  "box",
		}

		// act
		err := fx.module.AcceptOrderCourier(ctx, order)

		// assert
		fx.require.Error(err)
		fx.assert.ErrorIs(err, ErrOrderStorageTimeExpired)
	})
	t.Run("should return error order exists when creating order", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		packageType, _ := domain.NewPackageType("box")
		order := &dto.Order{
			OrderID:      10,
			RecipientID:  1,
			StorageUntil: time.Now().Add(time.Hour),
			IssuedAt:     time.Time{},
			ReturnAt:     time.Time{},
			Weight:       5.0,
			Cost:         120,
			PackageType:  "box",
		}

		orderEntity, _ := domain.NewOrder(order, packageType)

		fx.mockOrderSaver.EXPECT().
			CreateOrder(gomock.Any(), testutils.OrderEq(orderEntity)).
			Return(storage.ErrOrderExists).
			Times(1)

		// act
		err := fx.module.AcceptOrderCourier(ctx, order)

		// assert
		fx.require.Error(err)
		fx.assert.ErrorIs(err, ErrOrderExists)
	})
	t.Run("should return some error when creating order fails", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		packageType, _ := domain.NewPackageType("box")
		order := &dto.Order{
			OrderID:      10,
			RecipientID:  1,
			StorageUntil: time.Now().Add(time.Hour),
			IssuedAt:     time.Time{},
			ReturnAt:     time.Time{},
			Weight:       5.0,
			Cost:         120,
			PackageType:  "box",
		}

		orderEntity, _ := domain.NewOrder(order, packageType)

		fx.mockOrderSaver.EXPECT().
			CreateOrder(gomock.Any(), testutils.OrderEq(orderEntity)).
			Return(assert.AnError).
			Times(1)

		// act
		err := fx.module.AcceptOrderCourier(ctx, order)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
	})
}

func TestModule_ReturnOrderCourier(t *testing.T) {
	var (
		ctx               = context.Background()
		orderID     int64 = 1
		storageTime       = time.Now().Add(-time.Hour)
		issuedTime        = sql.NullTime{Valid: false}
	)

	// happy path
	t.Run("should return order successfully", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		order := &domain.Order{
			ID:           orderID,
			StorageUntil: storageTime,
			IssuedAt:     issuedTime,
		}

		gomock.InOrder(
			fx.mockOrderProvider.EXPECT().FindOrderByID(gomock.Any(), orderID).Return(order, nil).Times(1),
			fx.mockOrderDeleter.EXPECT().DeleteOrder(gomock.Any(), orderID).Return(nil).Times(1),
		)

		// act
		err := fx.module.ReturnOrderCourier(ctx, orderID)

		// assert
		fx.require.NoError(err)
	})
	t.Run("should return error when order is not found", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().
			FindOrderByID(gomock.Any(), orderID).
			Return(nil, storage.ErrOrderNotFound).
			Times(1)

		// act
		err := fx.module.ReturnOrderCourier(ctx, orderID)

		// assert
		fx.require.Error(err)
		fx.assert.ErrorIs(err, ErrOrderNotFound)
	})
	t.Run("should return error when find order is failed", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().
			FindOrderByID(gomock.Any(), orderID).
			Return(nil, storage.ErrOrderNotFound).
			Times(1)

		// act
		err := fx.module.ReturnOrderCourier(ctx, orderID)

		// assert
		fx.require.Error(err)
	})
	t.Run("should fail if error occurs while finding order", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().
			FindOrderByID(gomock.Any(), orderID).
			Return(nil, assert.AnError).
			Times(1)

		// act
		err := fx.module.ReturnOrderCourier(ctx, orderID)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
	})
	t.Run("should return error when order is already issued or expired", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		order := &domain.Order{
			ID:           orderID,
			StorageUntil: time.Now().Add(time.Hour), // not expired
			IssuedAt:     issuedTime,
		}

		fx.mockOrderProvider.EXPECT().FindOrderByID(gomock.Any(), orderID).Return(order, nil).Times(1)

		// act
		err := fx.module.ReturnOrderCourier(ctx, orderID)

		// assert
		fx.require.Error(err)
		fx.require.Equal(ErrOrderNotExpiredOrIssued, errors.Unwrap(err))
	})
}

func TestModule_IssueOrderClient(t *testing.T) {
	var (
		ctx      = context.Background()
		orderIDs = []int64{1, 2}
		orders   = []*domain.Order{
			{ID: 1, RecipientID: 3},
			{ID: 2, RecipientID: 3},
		}
		ordersDifferentRecipients = []*domain.Order{
			{ID: 1, RecipientID: 3},
			{ID: 2, RecipientID: 4},
		}
	)

	t.Run("should issue order successfully", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		gomock.InOrder(
			fx.mockOrderProvider.EXPECT().FindOrderByIDs(gomock.Any(), orderIDs).Return(orders, nil).Times(1),
			fx.mockTransactionManager.EXPECT().
				RunTransactionalQuery(gomock.Any(), repeatableRead, readWrite, gomock.Any()).
				DoAndReturn(
					func(ctx context.Context, isoLevel transactor.TxIsoLevel, accessMode transactor.TxAccessMode, queryFunc transactor.QueryFunc) error {
						return queryFunc(ctx)
					}).Times(1),
			fx.mockOrderSaver.EXPECT().UpdateOrder(gomock.Any(), orders[0]).Return(nil).Times(1),
			fx.mockOrderSaver.EXPECT().UpdateOrder(gomock.Any(), orders[1]).Return(nil).Times(1),
		)

		// act
		err := fx.module.IssueOrderClient(ctx, orderIDs)

		// assert
		fx.require.NoError(err)
	})
	t.Run("should return error when orders not exists", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().
			FindOrderByIDs(gomock.Any(), orderIDs).
			Return(nil, assert.AnError).
			Times(1)

		// act
		err := fx.module.IssueOrderClient(context.Background(), orderIDs)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
	})
	t.Run("should return error when orders have different clients", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().FindOrderByIDs(gomock.Any(), orderIDs).Return(ordersDifferentRecipients, nil).Times(1)

		// act
		err := fx.module.IssueOrderClient(ctx, orderIDs)

		// assert
		fx.require.Error(err)
		fx.assert.ErrorIs(err, ErrOrdersDifferentClients)
	})
	t.Run("should fail if unable to update order", func(t *testing.T) {
		t.Parallel()

		// assert
		fx := newFixture(t)

		gomock.InOrder(
			fx.mockOrderProvider.EXPECT().FindOrderByIDs(gomock.Any(), orderIDs).Return(orders, nil).Times(1),
			fx.mockTransactionManager.EXPECT().RunTransactionalQuery(gomock.Any(), repeatableRead, readWrite, gomock.Any()).DoAndReturn(
				func(ctx context.Context, isoLevel transactor.TxIsoLevel, accessMode transactor.TxAccessMode, queryFunc transactor.QueryFunc) error {
					return queryFunc(ctx)
				}).Times(1),
			fx.mockOrderSaver.EXPECT().
				UpdateOrder(gomock.Any(), gomock.AssignableToTypeOf(&domain.Order{IssuedAt: sql.NullTime{Valid: true}})).
				Times(2).
				Return(assert.AnError),
		)

		// act
		err := fx.module.IssueOrderClient(ctx, orderIDs)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
	})
	t.Run("should fail if transactional query fails", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		gomock.InOrder(
			fx.mockOrderProvider.EXPECT().FindOrderByIDs(gomock.Any(), orderIDs).Return(orders, nil).Times(1),
			fx.mockTransactionManager.EXPECT().
				RunTransactionalQuery(gomock.Any(), repeatableRead, readWrite, gomock.Any()).
				Return(assert.AnError).
				Times(1),
		)

		// act
		err := fx.module.IssueOrderClient(ctx, orderIDs)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
	})
}

func TestModule_AcceptReturnClient(t *testing.T) {
	var (
		ctx           = context.Background()
		orderID int64 = 10
		order         = &dto.Order{
			OrderID:     orderID,
			RecipientID: 1,
		}
	)

	t.Run("should accept return successfully", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		existedOrder := &domain.Order{
			ID:          orderID,
			RecipientID: 1,
			IssuedAt:    sql.NullTime{Time: time.Now().Add(-24 * time.Hour), Valid: true},
		}

		gomock.InOrder(
			fx.mockOrderProvider.EXPECT().FindOrderByID(gomock.Any(), orderID).Return(existedOrder, nil).Times(1),
			fx.mockOrderSaver.EXPECT().UpdateOrder(gomock.Any(), existedOrder).Return(nil).Times(1),
		)

		// act
		err := fx.module.AcceptReturnClient(ctx, order)

		// assert
		fx.require.NoError(err)
	})
	t.Run("should fail if order not found", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().FindOrderByID(gomock.Any(), orderID).Return(nil, storage.ErrOrderNotFound).Times(1)

		// act
		err := fx.module.AcceptReturnClient(ctx, order)

		// assert
		fx.require.ErrorIs(err, ErrOrderNotFound)
	})
	t.Run("should fail if occuers db error", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().FindOrderByID(gomock.Any(), orderID).Return(nil, assert.AnError).Times(1)

		// act
		err := fx.module.AcceptReturnClient(ctx, order)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
	})
	t.Run("should fail if recipientID does not match", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		existedOrder := &domain.Order{
			ID:          orderID,
			RecipientID: 2,
			IssuedAt:    sql.NullTime{Time: time.Now().Add(-24 * time.Hour), Valid: true},
		}

		fx.mockOrderProvider.EXPECT().FindOrderByID(gomock.Any(), order.OrderID).Return(existedOrder, nil).Times(1)

		// act
		err := fx.module.AcceptReturnClient(ctx, order)

		// assert
		fx.require.ErrorIs(err, ErrRecipientNotFound)
	})
	t.Run("should fail if order was issued more than 48 hours ago", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		existedOrder := &domain.Order{
			ID:          orderID,
			RecipientID: 1,
			IssuedAt:    sql.NullTime{Time: time.Now().Add(-72 * time.Hour), Valid: true},
		}

		fx.mockOrderProvider.EXPECT().FindOrderByID(gomock.Any(), order.OrderID).Return(existedOrder, nil).Times(1)

		// act
		err := fx.module.AcceptReturnClient(ctx, order)

		// assert
		fx.require.Error(err, ErrOrderNotIssuedOrExpired)
	})
	t.Run("should fail if update order fails", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		order := &dto.Order{
			OrderID:      10,
			RecipientID:  1,
			StorageUntil: time.Now().Add(time.Hour),
			IssuedAt:     time.Now().Add(48 * time.Hour),
			ReturnAt:     time.Time{},
			Weight:       5.0,
			Cost:         120,
		}

		packageType, _ := domain.NewPackageType("box")
		existedOrder, _ := domain.NewOrder(order, packageType)

		gomock.InOrder(
			fx.mockOrderProvider.EXPECT().FindOrderByID(gomock.Any(), order.OrderID).Return(existedOrder, nil).Times(1),
			fx.mockOrderSaver.EXPECT().
				UpdateOrder(gomock.Any(), testutils.OrderEq(existedOrder)).
				Return(assert.AnError).
				Times(1),
		)

		// act
		err := fx.module.AcceptReturnClient(context.Background(), order)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
	})
}

func TestModule_ListOrders(t *testing.T) {
	var (
		ctx               = context.Background()
		recipientID int64 = 1
		limit       int32 = 5
	)

	t.Run("should list orders successfully", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		orders := []*domain.Order{
			{ID: 1, RecipientID: 1},
			{ID: 2, RecipientID: 1},
			{ID: 3, RecipientID: 2},
		}

		fx.mockOrderProvider.EXPECT().FindOrdersByRecipientID(gomock.Any(), recipientID).Return(orders, nil).Times(1)

		// act
		result, err := fx.module.ListOrders(ctx, recipientID, limit)

		// assert
		fx.require.NoError(err)
		fx.require.Len(result, 2)
		fx.require.Equal(1, result[0].RecipientID)
		fx.require.Equal(1, result[1].RecipientID)
	})
	t.Run("should return error if no orders found", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().
			FindOrdersByRecipientID(gomock.Any(), recipientID).
			Return(nil, storage.ErrOrderNotFound).
			Times(1)

		// act
		result, err := fx.module.ListOrders(ctx, recipientID, limit)

		// assert
		fx.require.ErrorIs(err, ErrOrderNotFound)
		fx.assert.Nil(result)
	})
	t.Run("should failed if occurs db error", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().
			FindOrdersByRecipientID(gomock.Any(), recipientID).
			Return(nil, assert.AnError).
			Times(1)

		// act
		result, err := fx.module.ListOrders(ctx, recipientID, limit)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
		fx.assert.Nil(result)
	})
	t.Run("should handle limit larger than available orders", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		orders := []*domain.Order{
			{ID: 1, RecipientID: 1},
			{ID: 2, RecipientID: 1},
		}

		fx.mockOrderProvider.EXPECT().FindOrdersByRecipientID(gomock.Any(), recipientID).Return(orders, nil).Times(1)

		// act
		result, err := fx.module.ListOrders(ctx, recipientID, 10)

		// assert
		fx.require.NoError(err)
		fx.require.Len(result, len(orders))
	})
	t.Run("should handle no matching orders for recipient", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		orders := []*domain.Order{
			{ID: 3, RecipientID: 2}, // different recipient
		}

		fx.mockOrderProvider.EXPECT().FindOrdersByRecipientID(gomock.Any(), recipientID).Return(orders, nil).Times(1)

		// act
		result, err := fx.module.ListOrders(ctx, recipientID, limit)

		// assert
		fx.require.ErrorIs(err, ErrOrderNotFound)
		fx.require.Nil(result)
	})
	t.Run("should handle is issued false", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		orders := []*domain.Order{
			{ID: 3, RecipientID: recipientID, IssuedAt: sql.NullTime{Valid: true, Time: time.Unix(0, 0)}},
		}

		fx.mockOrderProvider.EXPECT().FindOrdersByRecipientID(gomock.Any(), recipientID).Return(orders, nil).Times(1)

		// act
		result, err := fx.module.ListOrders(ctx, recipientID, limit)

		// assert
		fx.require.ErrorIs(err, ErrOrderNotFound)
		fx.assert.Nil(result)
	})
}

func TestModule_ListReturnOrders(t *testing.T) {
	var (
		ctx         = context.Background()
		page  int32 = 1
		limit int32 = 10
	)

	t.Run("should list return orders successfully", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		orders := []*domain.Order{
			{ID: 1, RecipientID: 1, ReturnedAt: sql.NullTime{Valid: true}},
			{ID: 2, RecipientID: 1, ReturnedAt: sql.NullTime{Valid: true}},
		}

		fx.mockOrderProvider.EXPECT().FindReturnedOrdersWithPagination(ctx, limit, 0).Return(orders, nil).Times(1)

		// act
		result, err := fx.module.ListReturnOrders(ctx, page, limit)

		// assert
		fx.require.NoError(err)
		fx.assert.Len(result, 2)
	})

	t.Run("should handle error from order provider", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().
			FindReturnedOrdersWithPagination(ctx, limit, 0).
			Return(nil, assert.AnError).
			Times(1)

		// act
		result, err := fx.module.ListReturnOrders(ctx, page, limit)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
		fx.assert.Nil(result)
	})

	t.Run("should return empty list if no return orders found", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)

		fx.mockOrderProvider.EXPECT().
			FindReturnedOrdersWithPagination(ctx, limit, 0).
			Return([]*domain.Order{}, nil).
			Times(1)

		// act
		result, err := fx.module.ListReturnOrders(ctx, page, limit)

		// assert
		fx.require.NoError(err)
		fx.assert.Empty(result)
	})
}

func TestModule_DeleteIssuedOrders(t *testing.T) {
	var (
		ctx = context.Background()
	)

	t.Run("should delete issued orders successfully", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)
		var countDeletedOrders int64 = 5

		fx.mockOrderDeleter.EXPECT().
			DeleteRecipientOrders(ctx).
			Return(countDeletedOrders, nil).
			Times(1)

		// act
		count, err := fx.module.DeleteIssuedOrders(ctx)

		// assert
		fx.require.NoError(err)
		fx.assert.EqualValues(countDeletedOrders, count)
	})

	t.Run("should handle error from orderDeleter", func(t *testing.T) {
		t.Parallel()

		// arrange
		fx := newFixture(t)
		var countDeletedOrders int64 = 0

		fx.mockOrderDeleter.EXPECT().DeleteRecipientOrders(ctx).Return(countDeletedOrders, assert.AnError).Times(1)

		// act
		count, err := fx.module.DeleteIssuedOrders(ctx)

		// assert
		fx.require.ErrorIs(err, assert.AnError)
		fx.assert.EqualValues(countDeletedOrders, count)
	})
}
