package kafka

import (
	"context"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/darrenyjq/kratos-md/broker/xkafka"
	"github.com/darrenyjq/kratos-md/utils/async"
	"github.com/go-kratos/kratos/v2/log"
)

type Client interface {
	Topic() string
	// Deprecated: Use Publish in this package instead.
	Produce(value []byte, key []byte) error
	// Deprecated: Use PublishWithoutKey in this package instead.
	ProduceWithoutKey(value []byte) error
	// Deprecated: Use PublishWithEvent in this package instead.
	ProduceWithEvent(value []byte, key []byte, event chan kafka.Event) error
	// Deprecated: Use PublishRaw in this package instead.
	ProduceRaw(msg *kafka.Message, event chan kafka.Event) error
	Publish(ctx context.Context, value []byte, key []byte) error
	PublishWithoutKey(ctx context.Context, value []byte) error
	PublishWithEvent(ctx context.Context, value []byte, key []byte, event chan kafka.Event) error
	PublishRaw(ctx context.Context, msg *kafka.Message, event chan kafka.Event) error
	Close()
}

type EventHandlerFunc func(event kafka.Event)

func NewClient(cfg xkafka.Config, opts ...Option) (Client, error) {
	log.Info("init kafka producer, it may take a few seconds to init the connection\n")
	// common arguments
	var kafkaConf = &kafka.ConfigMap{
		"api.version.request": "true",
		"message.max.bytes":   1000000,
		"linger.ms":           10,
		"retries":             3,
		"retry.backoff.ms":    1000,
		"acks":                "1",
	}
	if cfg.ConfigMap != nil {
		for k, v := range cfg.ConfigMap {
			_ = kafkaConf.SetKey(k, v)
		}
	}
	_ = kafkaConf.SetKey("bootstrap.servers", cfg.BootstrapServers)
	switch cfg.SecurityProtocol {
	case "plaintext":
		_ = kafkaConf.SetKey("security.protocol", cfg.SecurityProtocol)
	case "sasl_ssl":
		_ = kafkaConf.SetKey("security.protocol", cfg.SecurityProtocol)
		_ = kafkaConf.SetKey("ssl.ca.location", cfg.SslCaLocation)
		_ = kafkaConf.SetKey("sasl.username", cfg.SaslUsername)
		_ = kafkaConf.SetKey("sasl.password", cfg.SaslPassword)
		_ = kafkaConf.SetKey("sasl.mechanism", cfg.SaslMechanism)
	case "sasl_plaintext":
		_ = kafkaConf.SetKey("security.protocol", cfg.SecurityProtocol)
		_ = kafkaConf.SetKey("sasl.username", cfg.SaslUsername)
		_ = kafkaConf.SetKey("sasl.password", cfg.SaslPassword)
		_ = kafkaConf.SetKey("sasl.mechanism", cfg.SaslMechanism)
	default:
		return nil, kafka.NewError(kafka.ErrUnknownProtocol, "unknown protocol", true)
	}
	cli := &client{conf: kafkaConf}
	cli.opts = Options{
		Topic: cfg.Topics[0],
	}
	for _, o := range opts {
		o(&cli.opts)
	}
	if err := cli.initProducer(); err != nil {
		return nil, err
	}
	return cli, nil
}

type client struct {
	opts     Options
	conf     *kafka.ConfigMap
	producer *kafka.Producer
}

func (cli *client) initProducer() error {
	producer, err := kafka.NewProducer(cli.conf)
	if err != nil {
		return err
	}
	if cli.opts.EventHandler == nil {
		cli.opts.EventHandler = func(e kafka.Event) {}
	}
	async.RecoverGO(func() {
		for e := range producer.Events() {
			cli.opts.EventHandler(e)
		}
	})
	cli.producer = producer
	log.Info("init kafka producer success\n")
	return nil
}

func (cli *client) Topic() string {
	return cli.opts.Topic
}

func (cli *client) Produce(value []byte, key []byte) error {
	return cli.ProduceRaw(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &cli.opts.Topic, Partition: kafka.PartitionAny},
		Value:          value,
		Key:            key,
	}, nil)
}

func (cli *client) ProduceWithoutKey(value []byte) error {
	return cli.ProduceRaw(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &cli.opts.Topic, Partition: kafka.PartitionAny},
		Value:          value,
	}, nil)
}

func (cli *client) ProduceWithEvent(value []byte, key []byte, event chan kafka.Event) error {
	return cli.ProduceRaw(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &cli.opts.Topic, Partition: kafka.PartitionAny},
		Value:          value,
		Key:            key,
	}, event)
}

func (cli *client) ProduceRaw(msg *kafka.Message, event chan kafka.Event) error {
	return cli.producer.Produce(msg, event)
}

func (cli *client) Publish(ctx context.Context, value []byte, key []byte) error {
	return cli.PublishRaw(ctx, &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &cli.opts.Topic, Partition: kafka.PartitionAny},
		Value:          value,
		Key:            key,
	}, nil)
}

func (cli *client) PublishWithoutKey(ctx context.Context, value []byte) error {
	return cli.PublishRaw(ctx, &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &cli.opts.Topic, Partition: kafka.PartitionAny},
		Value:          value,
	}, nil)
}

func (cli *client) PublishWithEvent(ctx context.Context, value []byte, key []byte, event chan kafka.Event) error {
	return cli.PublishRaw(ctx, &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &cli.opts.Topic, Partition: kafka.PartitionAny},
		Value:          value,
		Key:            key,
	}, event)
}

func (cli *client) PublishRaw(ctx context.Context, msg *kafka.Message, event chan kafka.Event) error {
	// var span opentracing.Span
	// if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
	// 	span, _ = opentracing.StartSpanFromContext(ctx, "kafka publish")
	// 	span.SetTag("component", "kafka")
	// 	defer span.Finish()
	// 	if err := kafkatracer.Inject(span, &msg.Headers); err != nil {
	// 		return err
	// 	}
	// }
	err := cli.producer.Produce(msg, event)
	// if err != nil && span != nil {
	// 	span.LogFields(opentracinglog.Object("err", err))
	// }
	return err
}

func (cli *client) Close() {
	cli.producer.Flush(3 * 1000)
	cli.producer.Close()
}
