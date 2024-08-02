//go:generate mockgen -source=./service.go -destination=./mocks/service.go -package=mock_service
package api

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/dto"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/kafka"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/metrics"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/pkg/api/proto/order/v1/order/v1"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/pkg/date"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Module interface {
	AcceptOrderCourier(ctx context.Context, order *dto.Order) error
	ReturnOrderCourier(ctx context.Context, orderID int64) error
	IssueOrderClient(ctx context.Context, orderIDs []int64) error
	AcceptReturnClient(ctx context.Context, order *dto.Order) error
	ListOrders(ctx context.Context, recipientID int64, limit int32) ([]*dto.Order, error)
	ListReturnOrders(ctx context.Context, page, limit int32) ([]*dto.Order, error)
}

type KafkaSender interface {
	SendMessage(message *kafka.EventMessage) error
	SendMessages(messages []kafka.EventMessage) error
}

const (
	defaultOrderLimit int32 = 10
)

type OrderService struct {
	order.UnimplementedOrderServer
	Module Module
	Sender KafkaSender
}

func NewOrderService(module Module, sender KafkaSender) *OrderService {
	return &OrderService{Module: module, Sender: sender}
}

func (s *OrderService) AcceptOrderFromCourier(ctx context.Context, req *order.AcceptOrderRequest) (*order.AcceptOrderResponse, error) {
	const op = "api.OrderService.AcceptOrderFromCourier"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	message := kafka.EventMessage{
		EventID:   uuid.New(),
		Timestamp: time.Now(),
		Method:    "/api/v1/orders/accept-order",
		Arguments: req,
	}
	err := s.Sender.SendMessage(&message)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "kafka_send_error", "error", err.Error())

		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := req.ValidateAll(); err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "validation_error", "error", err.Error())

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.Module.AcceptOrderCourier(ctx, &dto.Order{
		OrderID:      req.GetOrderId(),
		RecipientID:  req.GetRecipientId(),
		StorageUntil: req.GetStorageUntil().AsTime(),
		PackageType:  req.GetPackageType(),
		Weight:       req.GetWeight(),
		Cost:         req.GetCost(),
	})
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "module_error", "error", err.Error())

		return nil, handleOrderError(err)
	}

	span.LogKV("event", "order_accepted")

	return &order.AcceptOrderResponse{
		Message: "Order accepted successfully",
		OrderId: req.GetOrderId(),
	}, nil
}

func (s *OrderService) ReturnOrderToCourier(ctx context.Context, req *order.ReturnOrderRequest) (*order.ReturnOrderResponse, error) {
	const op = "api.OrderService.ReturnOrderToCourier"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	message := kafka.EventMessage{
		EventID:   uuid.New(),
		Timestamp: time.Now(),
		Method:    "/api/v1/orders/return-order",
		Arguments: req,
	}
	err := s.Sender.SendMessage(&message)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "kafka_send_error", "error", err.Error())

		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := req.ValidateAll(); err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "validation_error", "error", err.Error())

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.Module.ReturnOrderCourier(ctx, req.GetOrderId())
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "module_error", "error", err.Error())

		return nil, handleOrderError(err)
	}

	span.LogKV("event", "order_returned")

	return &order.ReturnOrderResponse{
		Message: "Order returned successfully",
		OrderId: req.GetOrderId(),
	}, nil
}

func (s *OrderService) IssueOrderToClient(ctx context.Context, req *order.IssueOrderRequest) (*order.IssueOrderResponse, error) {
	const op = "api.OrderService.IssueOrderToClient"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	message := kafka.EventMessage{
		EventID:   uuid.New(),
		Timestamp: time.Now(),
		Method:    "/api/v1/orders/issue-order",
		Arguments: req,
	}
	err := s.Sender.SendMessage(&message)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "kafka_send_error", "error", err.Error())

		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := req.ValidateAll(); err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "validation_error", "error", err.Error())

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.Module.IssueOrderClient(ctx, req.GetOrderIds())
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "module_error", "error", err.Error())

		return nil, handleOrderError(err)
	}

	span.LogKV("event", "order_issued")

	return &order.IssueOrderResponse{
		Message: "Order issued successfully",
	}, nil
}

func (s *OrderService) ListOrders(ctx context.Context, req *order.ListOrdersRequest) (*order.ListOrdersResponse, error) {
	const op = "api.OrderService.ListOrders"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	message := kafka.EventMessage{
		EventID:   uuid.New(),
		Timestamp: time.Now(),
		Method:    "/api/v1/orders/list-orders",
		Arguments: req,
	}
	err := s.Sender.SendMessage(&message)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "kafka_send_error", "error", err.Error())

		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := req.ValidateAll(); err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "validation_error", "error", err.Error())

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	limit := defaultOrderLimit
	if req.Limit != nil {
		limit = req.GetLimit()
	}

	orders, err := s.Module.ListOrders(ctx, req.GetRecipientId(), limit)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "module_error", "error", err.Error())

		return nil, handleOrderError(err)
	}

	return &order.ListOrdersResponse{Orders: orderListToResponse(orders)}, nil
}

func (s *OrderService) AcceptReturnFromClient(ctx context.Context, req *order.AcceptReturnRequest) (*order.AcceptReturnResponse, error) {
	const op = "api.OrderService.AcceptReturnFromClient"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	message := kafka.EventMessage{
		EventID:   uuid.New(),
		Timestamp: time.Now(),
		Method:    "/api/v1/orders/accept-return",
		Arguments: req,
	}
	err := s.Sender.SendMessage(&message)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "kafka_send_error", "error", err.Error())

		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := req.ValidateAll(); err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "validation_error", "error", err.Error())

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = s.Module.AcceptReturnClient(ctx, &dto.Order{
		OrderID:     req.GetOrderId(),
		RecipientID: req.GetRecipientId(),
	})
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "module_error", "error", err.Error())

		return nil, handleOrderError(err)
	}

	return &order.AcceptReturnResponse{
		Message: "Return accepted successfully",
		OrderId: req.GetOrderId(),
	}, nil
}

func (s *OrderService) ReturnList(ctx context.Context, req *order.ReturnListRequest) (*order.ReturnListResponse, error) {
	const op = "api.OrderService.ReturnList"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	start := time.Now()
	defer metrics.ObserveOperationDuration(op, time.Since(start))

	message := kafka.EventMessage{
		EventID:   uuid.New(),
		Timestamp: time.Now(),
		Method:    "/api/v1/orders/return-list",
		Arguments: req,
	}
	err := s.Sender.SendMessage(&message)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "kafka_send_error", "error", err.Error())

		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := req.ValidateAll(); err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "validation_error", "error", err.Error())

		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	limit := defaultOrderLimit
	if req.Limit != nil {
		limit = req.GetLimit()
	}

	orders, err := s.Module.ListReturnOrders(ctx, req.Page, limit)
	if err != nil {
		span.SetTag("error", true)
		span.LogKV("event", "module_error", "error", err.Error())

		return nil, handleOrderError(err)
	}

	return &order.ReturnListResponse{Orders: orderListToResponse(orders)}, nil
}

func orderToResponse(orderDTO *dto.Order) *order.OrderEntity {
	return &order.OrderEntity{
		OrderId:      orderDTO.OrderID,
		RecipientId:  orderDTO.RecipientID,
		StorageUntil: date.ConvertUTCToStr(orderDTO.StorageUntil),
	}
}

func orderListToResponse(orderList []*dto.Order) []*order.OrderEntity {
	orderEntityList := make([]*order.OrderEntity, 0, len(orderList))

	for _, orderDTO := range orderList {
		orderEntityList = append(orderEntityList, orderToResponse(orderDTO))
	}

	return orderEntityList
}
