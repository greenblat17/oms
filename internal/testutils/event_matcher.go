package testutils

import (
	"fmt"

	"github.com/golang/mock/gomock"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/kafka"
)

type EventMatcher struct {
	expected *kafka.EventMessage
}

func (m *EventMatcher) Matches(x interface{}) bool {
	order, ok := x.(*kafka.EventMessage)
	if !ok {
		return false
	}

	// Сравниваем все поля, кроме Hash
	return m.expected.Method == order.Method
}

func (m *EventMatcher) String() string {
	return fmt.Sprintf("is equal to %v, ignoring Hash", m.expected)
}

func EventEq(expected *kafka.EventMessage) gomock.Matcher {
	return &EventMatcher{expected: expected}
}
