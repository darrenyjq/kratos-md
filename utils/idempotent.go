// +build !redisv8

package utils

import (
	"time"

	"codeup.aliyun.com/qimao/go-contrib/prototype/engine/xgrpc/idempotent"
	"codeup.aliyun.com/qimao/go-contrib/prototype/pkg/client/redis"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func IdempotentCheckFuncUseRedis(client redis.Client, methods map[string]struct{}) idempotent.CheckFunc {
	return func(fullMethod string, requestId string, requestTime time.Time) error {
		if _, ok := methods[fullMethod]; !ok {
			return nil
		}
		if requestId == "" {
			return status.Error(codes.InvalidArgument, "请求metadata中必须包含: x-request-id")
		}
		if requestTime.Before(time.Now().Add(-30 * time.Second)) {
			return status.Error(codes.DeadlineExceeded, "已经时间超过了最后期限，拒绝服务")
		}
		if !client.SetNX("x-req-id:"+requestId, 1, time.Minute).Val() {
			return status.Error(codes.AlreadyExists, "已经请求过了，拒绝服务")
		}
		return nil
	}
}
