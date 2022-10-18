package jaeger

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

// Deprecated: Use NewTracerFromEnv
func NewTracer(service string, rate float64) (opentracing.Tracer, func(), error) {
	tracer, closer, err := config.Configuration{
		ServiceName: service,
		Sampler: &config.SamplerConfig{
			Type:  "probabilistic",
			Param: rate,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, nil, err
	}
	return tracer, func() {
		_ = closer.Close()
	}, nil
}

func NewTracerFromEnv(service string) (opentracing.Tracer, func(), error) {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil, nil, err
	}
	cfg.ServiceName = service
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		return nil, nil, err
	}
	return tracer, func() {
		_ = closer.Close()
	}, nil
}
