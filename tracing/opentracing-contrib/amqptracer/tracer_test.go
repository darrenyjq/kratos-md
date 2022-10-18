package amqptracer

import (
	"testing"

	"codeup.aliyun.com/qimao/go-contrib/prototype/log"
	"github.com/opentracing/opentracing-go"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestInject(t *testing.T) {
	tracer := testTracer{}
	opentracing.SetGlobalTracer(tracer)
	span := tracer.StartSpan("someSpan")
	fakeID := span.Context().(testSpanContext).FakeID
	log.Infof("%+v", fakeID)
	hdrs := amqp.Table{
		"test": "value",
	}
	log.Infof("%+v", Inject(span, hdrs))
	log.Infof("%+v", hdrs)

	tracer1 := testTracer{}
	opentracing.SetGlobalTracer(tracer1)
	ctx, err := Extract(hdrs)
	log.Infof("%+v", ctx)
	log.Infof("%+v", err)
}
