package domain

import (
	"database/sql"
	"time"

	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/dto"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/pkg/hash"
)

// Order структура заказа
type Order struct {
	ID           int64             `db:"id"`
	RecipientID  int64             `db:"recipient_id"`
	Weight       float64           `db:"weight"`
	Cost         float64           `db:"order_cost"`
	PackageCost  float64           `db:"package_cost"`
	PackageType  *OrderPackageType `db:"package_type"`
	StorageUntil time.Time         `db:"storage_until"`
	IssuedAt     sql.NullTime      `db:"issued_at"`
	ReturnedAt   sql.NullTime      `db:"returned_at"`
	Hash         string            `db:"hash"`
}

func NewOrder(order *dto.Order, packageType *OrderPackageType) (*Order, error) {
	err := packageType.ValidateWeight(order.Weight)
	if err != nil {
		return nil, err
	}

	return &Order{
		ID:           order.OrderID,
		RecipientID:  order.RecipientID,
		Weight:       order.Weight,
		Cost:         order.Cost,
		PackageCost:  packageType.GetPackageCost(),
		PackageType:  packageType,
		StorageUntil: order.StorageUntil.UTC(),
		IssuedAt:     sql.NullTime{Time: order.IssuedAt.UTC(), Valid: true},
		ReturnedAt:   sql.NullTime{Time: order.ReturnAt.UTC(), Valid: true},
		Hash:         hash.GenerateHash(),
	}, nil
}

// ToDomain преобразует сущность БД в сущность DTO
func ToDomain(order *Order) *dto.Order {
	return &dto.Order{
		OrderID:      order.ID,
		RecipientID:  order.RecipientID,
		StorageUntil: order.StorageUntil,
		IssuedAt:     order.IssuedAt.Time,
		ReturnAt:     order.ReturnedAt.Time,
	}
}
