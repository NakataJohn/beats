package http

import (
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/plugin"
)

var Bundle = plugin.Bundle(
	outputs.Plugin("http", makeHTTP),
)
