package http

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

type Config struct {
	Backoff          Backoff           `config:"backoff"`
	BatchPublish     bool              `config:"batch_publish"`
	BatchSize        int               `config:"batch_size"`
	CompressionLevel int               `config:"compression_level" validate:"min=0, max=9"`
	ContentType      string            `config:"content_type"`
	Headers          map[string]string `config:"headers"`
	LoadBalance      bool              `config:"loadbalance"`
	MaxRetries       int               `config:"max_retries"`
	Protocol         string            `config:"protocol"`
	Path             string            `config:"path"`
	Params           map[string]string `config:"parameters"`
	Password         string            `config:"password"`
	ProxyURL         string            `config:"proxy_url"`
	TLS              *tlscommon.Config `config:"tls"`
	Timeout          time.Duration     `config:"timeout"`
	Username         string            `config:"username"`
}

type Backoff struct {
	Init time.Duration `config:"init"`
	Max  time.Duration `config:"max"`
}

func defaultConfig() Config {
	return Config{
		Backoff: Backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		BatchPublish:     false,
		BatchSize:        2048,
		CompressionLevel: 0,
		LoadBalance:      false,
		MaxRetries:       3,
		Password:         "",
		Path:             "",
		Params:           nil,
		Protocol:         "",
		ProxyURL:         "",
		Timeout:          90 * time.Second,
		TLS:              nil,
		Username:         "",
	}
}

func (c *Config) Validate() error {
	if c.ProxyURL != "" {
		if _, err := parseProxyURL(c.ProxyURL); err != nil {
			return err
		}
	}
	return nil
}

func readConfig(cfg *config.C) (*Config, error) {
	c := defaultConfig()

	err := cfgwarn.CheckRemoved6xSettings(cfg, "port")
	if err != nil {
		return nil, err
	}

	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}

	return &c, nil
}
