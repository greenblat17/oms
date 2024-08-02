package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/dto"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/pkg/api/proto/order/v1/order/v1"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/pkg/date"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Module interface {
	AcceptOrderCourier(ctx context.Context, order *dto.Order) error
	ReturnOrderCourier(ctx context.Context, orderID int64) error
	IssueOrderClient(ctx context.Context, orderIDs []int64) error
	AcceptReturnClient(ctx context.Context, order *dto.Order) error
	ListOrders(ctx context.Context, recipientID int64, limit int32) ([]*dto.Order, error)
	ListReturnOrders(ctx context.Context, page, limit int32) ([]*dto.Order, error)
}

type Handler struct {
	client order.OrderClient
}

func NewHandler(client order.OrderClient) *Handler {
	return &Handler{
		client: client,
	}
}

// acceptOrderCourier - парсит параметры из командной строк и принимает заказ от клиента
func (h Handler) acceptOrderCourier(ctx context.Context, args []string) (any, error) {
	var (
		orderID, recipientID            int64
		storageUntilStr, packageTypeStr string
		weight, cost                    float64
	)

	fs := flag.NewFlagSet(acceptOrderCourierCommand, flag.ContinueOnError)
	fs.Int64Var(&orderID, "order_id", -1, "ID of the order")
	fs.Int64Var(&recipientID, "recipient_id", -1, "ID of the recipient")
	fs.StringVar(&storageUntilStr, "storage_until", "", "Storage until date (YYYY-MM-DD)")
	fs.StringVar(&packageTypeStr, "package_type", "", "type: film, package or box")
	fs.Float64Var(&weight, "weight", 0, "weight of the order")
	fs.Float64Var(&cost, "cost", 0, "cost of the order")

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	storageUntil, err := date.ParseDateToUTC(storageUntilStr)
	if err != nil {
		return "", fmt.Errorf("can not to parse storageUntil: %w ", err)
	}

	resp, err := h.client.AcceptOrderFromCourier(ctx, &order.AcceptOrderRequest{
		OrderId:      orderID,
		RecipientId:  recipientID,
		StorageUntil: timestamppb.New(storageUntil),
		PackageType:  &packageTypeStr,
		Weight:       weight,
		Cost:         cost,
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// returnOrderCourier - парсит параметры из командной строк и возвращает заказ курьеру
func (h Handler) returnOrderCourier(ctx context.Context, args []string) (any, error) {
	var orderID int64

	fs := flag.NewFlagSet(returnOrderCourierCommand, flag.ContinueOnError)
	fs.Int64Var(&orderID, "order_id", -1, "ID of the order")

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	resp, err := h.client.ReturnOrderToCourier(ctx, &order.ReturnOrderRequest{
		OrderId: orderID,
	})

	if err != nil {
		return nil, err
	}

	fmt.Printf("order with id=%d was returned to courier\n", orderID)

	return resp, nil
}

// issueOrderClient - парсит параметры из командной строк и выдает заказ клиенту
func (h Handler) issueOrderClient(ctx context.Context, args []string) (any, error) {
	var ordersIDStr string

	fs := flag.NewFlagSet(issueOrderClientCommand, flag.ContinueOnError)
	fs.StringVar(&ordersIDStr, "order_ids", "", "ID of the order")

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	orderIDListStr := strings.Split(ordersIDStr, ",")
	if len(orderIDListStr) == 0 {
		return "", errors.New("order_ids is empty")
	}

	ordersIDs := make([]int64, len(orderIDListStr))

	for _, id := range orderIDListStr {
		orderID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return "", err
		}

		ordersIDs = append(ordersIDs, orderID)
	}

	resp, err := h.client.IssueOrderToClient(ctx, &order.IssueOrderRequest{
		OrderIds: ordersIDs,
	})
	if err != nil {
		return nil, err
	}

	fmt.Printf("order with IDs=%v was issued to client", ordersIDs)

	return resp, nil
}

// listOrders - парсит параметры из командной строки и отображает список заказов в ПВЗ
func (h Handler) listOrders(ctx context.Context, args []string) (any, error) {
	var (
		recipientID int64
		limit       int
	)

	fs := flag.NewFlagSet(listOrdersCommand, flag.ContinueOnError)
	fs.Int64Var(&recipientID, "recipient_id", -1, "ID of the recipient")
	fs.IntVar(&limit, "limit", 10, "count of list orders")

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	resp, err := h.client.ListOrders(ctx, &order.ListOrdersRequest{
		RecipientId: recipientID,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// acceptReturnClient - парсит параметры из командной строки и принимает возврат товара от клиента
func (h Handler) acceptReturnClient(ctx context.Context, args []string) (any, error) {
	var orderID, recipientID int64

	fs := flag.NewFlagSet(acceptReturnClientCommand, flag.ContinueOnError)
	fs.Int64Var(&orderID, "order_id", -1, "ID of the order")
	fs.Int64Var(&recipientID, "recipient_id", -1, "ID of the recipient")

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	resp, err := h.client.AcceptReturnFromClient(ctx, &order.AcceptReturnRequest{
		OrderId:     orderID,
		RecipientId: recipientID,
	})
	if err != nil {
		return nil, err
	}

	fmt.Printf("order with id=%d was returned by recipient with id=%d\n", orderID, recipientID)

	return resp, nil
}

// returnList - парсит параметры из командной строки и отображает список возврато
func (h Handler) returnList(ctx context.Context, args []string) (any, error) {
	var (
		page  int
		limit int
	)

	fs := flag.NewFlagSet(returnListCommand, flag.ContinueOnError)
	fs.IntVar(&page, "page", -1, "number of page")
	fs.IntVar(&limit, "limit", 10, "count of returns")

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	resp, err := h.client.ReturnList(ctx, &order.ReturnListRequest{
		Page: int32(page),
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
