// Code generated by MockGen. DO NOT EDIT.
// Source: ./service.go

// Package mock_service is a generated GoMock package.
package mock_service

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	dto "gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/dto"
	kafka "gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/kafka"
)

// MockModule is a mock of Module interface.
type MockModule struct {
	ctrl     *gomock.Controller
	recorder *MockModuleMockRecorder
}

// MockModuleMockRecorder is the mock recorder for MockModule.
type MockModuleMockRecorder struct {
	mock *MockModule
}

// NewMockModule creates a new mock instance.
func NewMockModule(ctrl *gomock.Controller) *MockModule {
	mock := &MockModule{ctrl: ctrl}
	mock.recorder = &MockModuleMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModule) EXPECT() *MockModuleMockRecorder {
	return m.recorder
}

// AcceptOrderCourier mocks base method.
func (m *MockModule) AcceptOrderCourier(ctx context.Context, order *dto.Order) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AcceptOrderCourier", ctx, order)
	ret0, _ := ret[0].(error)
	return ret0
}

// AcceptOrderCourier indicates an expected call of AcceptOrderCourier.
func (mr *MockModuleMockRecorder) AcceptOrderCourier(ctx, order interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AcceptOrderCourier", reflect.TypeOf((*MockModule)(nil).AcceptOrderCourier), ctx, order)
}

// AcceptReturnClient mocks base method.
func (m *MockModule) AcceptReturnClient(ctx context.Context, order *dto.Order) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AcceptReturnClient", ctx, order)
	ret0, _ := ret[0].(error)
	return ret0
}

// AcceptReturnClient indicates an expected call of AcceptReturnClient.
func (mr *MockModuleMockRecorder) AcceptReturnClient(ctx, order interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AcceptReturnClient", reflect.TypeOf((*MockModule)(nil).AcceptReturnClient), ctx, order)
}

// IssueOrderClient mocks base method.
func (m *MockModule) IssueOrderClient(ctx context.Context, orderIDs []int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IssueOrderClient", ctx, orderIDs)
	ret0, _ := ret[0].(error)
	return ret0
}

// IssueOrderClient indicates an expected call of IssueOrderClient.
func (mr *MockModuleMockRecorder) IssueOrderClient(ctx, orderIDs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IssueOrderClient", reflect.TypeOf((*MockModule)(nil).IssueOrderClient), ctx, orderIDs)
}

// ListOrders mocks base method.
func (m *MockModule) ListOrders(ctx context.Context, recipientID int64, limit int32) ([]*dto.Order, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListOrders", ctx, recipientID, limit)
	ret0, _ := ret[0].([]*dto.Order)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListOrders indicates an expected call of ListOrders.
func (mr *MockModuleMockRecorder) ListOrders(ctx, recipientID, limit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListOrders", reflect.TypeOf((*MockModule)(nil).ListOrders), ctx, recipientID, limit)
}

// ListReturnOrders mocks base method.
func (m *MockModule) ListReturnOrders(ctx context.Context, page, limit int32) ([]*dto.Order, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListReturnOrders", ctx, page, limit)
	ret0, _ := ret[0].([]*dto.Order)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListReturnOrders indicates an expected call of ListReturnOrders.
func (mr *MockModuleMockRecorder) ListReturnOrders(ctx, page, limit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListReturnOrders", reflect.TypeOf((*MockModule)(nil).ListReturnOrders), ctx, page, limit)
}

// ReturnOrderCourier mocks base method.
func (m *MockModule) ReturnOrderCourier(ctx context.Context, orderID int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReturnOrderCourier", ctx, orderID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReturnOrderCourier indicates an expected call of ReturnOrderCourier.
func (mr *MockModuleMockRecorder) ReturnOrderCourier(ctx, orderID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReturnOrderCourier", reflect.TypeOf((*MockModule)(nil).ReturnOrderCourier), ctx, orderID)
}

// MockKafkaSender is a mock of KafkaSender interface.
type MockKafkaSender struct {
	ctrl     *gomock.Controller
	recorder *MockKafkaSenderMockRecorder
}

// MockKafkaSenderMockRecorder is the mock recorder for MockKafkaSender.
type MockKafkaSenderMockRecorder struct {
	mock *MockKafkaSender
}

// NewMockKafkaSender creates a new mock instance.
func NewMockKafkaSender(ctrl *gomock.Controller) *MockKafkaSender {
	mock := &MockKafkaSender{ctrl: ctrl}
	mock.recorder = &MockKafkaSenderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKafkaSender) EXPECT() *MockKafkaSenderMockRecorder {
	return m.recorder
}

// SendMessage mocks base method.
func (m *MockKafkaSender) SendMessage(message *kafka.EventMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessage", message)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockKafkaSenderMockRecorder) SendMessage(message interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockKafkaSender)(nil).SendMessage), message)
}

// SendMessages mocks base method.
func (m *MockKafkaSender) SendMessages(messages []kafka.EventMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessages", messages)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMessages indicates an expected call of SendMessages.
func (mr *MockKafkaSenderMockRecorder) SendMessages(messages interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessages", reflect.TypeOf((*MockKafkaSender)(nil).SendMessages), messages)
}