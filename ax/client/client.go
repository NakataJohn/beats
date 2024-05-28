package client

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/ax/client/common"
	"github.com/elastic/beats/v7/ax/client/config"
	"github.com/elastic/beats/v7/ax/client/handler"
	bc "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec/format"
	"github.com/go-netty/go-netty/codec/frame"
)

type AXClient struct {
	Host          string        //连接地址
	Port          string        //端口
	Rtime         time.Duration //重试间隔时间，默认10秒
	BusinessFunc  func(string)  //业务方法
	Name          string        //agentID
	HeartIdleTime time.Duration //心跳超时时间，默认30秒
	log           *logp.Logger
}

//var cpath string

func (c AXClient) Connect() {

	//如果小于10秒按10秒设置
	if c.Rtime < time.Duration(10*time.Second) {
		c.Rtime = time.Duration(10 * time.Second)
	}
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>断连重试间隔时间为:", c.Rtime.Seconds(), "秒")

	//如果小于30秒按30秒设置
	if c.HeartIdleTime < time.Duration(10*time.Second) {
		c.HeartIdleTime = time.Duration(10 * time.Second)
	}
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>心跳间隔时间为：", c.HeartIdleTime.Seconds(), "秒")

	//判断文件是否存在
	/*if _oK, _err := fileExists(c.KeyFile); !_oK {
		fmt.Println("文件异常", _err)
		panic("文件不存在,请检查")
	}

	//判断文件是否存在
	if _ok, _err := fileExists(c.CertFile); !_ok {
		fmt.Println("文件异常:", _err)
		panic("文件不存在，请检查")
	}*/

	// wd := handler.WatchDogHandler{
	// 	Rtime: c.Rtime,
	// 	URI:   c.Host + ":" + c.Port,
	// }

	wd := handler.NewWatchDogHandler(c.Rtime, c.Host+":"+c.Port)

	handlers := []netty.Handler{
		frame.DelimiterCodec(102400, "$$", true),
		format.TextCodec(),
		netty.WriteIdleHandler(c.HeartIdleTime),
		handler.HeartBeatHandler{},
		handler.ConnHandler{Gotctx: func(ctx netty.HandlerContext) { contxt = ctx }},
		handler.LogicHandler{SID: c.Name, Do: c.BusinessFunc},
		wd,
	}

	clientInitializer := func(channel netty.Channel) {
		pipeline := channel.Pipeline()

		for _, handler := range handlers {
			pipeline.AddLast(handler)
		}
		//添加业务handler
		//pipeline.AddLast(handler.LogicHandler{SID: c.SID, Do: c.BusinessFunc})
	}
	wd.Watch(netty.NewBootstrap(netty.WithClientInitializer(clientInitializer)))
}

var m = config.Monitor{}

func Start(cfg *bc.C) {
	cfg.Unpack(&m)
	//转换为全路径
	var fullPath = common.DirPath(m.Path)

	var dispatcher = &Dispatcher{
		Path: fullPath,
		log:  logp.NewLogger("AXClient-dispatcher"),
	}

	var Axclient = &AXClient{
		Rtime:         m.STCP.RetryTime,
		Host:          m.STCP.Host,
		Port:          m.STCP.Port,
		Name:          m.Name,
		HeartIdleTime: m.STCP.IdleTime,
		BusinessFunc:  dispatcher.Do,
		log:           logp.NewLogger("AXClient"),
	}

	Axclient.log.Info("Starting client connect......")

	go func(c *AXClient) {
		c.Connect()
	}(Axclient)
}

var contxt netty.HandlerContext

func SendMsg(msg string) error {
	if contxt != nil {
		if contxt.Channel().IsActive() {
			contxt.Write(msg)
			return nil
		}
	}
	return errors.New("context faild")
}
