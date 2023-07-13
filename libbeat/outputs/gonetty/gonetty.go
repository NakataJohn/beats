package gonetty

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	logSelector = "output_gonetty"
	networkTCP  = "tcp4"
)

func init() {
	outputs.RegisterType("gonetty", makeGoNetty)
}

func makeGoNetty(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *config.C) (outputs.Group, error) {
	log := logp.NewLogger(logSelector)
	log.Debug("initialize gonetty output")

	// hosts, err := outputs.ReadHostList(cfg)
	// if err != nil {
	// 	return outputs.Fail(err)
	// }

	config, err := readConfig(cfg)
	if err != nil {
		log.Error(err)
	}
	enc, err := codec.CreateEncoder(beat, config.Codec)
	if err != nil {
		log.Error(err)
	}

	client, err := newTcpOut(beat.Beat, config, observer, enc)
	if err != nil {
		return outputs.Fail(err)
	}
	return outputs.Success(config.BatchSize, config.MaxRetries, client)

}
