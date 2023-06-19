package common

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-ucfg"
	byaml "github.com/elastic/go-ucfg/yaml"
	"gopkg.in/yaml.v2"
)

var opts = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

// json字符串转yaml
func JSONToYAML(j []byte) ([]byte, error) {
	var jsonObj interface{}

	err := yaml.Unmarshal(j, &jsonObj)
	if err != nil {
		fmt.Println("tool.go_转yaml失败：", err)
		return nil, err
	}
	return yaml.Marshal(jsonObj)
}

// dict格式
func DictToYAML(in interface{}) ([]byte, error) {
	return yaml.Marshal(in)
}

// dict 格式转为cfg
func LoadConfigsFromDict(in interface{}) ([]*reload.ConfigWithMeta, error) {
	b, err := DictToYAML(in)
	if err != nil {
		fmt.Println("tool.go.LoadConfigsFromDict格式转化失败：", err)
		return nil, err
	}
	return loadConfigsFromYaml(b)
}

// json字符串转cfg
func LoadConfigsFromJson(s string) ([]*reload.ConfigWithMeta, error) {

	b, err := JSONToYAML([]byte(s))

	if err != nil {
		fmt.Println("tool.go.LoadConfigsFromJson字符串转化失败：", err)
		return nil, err
	}

	return loadConfigsFromYaml(b)
}

func loadConfigsFromYaml(b []byte) ([]*reload.ConfigWithMeta, error) {
	result := []*reload.ConfigWithMeta{}
	opts = append([]ucfg.Option{
		ucfg.MetaData(ucfg.Meta{Source: ""}),
	}, opts...)

	cs, err2 := byaml.NewConfig(b, opts...)

	if err2 != nil {
		fmt.Println("tool.go.loadConfigsFromYaml解析失败：", err2)
		return nil, err2
	}
	cfg := (*config.C)(cs)

	var c []*config.C
	cfg.Unpack(&c)

	for _, _c := range c {
		result = append(result, &reload.ConfigWithMeta{Config: _c})
	}
	return result, nil
}

// 文件转cfg
func LoadConfigsFromFile(file string) ([]*reload.ConfigWithMeta, error) {
	result := []*reload.ConfigWithMeta{}

	cfgs, err := cfgfile.LoadList(file)
	if err != nil {
		fmt.Println("tool.go.LoadConfigsFromFile转化错误：", err)
		return nil, err
	}

	for _, _c := range cfgs {
		result = append(result, &reload.ConfigWithMeta{Config: _c})
	}

	return result, nil

}
