package testutils

import (
	"fmt"

	"github.com/golang/mock/gomock"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/domain"
)

type OrderMatcher struct {
	expected *domain.Order
}

func (m *OrderMatcher) Matches(x interface{}) bool {
	order, ok := x.(*domain.Order)
	if !ok {
		return false
	}

	// Сравниваем все поля, кроме Hash
	return m.expected.ID == order.ID &&
		m.expected.RecipientID == order.RecipientID &&
		m.expected.Weight == order.Weight &&
		m.expected.Cost == order.Cost &&
		m.expected.PackageCost == order.PackageCost &&
		m.expected.PackageType.Type() == order.PackageType.Type() &&
		m.expected.StorageUntil.Equal(order.StorageUntil) &&
		m.expected.IssuedAt.Time.Equal(order.IssuedAt.Time) &&
		m.expected.IssuedAt.Valid == order.IssuedAt.Valid &&
		m.expected.ReturnedAt.Time.Equal(order.ReturnedAt.Time) &&
		m.expected.ReturnedAt.Valid == order.ReturnedAt.Valid
}

func (m *OrderMatcher) String() string {
	return fmt.Sprintf("is equal to %v, ignoring Hash", m.expected)
}

func OrderEq(expected *domain.Order) gomock.Matcher {
	return &OrderMatcher{expected: expected}
}
