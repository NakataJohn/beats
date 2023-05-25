package http

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	logSelector = "output_http"
)

func init() {
	outputs.RegisterType("http", makeHTTP)
}

func makeHTTP(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	// cfg *common.Config,
	cfg *conf.C,
) (outputs.Group, error) {
	log := logp.NewLogger(logSelector)
	log.Debug("initialize http output")

	// config := defaultConfig
	config, err := readConfig(cfg)
	if err != nil {
		outputs.Fail(err)
	}

	////unpack config that will init a http client,now it already
	//// be action in readConfig function.
	// if err := cfg.Unpack(&config); err != nil {
	// 	return outputs.Fail(err)
	// }

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	tls, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(err)
	}

	proxyURL, err := common.ParseURL(config.ProxyURL)
	if err != nil {
		return outputs.Fail(err)
	}

	if proxyURL != nil {
		log.Infof("Using proxy URL: %s", proxyURL)
	}

	params := config.Params
	if len(params) == 0 {
		params = nil
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		log.Info("Making client for host: ", host)
		hostURL, err := common.MakeURL(config.Protocol, config.Path, host, 80)
		if err != nil {
			logger.Error("Invalid host param set: %s, Error: %v", host, err)
			return outputs.Fail(err)
		}
		log.Info("Final host URL: ", hostURL)
		// fmt.Println("Final host URL: ", hostURL)
		var client outputs.NetworkClient
		client, err = newClient(clientSettings{
			URL:              hostURL,
			Proxy:            proxyURL,
			TLS:              tls,
			Username:         config.Username,
			Password:         config.Password,
			Parameters:       params,
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
			Observer:         observer,
			BatchPublish:     config.BatchPublish,
			Headers:          config.Headers,
			ContentType:      config.ContentType,
		})

		if err != nil {
			return outputs.Fail(err)
		}
		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
		clients[i] = client
	}
	return outputs.SuccessNet(config.LoadBalance, config.BatchSize, config.MaxRetries, clients)
}
