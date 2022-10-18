package kafkatracer

import (
	"testing"

	"codeup.aliyun.com/qimao/go-contrib/prototype/log"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/opentracing/opentracing-go"
)

func TestInject(t *testing.T) {
	tracer := testTracer{}
	opentracing.SetGlobalTracer(tracer)
	span := tracer.StartSpan("someSpan")
	fakeID := span.Context().(testSpanContext).FakeID
	log.Infof("%+v", fakeID)
	hdrs := []kafka.Header{
		{
			Key:   "test",
			Value: []byte("value"),
		},
	}
	log.Infof("%+v", Inject(span, &hdrs))
	log.Infof("%+v", hdrs)

	tracer1 := testTracer{}
	opentracing.SetGlobalTracer(tracer1)
	ctx, err := Extract(hdrs)
	log.Infof("%+v", ctx)
	log.Infof("%+v", err)
}
