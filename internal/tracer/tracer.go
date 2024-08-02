package tracer

import (
	"context"
	"io"
	"log"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	traceconfig "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics/prometheus"
)

func MustSetup(ctx context.Context, serviceName string) {
	cfg := createTracerConfig(serviceName)

	tracer, closer, err := initializeTracer(cfg)
	if err != nil {
		log.Fatalf("ERROR: cannot initialize Jaeger tracer: %s", err)
	}

	handleTracerClosure(ctx, closer)
	setupGlobalTracer(tracer)
}

func StartSpanFromContext(ctx context.Context, operationName string) (context.Context, opentracing.Span) {
	span, newCtx := opentracing.StartSpanFromContext(ctx, operationName)
	return newCtx, span
}

func createTracerConfig(serviceName string) traceconfig.Configuration {
	return traceconfig.Configuration{
		ServiceName: serviceName,
		Sampler: &traceconfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &traceconfig.ReporterConfig{
			// LogSpans: true,
		},
	}
}

func initializeTracer(cfg traceconfig.Configuration) (opentracing.Tracer, io.Closer, error) {
	return cfg.NewTracer(
		traceconfig.Logger(jaeger.StdLogger),
		traceconfig.Metrics(prometheus.New()),
	)
}

func setupGlobalTracer(tracer opentracing.Tracer) {
	opentracing.SetGlobalTracer(tracer)
}

func handleTracerClosure(ctx context.Context, closer io.Closer) {
	go func() {
		<-ctx.Done()
		log.Println("closing tracer")
		if err := closer.Close(); err != nil {
			log.Printf("error closing tracer: %v", err)
		}
	}()
}
