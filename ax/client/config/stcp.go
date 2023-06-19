package config

import "time"

type STCP struct {
	IdleTime  time.Duration `config:"idleTime"`
	RetryTime time.Duration `config:"retryTime"`
	Host      string        `config:"host"`
	Port      string        `config:"port"`
	CertFile  string        `config:"certFile"`
	KeyFile   string        `config:"keyFile"`
}

type Monitor struct {
	STCP STCP
	Name string
	Path string `config:"heartbeat.config.monitors.path"`
}
