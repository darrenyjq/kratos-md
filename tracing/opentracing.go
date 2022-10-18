package tracing

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
)

func init() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Warnf("%+v", err)
		return
	}
	tracer, _, err := cfg.NewTracer()
	if err != nil {
		log.Warnf("%+v", err)
		return
	}
	opentracing.SetGlobalTracer(tracer)
}

func GlobalOpenTracer() opentracing.Tracer {
	return opentracing.GlobalTracer()
}

func StartOpenSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) (opentracing.Span, context.Context) {
	return opentracing.StartSpanFromContext(ctx, operationName, opts...)
}
