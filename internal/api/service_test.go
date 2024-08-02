package api

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mock_service "gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/api/mocks"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/dto"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/kafka"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/testutils"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/pkg/api/proto/order/v1/order/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fixture struct {
	t           *testing.T
	ctrl        *gomock.Controller
	mockModule  *mock_service.MockModule
	mockSender  *mock_service.MockKafkaSender
	grpcService *OrderService
	assert      *assert.Assertions
}

func newFixture(t *testing.T) *fixture {
	ctrl := gomock.NewController(t)

	mockModule := mock_service.NewMockModule(ctrl)
	mockSender := mock_service.NewMockKafkaSender(ctrl)

	grpcService := NewOrderService(mockModule, mockSender)

	assertions := assert.New(t)

	return &fixture{
		t:           t,
		ctrl:        ctrl,
		mockModule:  mockModule,
		mockSender:  mockSender,
		grpcService: grpcService,
		assert:      assertions,
	}
}

func TestOrderGRPCService_AcceptOrderFromCourier(t *testing.T) {
	var (
		ctx = context.Background()
	)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.AcceptOrderRequest{
			OrderId:      1,
			RecipientId:  123,
			StorageUntil: timestamppb.New(time.Now().Add(48 * time.Hour)),
			PackageType:  "Box",
			Weight:       5.5,
			Cost:         100.75,
		}
		fx.mockModule.EXPECT().AcceptOrderCourier(gomock.Any(), &dto.Order{
			OrderID:      req.GetOrderId(),
			RecipientID:  req.GetRecipientId(),
			StorageUntil: req.GetStorageUntil().AsTime(),
			PackageType:  req.GetPackageType(),
			Weight:       req.GetWeight(),
			Cost:         req.GetCost(),
		}).Return(nil)

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/accept-order",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(nil)

		resp, err := fx.grpcService.AcceptOrderFromCourier(ctx, req)

		fx.assert.NoError(err)
		fx.assert.NotNil(resp)
		fx.assert.Equal(uint32(codes.OK), resp.GetStatus())
		fx.assert.Equal("order was accepted", resp.GetMessage())
	})
	t.Run("Invalid Request", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		invalidReq := &order.AcceptOrderRequest{
			OrderId:      -1, // Invalid OrderID
			RecipientId:  12,
			StorageUntil: timestamppb.New(time.Now().Add(time.Hour)),
			PackageType:  "box",
			Weight:       12.0,
			Cost:         125.4,
		}

		_, err := fx.grpcService.AcceptOrderFromCourier(ctx, invalidReq)

		fx.assert.Error(err)
		fx.assert.Equal(codes.InvalidArgument, status.Code(err))
	})
	t.Run("Kafka Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.AcceptOrderRequest{
			OrderId:      1,
			RecipientId:  123,
			StorageUntil: timestamppb.New(time.Now().Add(48 * time.Hour)),
			PackageType:  "Box",
			Weight:       5.5,
			Cost:         100.75,
		}

		fx.mockModule.EXPECT().AcceptOrderCourier(gomock.Any(), &dto.Order{
			OrderID:      req.GetOrderId(),
			RecipientID:  req.GetRecipientId(),
			StorageUntil: req.GetStorageUntil().AsTime(),
			PackageType:  req.GetPackageType(),
			Weight:       req.GetWeight(),
			Cost:         req.GetCost(),
		}).Return(nil)

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/accept-order",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(assert.AnError)

		_, err := fx.grpcService.AcceptOrderFromCourier(ctx, req)

		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
	t.Run("Module Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.AcceptOrderRequest{
			OrderId:      1,
			RecipientId:  123,
			StorageUntil: timestamppb.New(time.Now().Add(48 * time.Hour)),
			PackageType:  "Box",
			Weight:       5.5,
			Cost:         100.75,
		}

		fx.mockModule.EXPECT().AcceptOrderCourier(gomock.Any(), &dto.Order{
			OrderID:      req.GetOrderId(),
			RecipientID:  req.GetRecipientId(),
			StorageUntil: req.GetStorageUntil().AsTime(),
			PackageType:  req.GetPackageType(),
			Weight:       req.GetWeight(),
			Cost:         req.GetCost(),
		}).Return(assert.AnError)

		resp, err := fx.grpcService.AcceptOrderFromCourier(ctx, req)

		fx.assert.Nil(resp)
		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
}

func TestOrderGRPCService_ReturnOrderToCourier(t *testing.T) {
	var (
		orderID int64 = 1
		ctx           = context.Background()
	)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ReturnOrderRequest{
			OrderID: orderID,
		}
		fx.mockModule.EXPECT().ReturnOrderCourier(gomock.Any(), req.GetOrderID()).Return(nil)

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/return-order",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(nil)

		resp, err := fx.grpcService.ReturnOrderToCourier(ctx, req)

		fx.assert.NoError(err)
		fx.assert.NotNil(resp)
		fx.assert.Equal(uint32(codes.OK), resp.GetStatus())
		fx.assert.Equal("order was returned to courier", resp.GetMessage())
	})
	t.Run("Invalid Request", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		invalidReq := &order.ReturnOrderRequest{
			OrderID: -1, // Invalid OrderID
		}

		_, err := fx.grpcService.ReturnOrderToCourier(ctx, invalidReq)

		fx.assert.Error(err)
		fx.assert.Equal(codes.InvalidArgument, status.Code(err))
	})
	t.Run("Kafka Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ReturnOrderRequest{
			OrderID: orderID,
		}

		fx.mockModule.EXPECT().ReturnOrderCourier(gomock.Any(), req.GetOrderID()).Return(nil)

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/return-order",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(assert.AnError)

		_, err := fx.grpcService.ReturnOrderToCourier(ctx, req)

		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
	t.Run("Module Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ReturnOrderRequest{
			OrderID: orderID,
		}
		fx.mockModule.EXPECT().ReturnOrderCourier(gomock.Any(), req.GetOrderID()).Return(assert.AnError)

		resp, err := fx.grpcService.ReturnOrderToCourier(ctx, req)

		fx.assert.Nil(resp)
		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
}

func TestOrderGRPCService_IssueOrderToClient(t *testing.T) {
	var (
		orderIDs = []int64{1, 2, 3}
		ctx      = context.Background()
	)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.IssueOrderRequest{
			OrderIDs: orderIDs,
		}
		fx.mockModule.EXPECT().IssueOrderClient(gomock.Any(), req.OrderIDs).Return(nil)

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/issue-order",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(nil)

		resp, err := fx.grpcService.IssueOrderToClient(ctx, req)

		fx.assert.NoError(err)
		fx.assert.NotNil(resp)
		fx.assert.Equal(uint32(codes.OK), resp.GetStatus())
		fx.assert.Equal("orders was issued to client", resp.GetMessage())
	})
	t.Run("Invalid Request", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		invalidReq := &order.IssueOrderRequest{
			OrderIDs: []int64{}, // Invalid OrderIDs
		}

		_, err := fx.grpcService.IssueOrderToClient(ctx, invalidReq)

		fx.assert.Error(err)
		fx.assert.Equal(codes.InvalidArgument, status.Code(err))
	})
	t.Run("Kafka Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.IssueOrderRequest{
			OrderIDs: orderIDs,
		}

		fx.mockModule.EXPECT().IssueOrderClient(gomock.Any(), req.GetOrderIDs()).Return(nil)

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/issue-order",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(assert.AnError)

		_, err := fx.grpcService.IssueOrderToClient(ctx, req)

		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
	t.Run("Module Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.IssueOrderRequest{
			OrderIDs: orderIDs,
		}

		fx.mockModule.EXPECT().IssueOrderClient(gomock.Any(), req.OrderIDs).Return(assert.AnError)

		resp, err := fx.grpcService.IssueOrderToClient(ctx, req)

		fx.assert.Nil(resp)
		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
		fx.assert.Equal("rpc error: code = Internal desc = assert.AnError general error for testing", err.Error())
	})
}

func TestOrderGRPCService_ListOrders(t *testing.T) {
	var (
		recipientID int64 = 1
		customLimit int32 = 1
		ctx               = context.Background()
	)

	t.Run("Success with default Limit", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ListOrdersRequest{
			RecipientID: recipientID,
			Limit:       nil,
		}
		mockOrders := []*dto.Order{
			{OrderID: 1, RecipientID: 1},
			{OrderID: 2, RecipientID: 1},
			// ...
		}

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/list-orders",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(nil)

		fx.mockModule.EXPECT().ListOrders(gomock.Any(), req.RecipientID, defaultOrderLimit).Return(mockOrders, nil)

		resp, err := fx.grpcService.ListOrders(ctx, req)

		fx.assert.NoError(err)
		fx.assert.NotNil(resp)
		fx.assert.Len(resp.GetOrders(), len(mockOrders))
	})

	t.Run("Success with custom Limit", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ListOrdersRequest{
			RecipientID: recipientID,
			Limit:       &customLimit,
		}
		mockOrders := []*dto.Order{
			{OrderID: 1, RecipientID: 2, StorageUntil: time.Now().Add(time.Hour)},
		}

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/list-orders",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(nil)

		fx.mockModule.EXPECT().ListOrders(gomock.Any(), req.RecipientID, customLimit).Return(mockOrders, nil)

		resp, err := fx.grpcService.ListOrders(ctx, req)

		fx.assert.NoError(err)
		fx.assert.NotNil(resp)
		fx.assert.Len(resp.GetOrders(), len(mockOrders))
	})
	t.Run("Invalid Request", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		invalidReq := &order.ListOrdersRequest{
			RecipientID: 0, // Invalid RecipientID
		}

		_, err := fx.grpcService.ListOrders(ctx, invalidReq)

		fx.assert.Error(err)
		fx.assert.Equal(codes.InvalidArgument, status.Code(err))
	})
	t.Run("Kafka Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ListOrdersRequest{
			RecipientID: recipientID,
			Limit:       &customLimit,
		}
		mockOrders := []*dto.Order{
			{OrderID: 1, RecipientID: 2, StorageUntil: time.Now().Add(time.Hour)},
		}

		fx.mockModule.EXPECT().ListOrders(gomock.Any(), req.RecipientID, customLimit).Return(mockOrders, nil)

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/list-orders",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(assert.AnError)

		_, err := fx.grpcService.ListOrders(ctx, req)

		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
	t.Run("Module Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ListOrdersRequest{
			RecipientID: recipientID,
			Limit:       nil,
		}

		fx.mockModule.EXPECT().ListOrders(gomock.Any(), req.RecipientID, defaultOrderLimit).Return(nil, assert.AnError)

		resp, err := fx.grpcService.ListOrders(ctx, req)

		fx.assert.Nil(resp)
		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
}

func TestOrderGRPCService_AcceptReturnFromClient(t *testing.T) {
	var (
		ctx = context.Background()
	)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.AcceptReturnRequest{
			OrderID:     1,
			RecipientID: 2,
		}
		fx.mockModule.EXPECT().AcceptReturnClient(gomock.Any(), &dto.Order{
			OrderID:     req.OrderID,
			RecipientID: req.RecipientID,
		}).Return(nil)

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/accept-return",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(nil)

		resp, err := fx.grpcService.AcceptReturnFromClient(ctx, req)

		fx.assert.NoError(err)
		fx.assert.NotNil(resp)
		fx.assert.Equal(uint32(codes.OK), resp.Status)
		fx.assert.Equal("order was returned by recipient", resp.Message)
	})
	t.Run("Invalid Request", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		invalidReq := &order.AcceptReturnRequest{
			OrderID:     0, // Invalid OrderID
			RecipientID: 0, // Invalid RecipientID
		}

		_, err := fx.grpcService.AcceptReturnFromClient(ctx, invalidReq)

		fx.assert.Error(err)
		fx.assert.Equal(codes.InvalidArgument, status.Code(err))
	})
	t.Run("Kafka Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.AcceptReturnRequest{
			OrderID:     1,
			RecipientID: 2,
		}
		fx.mockModule.EXPECT().AcceptReturnClient(gomock.Any(), &dto.Order{
			OrderID:     req.OrderID,
			RecipientID: req.RecipientID,
		}).Return(nil)

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/accept-return",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(assert.AnError)

		_, err := fx.grpcService.AcceptReturnFromClient(ctx, req)

		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
	t.Run("Module Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.AcceptReturnRequest{
			OrderID:     1,
			RecipientID: 2,
		}
		fx.mockModule.EXPECT().AcceptReturnClient(gomock.Any(), &dto.Order{
			OrderID:     req.OrderID,
			RecipientID: req.RecipientID,
		}).Return(assert.AnError)

		resp, err := fx.grpcService.AcceptReturnFromClient(ctx, req)

		fx.assert.Nil(resp)
		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
}

func TestOrderGRPCService_ReturnList(t *testing.T) {
	var (
		page        int32 = 1
		customLimit int32 = 1
		ctx               = context.Background()
	)

	t.Run("Success with default Limit", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ReturnListRequest{
			Page:  page,
			Limit: nil,
		}
		mockOrders := []*dto.Order{
			{OrderID: 1, RecipientID: 2, StorageUntil: time.Now().Add(time.Hour)},
			{OrderID: 2, RecipientID: 2, StorageUntil: time.Now().Add(time.Hour)},
		}

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/return-list",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(nil)

		fx.mockModule.EXPECT().ListReturnOrders(gomock.Any(), req.Page, defaultOrderLimit).Return(mockOrders, nil)

		resp, err := fx.grpcService.ReturnList(ctx, req)

		fx.assert.NoError(err)
		fx.assert.NotNil(resp)
		fx.assert.Len(resp.GetOrders(), len(mockOrders))
	})

	t.Run("Success with custom Limit", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ReturnListRequest{
			Page:  page,
			Limit: &customLimit,
		}
		mockOrders := []*dto.Order{
			{OrderID: 1, RecipientID: 2, StorageUntil: time.Now().Add(time.Hour)},
		}

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/return-list",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(nil)

		fx.mockModule.EXPECT().ListReturnOrders(gomock.Any(), req.Page, *req.Limit).Return(mockOrders, nil)

		resp, err := fx.grpcService.ReturnList(ctx, req)

		fx.assert.NoError(err)
		fx.assert.NotNil(resp)
		fx.assert.Len(resp.GetOrders(), len(mockOrders))
	})

	t.Run("Invalid Request", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		invalidReq := &order.ReturnListRequest{
			Page: -1, // Invalid Page
		}

		_, err := fx.grpcService.ReturnList(ctx, invalidReq)

		fx.assert.Error(err)
		fx.assert.Equal(codes.InvalidArgument, status.Code(err))
	})

	t.Run("Kafka Error", func(t *testing.T) {
		t.Parallel()

		fx := newFixture(t)

		req := &order.ReturnListRequest{
			Page:  page,
			Limit: &customLimit,
		}
		mockOrders := []*dto.Order{
			{OrderID: 1, RecipientID: 2, StorageUntil: time.Now().Add(time.Hour)},
		}

		event := &kafka.EventMessage{
			Method: "/api/v1/orders/return-list",
		}
		fx.mockSender.EXPECT().SendMessage(testutils.EventEq(event)).Return(assert.AnError)

		fx.mockModule.EXPECT().ListReturnOrders(gomock.Any(), req.Page, *req.Limit).Return(mockOrders, nil)

		_, err := fx.grpcService.ReturnList(ctx, req)

		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})

	t.Run("Module Error", func(t *testing.T) {
		fx := newFixture(t)

		req := &order.ReturnListRequest{
			Page:  page,
			Limit: nil,
		}
		fx.mockModule.EXPECT().ListReturnOrders(gomock.Any(), req.Page, defaultOrderLimit).Return(nil, assert.AnError)

		resp, err := fx.grpcService.ReturnList(ctx, req)

		fx.assert.Nil(resp)
		fx.assert.Error(err)
		fx.assert.Equal(codes.Internal, status.Code(err))
	})
}
