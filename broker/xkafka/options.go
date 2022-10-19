package xkafka

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Options struct {
	Topics      []string
	Handler     HandlerFunc
	RebalanceCb kafka.RebalanceCb
	IsDebug     bool
}

type Option func(opts *Options)

func Handler(handler HandlerFunc) Option {
	return func(opts *Options) {
		opts.Handler = func(msg *kafka.Message) {
			// spCtx, _ := kafkatracer.Extract(msg.Headers)
			// sp := opentracing.StartSpan(
			// 	"ConsumeMessage",
			// 	opentracing.FollowsFrom(spCtx),
			// )
			// defer sp.Finish()
			handler(msg)
		}
	}
}

func RebalanceCb(rebalanceCb kafka.RebalanceCb) Option {
	return func(opts *Options) {
		opts.RebalanceCb = rebalanceCb
	}
}

func IsDebug(isDebug bool) Option {
	return func(opts *Options) {
		opts.IsDebug = isDebug
	}
}
