package logging

import (
	"net/http"
	"strings"
	"time"
	// "kratos-md/logging/usertrack"
)

//
// type UserTracker interface {
// 	setFeature(feature usertrack.Feature)
// 	getFeature() (feature usertrack.Feature)
// 	SetUserTrackInfo(userID int64, username string, userNodeID int64)
// }
//
// var _ UserTracker = &UserTrack{}
//
// type UserTrack struct {
// 	Track usertrack.Feature
// }
//
// func (this *UserTrack) setFeature(feature usertrack.Feature) {
// 	this.Track = feature
// }
//
// func (this *UserTrack) getFeature() usertrack.Feature {
// 	return this.Track
// }
//
// func (this *UserTrack) SetUserTrackInfo(userID int64, username string, userNodeID int64) {
// 	this.Track.UserID = userID
// 	this.Track.Username = username
// 	this.Track.UserNodeID = userNodeID
// }
//
// // type bufferWriter struct {
// //	tango.ResponseWriter
// //	content []byte
// // }
// //
// // func (b *bufferWriter) Write(bs []byte) (int, error) {
// //	b.content = append(b.content, bs...)
// //	return b.ResponseWriter.Write(bs)
// // }
//
type RequestLogger interface {
	Log(access *Access)
}

type Request struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	URI    string            `json:"uri"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type Response struct {
	Status int               `json:"status"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type Access struct {
	Time time.Time `json:"time"`

	ServerID   string   `json:"server_id"`
	ServerPort string   `json:"server_port"`
	RequestID  string   `json:"request_id"`
	Request    Request  `json:"request"`
	Response   Response `json:"response"`

	Latency   string `json:"latency"`
	LatencyNs int64  `json:"latency_ns"`

	// UserTrackFeature usertrack.Feature `json:"user_track_feature"`
}

func toMapString(bean http.Header) (ret map[string]string) {
	ret = make(map[string]string)
	for key, value := range bean {
		ret[key] = strings.Join(value, "\u0020")
	}
	return ret
}
