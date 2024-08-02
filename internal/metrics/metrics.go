package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	statusLabel = "status"
)

type metricStatus string

const (
	unknownStatus metricStatus = "unknown"
	successStatus metricStatus = "success"
	errorStatus   metricStatus = "error"
)

var (
	OrdersProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oms_orders_processed",
		Help: "Number of orders processed, labeled by status (e.g., success, error)",
	}, []string{
		statusLabel,
	})

	AcceptedOrders = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oms_accepted_orders",
		Help: "Total number of orders accepted by the courier",
	})

	ReturnedOrders = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oms_returned_orders",
		Help: "Total number of orders returned by the courier",
	})

	IssuedOrders = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oms_issued_orders",
		Help: "Total number of orders issued to clients",
	})

	AcceptedReturns = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oms_accepted_returns",
		Help: "Total number of return orders accepted from clients",
	})

	OperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "oms_operation_duration_seconds",
		Help:    "Duration of operations",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})
)

func addOrdersProcessed(status metricStatus, count int) {
	OrdersProcessed.With(prometheus.Labels{statusLabel: string(status)}).Add(float64(count))
}

func AddOrdersProcessedSuccess(count int) {
	addOrdersProcessed(successStatus, count)
}

func AddOrdersProcessedError(count int) {
	addOrdersProcessed(errorStatus, count)
}

func AddAcceptedOrders() {
	AcceptedOrders.Inc()
}

func AddReturnedOrders() {
	ReturnedOrders.Inc()
}

func AddIssuedOrders() {
	IssuedOrders.Inc()
}

func AddAcceptedReturns() {
	AcceptedReturns.Inc()
}

func ObserveOperationDuration(operation string, duration time.Duration) {
	OperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}
