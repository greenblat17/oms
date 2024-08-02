package dto

import (
	"time"
)

type Order struct {
	OrderID      int64     `json:"order_id"`
	RecipientID  int64     `json:"recipient_id"`
	StorageUntil time.Time `json:"storage_until"`
	IssuedAt     time.Time `json:"issued_at"`
	ReturnAt     time.Time `json:"return_at"`
	PackageType  string    `json:"package_type"`
	Weight       float64   `json:"weight"`
	Cost         float64   `json:"cost"`
}

func (o *Order) IsIssued() bool {
	return !o.IssuedAt.IsZero()
}

func (o *Order) IsReturned() bool {
	return !o.ReturnAt.IsZero()
}
