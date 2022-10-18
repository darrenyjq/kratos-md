package logging

import (
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/panjf2000/ants"
)

var (
	serverID        string
	serverPort      string
	HttpAccessTopic string
)

var (
	asyncProducer        sarama.AsyncProducer
	HttpAccessWorkerPool *ants.PoolWithFunc
)

func InitKafka(brokerList []string, topic string, logger log.Logger) {
	log.Info(brokerList, topic)
	sarama.Logger = NewKafkaLogger(log.NewHelper(logger))
	// sarama.Logger = sarama.DebugLogger
	NewKafkaProducer(brokerList, topic)
	log.Info(brokerList, topic)
	HttpAccessWorkerPool, _ = ants.NewPoolWithFunc(60, func(i interface{}) {
		acc, is := i.(*Access)
		if is {
			httpAccess(acc)
		}
	})

}

func NewKafkaProducer(brokers []string, topic string) {
	config := sarama.NewConfig()
	config.ClientID = "pixiu"

	config.Producer.RequiredAcks = sarama.WaitForLocal     // Only wait for the leader to ack
	config.Producer.Compression = sarama.CompressionSnappy // Compress messages
	config.Producer.Flush.Frequency = 3 * time.Second      // Flush batches every 3s
	config.Producer.Flush.Messages = 1000
	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		log.Error("err", err, "Sec NewAsyncProducer error")
		return
	}

	// We will just log to STDOUT if we're not able to produce messages.
	// Note: messages will only be returned here after all retry attempts are exhausted.
	go func() {
		for err1 := range producer.Errors() {
			log.Error(err1)
		}
	}()
	HttpAccessTopic = topic
	asyncProducer = producer
	return
}

type HttpRequestLogger struct{}

func (HttpRequestLogger) Log(access *Access) {
	HttpAccess(access)
}

func HttpAccess(access *Access) {
	if HttpAccessWorkerPool == nil {
		return
	}
	if HttpAccessWorkerPool.Free() == 0 {
		fmt.Println("message abandoned, http access pool free=0 running=", HttpAccessWorkerPool.Running())
		return
	}
	log.Info(access)
	log.Info(333)
	HttpAccessWorkerPool.Invoke(access)
}

func httpAccess(access *Access) {
	if asyncProducer == nil {
		return
	}

	// 忽略公共文件存储系统请求
	if strings.HasPrefix(access.Request.Path, "/dfs/public") {
		return
	}

	asyncProducer.Input() <- &sarama.ProducerMessage{
		Topic: HttpAccessTopic,
		Key:   sarama.StringEncoder(access.RequestID),
		Value: &httpAccessEncoder{Access: access},
	}
}
