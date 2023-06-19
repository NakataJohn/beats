package client

import (
	"fmt"
	"testing"

	"github.com/elastic/go-ucfg"

	"github.com/elastic/beats/v7/ax/client/common"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
)

var opts = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

var jsonStr = `[{
	"check": { "request" : { "method": "GET"} },
	"enable": true,
	"hosts": "https://bgxt.bda.gov.cn/x5",
	"max_redirects": 5,
	"id": "test", 
	"ipv4": true,
	"ipv6": true,
	"mode": "any",
	"name": "无纸化办公OA系统-aa",
	"response": {"include_body": "never"},
	"schedule": "@every 60s",
	"timeout": 30,
	"type": "http",
	"ssl": { "verification_mode": "none"},
	"retry": { "times": 1, "interval": 10}
}]
`

func TestConfigLoad(t *testing.T) {

	loadFromJson()

	loadFromFile()

}

func loadFromJson() {

	result, err := common.LoadConfigsFromJson(jsonStr)

	if err != nil {
		fmt.Println(err)
	}

	for _, c := range result {

		hash, err2 := cfgfile.HashConfig(c.Config)

		if err2 != nil {
			fmt.Println(err2)
		}

		fmt.Printf("json hash =>: %v \n", hash)

	}

}

func loadFromFile() {

	file := "D:\\go-works\\src\\github.com\\elastic\\beats\\v7\\heartbeat\\monitors.d\\ok-test.yml"

	result, _ := common.LoadConfigsFromFile(file)

	for _, c := range result {

		hash, err2 := cfgfile.HashConfig(c.Config)

		if err2 != nil {
			fmt.Println(err2)
		}

		fmt.Printf("file hash =>: %v \n", hash)

	}
}
