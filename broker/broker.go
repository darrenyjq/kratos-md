package broker

import (
	"context"

	_ "github.com/go-kratos/kratos/v2/log" // 初始化日志
)

type Broker interface {
	Start(ctx context.Context) error
	Stop() error
}
