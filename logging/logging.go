package logging

//
import (
	"encoding/json"
	// "kratos-middleware/logging/usertrack"
	// "kratos-middleware/util"

	"github.com/Shopify/sarama"
	"github.com/go-kratos/kratos/v2/log"
)

//
// type Options struct {
// 	Debug bool
//
// 	IgnorePrefix       string
// 	IgnoreContentTypes []string
//
// 	HideRequestBodyFunc func(nethttp.Header) bool
// 	// RequestLogger       RequestLogger
// 	Logger log.Logger
// }
//
// func prepareOptions(opts []Options) Options {
// 	var opt Options
// 	if len(opts) > 0 {
// 		opt = opts[0]
// 	}
// 	return opt
// }
//
// func Logger(options ...Options) middleware.Middleware {
// 	opt := prepareOptions(options)
// 	return func(handler middleware.Handler) middleware.Handler {
// 		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
// 			var (
// 				stdreq *nethttp.Request
// 				start  = time.Now()
// 			)
// 			if tr, ok := transport.FromServerContext(ctx); ok {
// 				// 断言成HTTP的Transport可以拿到特殊信息
// 				if ht, ok := tr.(*http.Transport); ok {
// 					stdreq = ht.Request()
// 				}
// 			}
// 			if stdreq == nil ||
// 				len(opt.IgnorePrefix) > 0 && strings.HasPrefix(stdreq.URL.Path, opt.IgnorePrefix) {
// 				return handler(ctx, req)
// 			}
// 			requestID := stdreq.Header.Get("trace.id")
// 			if requestID == "" {
// 				requestID = fmt.Sprintf("%d", time.Now().UnixNano())
// 			}
// 			requestFeatures := prepareRequestFeatureMap(stdreq)
// 			requestFeatures["_request_id"] = requestID
// 			userTrack := usertrack.Parse(requestFeatures)
// 			userTrack.AccessToken = stdreq.URL.Query().Get("access_token")
// 			if userTrack.AccessToken == "" {
// 				userTrack.AccessToken = stdreq.Header.Get("")
// 			}
// 			cookieSid, _ := stdreq.Cookie("SESSIONID")
// 			if cookieSid == nil {
// 				cookieSid, _ = stdreq.Cookie("com.zto.sessionId")
// 			}
// 			if cookieSid != nil {
// 				userTrack.SessionID = cookieSid.Value
// 			}
//
// 			var requestBody string
// 			var ignoreBody = false
// 			if opt.HideRequestBodyFunc != nil {
// 				if opt.HideRequestBodyFunc(stdreq.Header) {
// 					ignoreBody = true
// 				}
// 			}
//
// 			if !ignoreBody {
// 				requestBody = util.ToJson(req)
// 			} else {
// 				requestBody = "[ignored]"
// 			}
//
// 			reply, err = handler(ctx, req)
//
// 			// Stop timer
// 			latency := time.Now().Sub(start)
//
// 			httpAccess := Access{
// 				ServerID:   serverID,
// 				ServerPort: serverPort,
// 				Time:       start,
// 				RequestID:  requestID,
// 				Request: Request{
// 					Method: stdreq.Method,
// 					Path:   stdreq.URL.Path,
// 					URI:    stdreq.URL.String(),
// 					Header: toMapString(stdreq.Header),
// 					Body:   requestBody,
// 				},
// 				Response: Response{
// 					Status: nethttp.StatusOK,
// 					Header: toMapString(nil),
// 					Body:   util.ToJson(reply),
// 				},
// 				Latency:   latency.String(),
// 				LatencyNs: int64(latency),
//
// 				UserTrackFeature: userTrack,
// 			}
// 			if err != nil {
// 				httpAccess.Response.Body = util.ToJson(err)
// 			} else {
// 				httpAccess.Response.Body = util.ToJson(reply)
// 			}
//
// 			// if stdreq.Header.Get("Content-Type") == "" {
// 			//	httpAccess.Response.Body = "[ignored]"
// 			// } else if len(opt.IgnoreContentTypes) > 0 {
// 			//	for _, tp := range opt.IgnoreContentTypes {
// 			//		if strings.HasPrefix(stdreq.Header.Get("Content-Type"), tp) {
// 			//			httpAccess.Response.Body = "[ignored]"
// 			//		}
// 			//	}
// 			// }
//
// 			if opt.RequestLogger != nil {
// 				// xgo.GoDirect(opt.RequestLogger.Log, &httpAccess)
// 				go opt.RequestLogger.Log(&httpAccess)
// 			}
//
// 			if opt.Debug {
// 				opt.Logger.Log(log.LevelDebug, "【logging】", util.ToJson(httpAccess))
// 			}
// 			return
// 		}
// 	}
// }
//
// func prepareRequestFeatureMap(r *nethttp.Request) map[string]interface{} {
// 	features := make(map[string]interface{})
// 	features["_client_ip"] = util.ClientIP(r)
//
// 	for _, headerName := range featureHeaders {
// 		if h := r.Header.Get(headerName); h != "" {
// 			features[headerName] = h
// 		} else {
// 			ck, _ := r.Cookie(strings.TrimPrefix(headerName, "X-"))
// 			if ck != nil {
// 				features[headerName] = ck.Value
// 			}
// 		}
// 	}
//
// 	return features
// }
//
// var featureHeaders = []string{
// 	// "X-Client-Ip",
// 	"X-Remote-Port",
// 	"X-Client-Mac",
//
// 	"X-Device-Wifi-Ssid",
// 	"X-Device-Wifi-Bssid",
//
// 	"X-Device-Id",
// 	"X-Device-Name",
// 	"X-Device-Model",
// 	"X-Device-Locale",
// 	"X-Device-Location",
// 	"X-Device-Private-Ip-Addr",
//
// 	"X-Platform-Name",
// 	"X-Platform-Version",
//
// 	"X-Bundle-Id",
// 	"X-App-Version",
// 	"X-App-Bundle-Id",
// 	"X-App-Bundle-Name",
//
// 	"X-Canvas-Fingerprint",
//
// 	"User-Agent",
// 	"X-User-Agent-Plugins",
// 	"X-Screen-Resolution",
// 	"X-Webrtc-Addrs",
//
// 	"X-Access-Token",
//
// 	"X-Sign-Timestamp",
// 	"X-Sign",
// }
//
type httpAccessEncoder struct {
	*Access
	encoded []byte
	err     error
}

func (e *httpAccessEncoder) ensureEncoded() {
	if e.encoded == nil && e.err == nil {
		e.encoded, e.err = json.Marshal(e)
	}
}

func (e *httpAccessEncoder) Length() int {
	e.ensureEncoded()
	return len(e.encoded)
}

func (e *httpAccessEncoder) Encode() ([]byte, error) {
	e.ensureEncoded()
	return e.encoded, e.err
}

type authRequestEncoder struct {
	// auth.Request
	encoded []byte
	err     error
}

func (e *authRequestEncoder) ensureEncoded() {
	if e.encoded == nil && e.err == nil {
		e.encoded, e.err = json.Marshal(e)
	}
}

func (e *authRequestEncoder) Length() int {
	e.ensureEncoded()
	return len(e.encoded)
}

func (e *authRequestEncoder) Encode() ([]byte, error) {
	e.ensureEncoded()
	return e.encoded, e.err
}

var _ sarama.StdLogger = &KafkaLogger{}

type KafkaLogger struct {
	*log.Helper
}

func NewKafkaLogger(lHelper *log.Helper) *KafkaLogger {
	return &KafkaLogger{lHelper}
}

func (l KafkaLogger) Print(v ...interface{}) {
	l.Helper.Info(v...)
	return
}
func (l KafkaLogger) Printf(format string, v ...interface{}) {
	l.Helper.Infof(format, v...)
	return
}
func (l KafkaLogger) Println(v ...interface{}) {
	l.Helper.Info(v...)
	return
}
