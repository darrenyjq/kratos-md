package broker

import (
	"context"

	_ "codeup.aliyun.com/qimao/go-contrib/prototype/log"     //初始化日志
	_ "codeup.aliyun.com/qimao/go-contrib/prototype/tracing" //初始化分布式追踪
)

type Broker interface {
	Start(ctx context.Context) error
	Stop() error
}
