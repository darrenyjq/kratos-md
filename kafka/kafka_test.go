package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/panjf2000/ants"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/darrenyjq/kratos-md/broker/xkafka"
)

func TestProducer(t *testing.T) {
	NewAdAdxKafkaClient()
	srv := NewAdxProducer(KafkaAdAdxClient)

	go func() {
		for i := 0; i < 100; i++ {
			err := srv.ProduceWithoutKey(ReportBigData{Msg: "test", Id: 4})
			if err != nil {
				log.Error(err, "失败")
			}
			time.Sleep(time.Second * 3)
		}
	}()

	time.Sleep(time.Hour)
}

func TestConsumer(t *testing.T) {

	pool, err := ants.NewPoolWithFunc(3, func(i interface{}) {
		message := i.(*kafka.Message)
		log.Info(message)
	})
	if err != nil {
		log.Fatalf("%+v", err)
	}
	defer pool.Release()
	cfg := xkafka.Config{
		Topics:           []string{"pixiu_log"},
		GroupId:          "pixiu_log",
		BootstrapServers: "139.196.93.141:9093,106.14.13.204:9093,139.224.23.125:9093",
		SecurityProtocol: "sasl_ssl",
		SaslMechanism:    "PLAIN",
		SslCaLocation:    "./assets/ca-cert.pem",
		SaslUsername:     "businessadvuser",
		SaslPassword:     "m7feiSh8nG9uAnMs",
	}
	bk := xkafka.NewBroker(cfg, xkafka.Handler(func(message *kafka.Message) {
		err := pool.Invoke(message)
		if err != nil {
			log.Errorf("%+v", err)
		}
	}))

	err = bk.Start(context.TODO())
	if err != nil {
		log.Error(err)
	}
}

var KafkaAdAdxClient Client

// NewAdAdxKafkaClient 统计日志上报到kafka
func NewAdAdxKafkaClient() {
	var cfg = Config{
		BootstrapServers: "139.196.93.141:9093,106.14.13.204:9093,139.224.23.125:9093",
		SecurityProtocol: "sasl_ssl",
		SaslMechanism:    "PLAIN",
		ConfigMap:        map[string]kafka.ConfigValue{"compression.type": "lz4"},
		SslCaLocation:    "./assets/ca-cert.pem",
		SaslUsername:     "businessadvuser",
		SaslPassword:     "m7feiSh8nG9uAnMs",
		Topic:            "pixiu_log",
	}

	kafkaClient, err := NewClient(cfg, EventHandler(func(e kafka.Event) {
		switch ev := e.(type) {
		case *kafka.Message:
			m := ev
			if m.TopicPartition.Error != nil {
				log.Error("发送失败", m.TopicPartition.Error)
			}
			// 打印发送成功后的消息
			log.Infof("发送消息到topic %s [分区%d] at offset %v\n", *m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
			return
		default:
			log.Error("Ignored event", ev)
		}
	}))
	if err != nil {
		log.Errorf("kafkaClient err：%+v", err)
	}
	KafkaAdAdxClient = kafkaClient
	return
}

type AdxProducer interface {
	ProduceWithoutKey(msgData ReportBigData) error
}

func NewAdxProducer(kafkaClient Client) AdxProducer {
	return &adxProducer{
		Client: kafkaClient,
	}
}

type adxProducer struct {
	Client Client
}

type ReportBigData struct {
	Msg string `json:"msg"`
	Id  int64  `json:"id"`
}

func (repo *adxProducer) ProduceWithoutKey(msgData ReportBigData) error {
	msgByte, _ := json.Marshal(msgData)
	log.Info(msgByte)
	err := repo.Client.PublishWithoutKey(context.TODO(), msgByte)
	if err != nil {
		log.Error("推送事件信息到kafka失败:", err)
		return err
	}
	return nil
}
