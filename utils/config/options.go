package config

type Option func(*remoteOption)

func WithMonitor() Option {
	return func(r *remoteOption) {
		r.monitor = true
	}
}
