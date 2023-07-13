package gonetty

import (
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/pkg/errors"
)

type gonettyConfig struct {
	Host         string  `config:"host"`
	Port         string  `config:"port"`
	Backoff      Backoff `config:"backoff"`
	BatchPublish bool    `config:"batch_publish"`

	BufferSize    int          `config:"buffer_size"`
	SSLEnable     bool         `config:"ssl.enable"`
	SSLCertPath   string       `config:"ssl.cert_path"`
	SSLKeyPath    string       `config:"ssl.key_path"`
	BatchSize     int          `config:"batch_size"`
	MaxRetries    int          `config:"max_retries"`
	lineDelimiter string       `config:"line_delimiter"`
	Codec         codec.Config `config:"codec"`
}

type Backoff struct {
	Init time.Duration `config:"init"`
	Max  time.Duration `config:"max"`
}

// var defaultConfig = gonettyConfig{
// 	BufferSize:    1 << 15,
// 	WritevEnable:  true,
// 	lineDelimiter: "\n",
// }

func defaultConfig() gonettyConfig {
	return gonettyConfig{
		Backoff: Backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		BatchPublish:  false,
		BatchSize:     2048,
		MaxRetries:    3,
		lineDelimiter: "\n",
		SSLEnable:     false,
	}
}

func (c *gonettyConfig) Validate() error {
	if c.SSLEnable {
		if _, err := os.Stat(c.SSLCertPath); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("certificate %s not found", c.SSLCertPath))
		}
		if _, err := os.Stat(c.SSLKeyPath); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("key %s not found", c.SSLKeyPath))
		}
	}
	return nil
}

func readConfig(cfg *config.C) (*gonettyConfig, error) {
	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}
	return &c, nil
}
