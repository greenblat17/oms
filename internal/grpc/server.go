package grpc

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/api"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/config"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/kafka"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/metrics"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/middleware"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/internal/module"
	"gitlab.ozon.dev/a_zhuravlev_9785/homework/pkg/api/proto/order/v1/order/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type OrderServer struct {
	Server       *grpc.Server
	orderService *module.Module
	sender       *kafka.Sender
}

func init() {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		metrics.AcceptedReturns,
		metrics.AcceptedOrders,
		metrics.IssuedOrders,
		metrics.ReturnedOrders,
		metrics.OperationDuration,
		metrics.OrdersProcessedTotal,
	)
}

func NewGRPCServer(orderService *module.Module, sender *kafka.Sender) *OrderServer {
	grpcMetrics := grpc_prometheus.NewServerMetrics()

	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     30 * time.Minute, // максимальное время бездействия соединения
		MaxConnectionAge:      30 * time.Minute, // максимальное время соединения
		MaxConnectionAgeGrace: 10 * time.Minute, // время на завершение активных соединений после достижения MaxConnectionAge
		Time:                  10 * time.Minute, // время ожидания перед первым PING
		Timeout:               5 * time.Minute,  // время ожидания PING от клиента
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(kasp),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.ChainUnaryInterceptor(
			grpc_opentracing.UnaryServerInterceptor(),
			grpcMetrics.UnaryServerInterceptor(),
			middleware.Logging,
		),
	)

	grpcMetrics.InitializeMetrics(grpcServer)

	order.RegisterOrderServer(grpcServer, api.NewOrderGRPCService(orderService, sender))

	return &OrderServer{
		Server:       grpcServer,
		orderService: orderService,
		sender:       sender,
	}
}

func (s *OrderServer) RunGRPCServer(cfg *config.Config) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		httpServer := &http.Server{
			Addr:    fmt.Sprintf("%d", cfg.PrometheusPort),
			Handler: promhttp.Handler(),
		}

		if err := httpServer.Serve(lis); err != nil {
			log.Fatalf("Error starting Prometheus HTTP grpc: %v", err)
		}
	}()

	go func() {
		log.Printf("Starting gRPC grpc on port %d...", cfg.GRPCPort)
		if err := s.Server.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC grpc: %v", err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down gRPC grpc...")
	s.Server.GracefulStop()
}

func (s *OrderServer) RunProxyServer(cfg *config.Config) {
	grpcServerEndpoint := flag.String("grpc-endpoint", "localhost:50051", "gRPC endpoint")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Minute,
			Timeout:             5 * time.Minute,
			PermitWithoutStream: true,
		}),
	}

	err := order.RegisterOrderHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)
	if err != nil {
		log.Fatalf("failed to RegisterOrderHandlerFromEndpoint: %v", err)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: middleware.WithHTTPLoggingMiddleware(mux),
	}

	go func() {
		log.Printf("Starting proxy grpc on port %d...", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("Error starting proxy grpc: %v", err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutting down proxy grpc...")
}
