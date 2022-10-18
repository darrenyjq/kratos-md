package xamqp

import (
	"codeup.aliyun.com/qimao/go-contrib/prototype/tracing/opentracing-contrib/amqptracer"
	"github.com/opentracing/opentracing-go"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Options struct {
	URL          string
	QueueName    string
	ExchangeName string
	AutoAck      bool
	Handler      HandlerFunc
	IsDebug      bool
}

type Option func(opts *Options)

func Handler(handler HandlerFunc) Option {
	return func(opts *Options) {
		opts.Handler = func(d amqp.Delivery) error {
			spCtx, _ := amqptracer.Extract(&d.Headers)
			sp := opentracing.StartSpan(
				"ConsumeMessage",
				opentracing.FollowsFrom(spCtx),
			)
			defer sp.Finish()
			return handler(d)
		}
	}
}

func IsDebug(isDebug bool) Option {
	return func(opts *Options) {
		opts.IsDebug = isDebug
	}
}
