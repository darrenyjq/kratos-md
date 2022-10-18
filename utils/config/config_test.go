package config

import (
	"testing"

	"codeup.aliyun.com/qimao/go-contrib/prototype/log"
	"codeup.aliyun.com/qimao/go-contrib/prototype/pkg/config/encoder/yaml"
	"codeup.aliyun.com/qimao/go-contrib/prototype/pkg/env"
)

//初始化配置
func LoadConfig(mode string) {
	strategy := NewStrategy(mode)
	source := getSource(mode)
	err := strategy.Load(source)
	if err != nil {
		log.Fatalf("load config file fail2: %s", err.Error())
	}
	return
}
func getSource(mode string) (s Source) {
	baseSource := getBaseSource()
	switch mode {
	case env.DebugMode:
		return NewLocalSources(baseSource)
	case env.TestMode, env.ReleaseMode:
		return NewRemoteSources([]RemoteSource{
			{
				ServerName: "your config name",
				BaseSource: baseSource,
			},
		},
		)
	default:
		log.Fatalf("load config error mode: %s", mode)
	}
	return
}

//列举出需要加载的配置
func getBaseSource() []BaseSource {
	return []BaseSource{
		{
			FileName: "common.yaml",
			DataId:   "common.yaml",
			Encoder:  yaml.NewEncoder(),
		},
	}
}

func NewStrategy(mode string) Loader {
	switch mode {
	case env.DebugMode:
		return NewLocalStrategy("configs/" + mode)
	case env.TestMode, env.ReleaseMode:
		return NewRemoteStrategy("configs/"+mode, WithMonitor())
	default:
		log.Fatalf("load config error mode: %s", mode)
	}
	return nil
}

//加载配置
func TestLoadConfig(t *testing.T) {
	LoadConfig(env.TestMode)
}
