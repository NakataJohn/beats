package client

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/ax/client/common"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/elastic-agent-libs/logp"
)

var mutex sync.RWMutex

type Dispatcher struct {
	Path string
	log  *logp.Logger
}

func (d *Dispatcher) Do(msg string) {

	// wg := &sync.WaitGroup{}
	if !strings.Contains(msg, "action") {
		d.log.Errorf("消息异常")
		fmt.Println("消息异常：", msg)
		return
	}

	fmt.Println("====> 收到消息：", msg)

	_action, _err := d.action(msg)

	if _err != nil {
		fmt.Println(_err)
		return
	}

	switch _action.Action {
	case "start":
		// 检查数据
		// 1、加入监控
		// 2、创建配置文件
		startConfigs, err := common.LoadConfigsFromDict(_action.Data)
		if err != nil {
			fmt.Println(err)
		} else {
			for _, startConfig := range startConfigs {
				cfgfile.MonitorList().StartRunner(startConfig)
			}
		}
		//path := cpath //"D:/go-works/src/github.com/elastic/beats/v7/heartbeat/monitors.d"
		for _, data := range _action.Data {
			id := data["id"].(string)
			//file := cpath + "/" + id + ".yml"
			common.CreateFile(d.getFileName(id), [1]interface{}{data})
		}

	case "stop":
		//检查数据
		// 1、获取配置文件
		// 2、停止监控
		// 3、删除配置文件
		//path := "D:/go-works/src/github.com/elastic/beats/v7/heartbeat/monitors.d"
		for _, data := range _action.Data {
			id := data["id"].(string)
			//file := cpath + "/" + id + ".yml"

			if ok, ferr := common.FileExists(d.getFileName(id)); !ok {
				fmt.Println(ferr)
				continue
			}

			stopConfigs, err := common.LoadConfigsFromFile(d.getFileName(id))
			if err != nil {
				fmt.Println(err)
			} else {
				for _, stopstopConfig := range stopConfigs {
					cfgfile.MonitorList().StopRunner(stopstopConfig)
				}
			}

			common.Delconfigfile(d.getFileName(id))

		}
	// by john add asyncjob
	// TODO 异步监测任务
	case "async":
		// 检查数据
		// 加入监控
		asyncjobConfigs, err := common.LoadConfigsFromDict(_action.Data)
		if err != nil {
			fmt.Println(err)
		} else {
			go func() {
				for _, asyncjobConfig := range asyncjobConfigs {
					cfgfile.MonitorList().StartRunner(asyncjobConfig)
					time.Sleep(50 * time.Second)
					mutex.Lock()
					defer mutex.Unlock()
					cfgfile.MonitorList().StopRunner(asyncjobConfig)
				}
			}()
		}

	case "change":
		// 检查数据
		// 1、获取配置文件
		// 2、删除监控
		// 3、加入监控
		// 4、更新配置文件
		//path := "D:/go-works/src/github.com/elastic/beats/v7/heartbeat/monitors.d"
		for _, data := range _action.Data {
			id := data["id"].(string)
			//path := cpath + "/" + id + ".yaml"
			stopConfigs, err := common.LoadConfigsFromFile(d.getFileName(id))
			if err != nil {
				fmt.Println(err)
			} else {
				for _, stopstopConfig := range stopConfigs {
					cfgfile.MonitorList().StopRunner(stopstopConfig)
				}
			}
		}
		startConfigs, err := common.LoadConfigsFromDict(_action.Data)
		if err != nil {
			fmt.Println(err)
		} else {
			for _, startConfig := range startConfigs {
				cfgfile.MonitorList().StartRunner(startConfig)
			}
		}

		for _, data := range _action.Data {
			id := data["id"].(string)
			//file := cpath + "/" + id + ".yml"
			common.CreateFile(d.getFileName(id), [1]interface{}{data})
		}

	default:
	}
}

// 获取action操作
func (d *Dispatcher) action(msg string) (actionData, error) {
	//m := make(map[string]interface{})

	m := actionData{}

	err := json.Unmarshal([]byte(msg), &m)

	if err != nil {
		return m, err
	}

	return m, nil
}

type actionData struct {
	Action string
	Data   []map[string]interface{}
}

func (d *Dispatcher) getFileName(id string) string {

	return filepath.FromSlash(strings.Join([]string{d.Path, "/", id, ".yml"}, ""))
}
