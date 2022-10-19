package broker

import (
	"context"

	_ "github.com/darrenyjq/kratos-md/tracing" // 初始化分布式追踪
	_ "github.com/go-kratos/kratos/v2/log"     // 初始化日志
)

type Broker interface {
	Start(ctx context.Context) error
	Stop() error
}
