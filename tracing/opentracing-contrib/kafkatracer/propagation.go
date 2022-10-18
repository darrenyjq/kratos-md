package kafkatracer

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type kafkaHeadersCarrier []kafka.Header

// ForeachKey conforms to the opentracing.TextMapReader interface.
func (c kafkaHeadersCarrier) ForeachKey(handler func(key, val string) error) error {
	for _, hdr := range c {
		k := hdr.Key
		v := string(hdr.Value)
		if err := handler(k, v); err != nil {
			return err
		}
	}
	return nil
}

// Set implements Set() of opentracing.TextMapWriter.
func (c *kafkaHeadersCarrier) Set(key, val string) {
	for _, hdr := range *c {
		if hdr.Key == key {
			hdr.Value = []byte(val)
			return
		}
	}
	*c = append(*c, kafka.Header{
		Key:   key,
		Value: []byte(val),
	})
}
