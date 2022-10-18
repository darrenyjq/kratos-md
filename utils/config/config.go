package config

import (
	"fmt"
	"net/http"
	"sync"

	"codeup.aliyun.com/qimao/go-contrib/prototype/log"
	"codeup.aliyun.com/qimao/go-contrib/prototype/pkg/config"
	"codeup.aliyun.com/qimao/go-contrib/prototype/pkg/config/encoder"
	"codeup.aliyun.com/qimao/go-contrib/prototype/pkg/config/encoder/yaml"
	"codeup.aliyun.com/qimao/go-contrib/prototype/pkg/config/source"
	"codeup.aliyun.com/qimao/go-contrib/prototype/pkg/config/source/file"
	"codeup.aliyun.com/qimao/go-contrib/prototype/pkg/config/source/nacos"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/ksuid"
)

const (
	nacosPort  = 8848
	rotateTime = "24h"
	maxAge     = 15
	timeoutMs  = 3000
)

var metricsOnce sync.Once

type Loader interface {
	Load(Source) error
}

//Source 数据源
type Source interface {
	Read() error
}

type remoteSources struct {
	remote []RemoteSource
	remoteStrategy
}

func NewRemoteSources(remote []RemoteSource) Source {
	return &remoteSources{remote: remote}
}

type localSources struct {
	local []BaseSource
	localStrategy
}

func NewLocalSources(local []BaseSource) Source {
	return &localSources{local: local}
}

type BaseSource struct {
	FileName string          //本地文件名(主要用于本地加载)
	DataId   string          //nacos dataId(用于远程加载)
	Encoder  encoder.Encoder //编码方式
}
type RemoteSource struct {
	ServerName string
	BaseSource []BaseSource
}

type localStrategy struct {
	prefixPath string //目录前缀
}

func NewLocalStrategy(path string) *localStrategy {
	return &localStrategy{prefixPath: path}
}

//Load 本地调试配置加载
func (l *localStrategy) Load(source Source) error {
	s, ok := source.(*localSources)
	if !ok {
		log.Fatal("localStrategy source need type(localSources)")
	}
	if l == nil {
		log.Fatal("localStrategy is nil")
	}
	s.localStrategy = *l
	return s.Read()
}

//远程配置加载（暂指nacos）
type remoteStrategy struct {
	nacosConfPath string //nacos配置目录
	option        *remoteOption
}
type remoteOption struct {
	monitor bool // 是否开启监控
}

func NewRemoteStrategy(path string, opt ...Option) *remoteStrategy {
	r := &remoteStrategy{
		nacosConfPath: path,
		option:        &remoteOption{},
	}
	for _, o := range opt {
		o(r.option)
	}
	return r
}

//Load 远程配置加载
func (r *remoteStrategy) Load(source Source) error {
	//加载本地nacos配置，用于后续加载远程指定endpoint namespace group的配置
	loadLocalNacosConfig(r.nacosConfPath)
	s, ok := source.(*remoteSources)
	if !ok {
		log.Fatal("remoteStrategy source need type(remoteSources)")
	}
	if r == nil {
		log.Fatal("remoteStrategy is nil")
	}
	s.remoteStrategy = *r
	if err := s.Read(); err != nil {
		return err
	}
	if r.monitor() {
		startMetrics()
	}
	return nil
}
func (r *remoteStrategy) monitor() bool {
	return r.option.monitor
}

func (l *localSources) Read() error {
	sourceSli := make([]source.Source, 0, len(l.local))
	for _, s := range l.local {
		sourceSli = append(sourceSli, file.NewSource(file.WithPath(l.prefixPath+"/"+s.FileName), source.WithEncoder(s.Encoder)))
	}
	return config.Load(sourceSli...)
}
func (s *remoteSources) Read() error {
	for _, cf := range s.remote {
		nacosBaseConf, err := getNacosBaseConfig(cf.ServerName)
		if err != nil {
			return err
		}
		clientConfig := getClientConfig(nacosBaseConf)
		serverConfig := getServerConfig(nacosBaseConf.EndPoint)
		sourceSli := cf.getSource(clientConfig, serverConfig, s.monitor(), nacosBaseConf.Group)
		if err := config.Load(sourceSli...); err != nil {
			return err
		}
	}
	return nil
}

func newSource(dataID, group string, clientConf constant.ClientConfig, serverConf []constant.ServerConfig, encoder encoder.Encoder, monitor bool) source.Source {
	opts := []source.Option{
		nacos.DataId(dataID),
		nacos.WithClientConfig(clientConf),
		nacos.WithServerConfigs(serverConf...),
		source.WithEncoder(encoder),
		nacos.WithMonitor(monitor),
	}
	if len(group) != 0 {
		opts = append(opts, nacos.Group(group))
	}
	return nacos.NewSource(opts...)
}

func getClientConfig(cf nacosBaseConfig) (client constant.ClientConfig) {
	randStr := ksuid.New().String()
	logDir := fmt.Sprintf("/tmp/nacos_%s/log", randStr)
	cacheDir := fmt.Sprintf("/tmp/nacos_%s/cache", randStr)
	return constant.ClientConfig{
		Endpoint:            cf.EndPoint + ":8848 ",
		NamespaceId:         cf.Namespace,
		TimeoutMs:           timeoutMs, //请求超时时间
		NotLoadCacheAtStart: false,
		LogDir:              logDir,
		CacheDir:            cacheDir,
		RotateTime:          rotateTime,
		MaxAge:              maxAge,
		AccessKey:           cf.AccessKey,
		SecretKey:           cf.SecretKey,
	}
}
func (r *RemoteSource) getSource(clientConfig constant.ClientConfig, serverConfig []constant.ServerConfig, monitor bool, group string) []source.Source {
	sourceSli := make([]source.Source, 0, len(r.BaseSource))
	for _, s := range r.BaseSource {
		sourceSli = append(sourceSli, newSource(s.DataId, group, clientConfig, serverConfig, s.Encoder, monitor))
	}
	return sourceSli
}

func getServerConfig(ipAddr string) []constant.ServerConfig {
	return []constant.ServerConfig{
		{IpAddr: ipAddr, Port: nacosPort},
	}
}

//监听端口 方便prometheus收集metrics
func startMetrics() {
	//只执行一次，防止多次调用
	metricsOnce.Do(func() {
		go func() {
			//处理异常，不影响主程序
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("nacos metrics panic: %v", r)
				}
			}()
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe(":8088", nil); err != nil {
				log.Errorf("nacos metrics error: %v", err.Error())
			}
		}()
	})
}

//加载本地nacos配置
func loadLocalNacosConfig(path string) {
	strategy := &localStrategy{prefixPath: path}
	baseSource := getNacosSource()
	err := strategy.Load(&localSources{local: baseSource})
	if err != nil {
		log.Fatalf("load nacos base config file: %s", err.Error())
	}
	return
}
func getNacosSource() []BaseSource {
	return []BaseSource{
		{
			FileName: "nacos.yaml",
			Encoder:  yaml.NewEncoder(),
		},
	}
}

type nacosBaseConfig struct {
	EndPoint  string `json:"endpoint"`   //nacos 远程地址
	Namespace string `json:"namespace"`  //命名空间
	Group     string `json:"group"`      //group
	AccessKey string `json:"access_key"` //ak
	SecretKey string `json:"secret_key"` //sk
}

func getNacosBaseConfig(serverName string) (nacosBaseConfig, error) {
	var baseConfig nacosBaseConfig
	err := config.Get(serverName).Scan(&baseConfig)
	return baseConfig, err
}
