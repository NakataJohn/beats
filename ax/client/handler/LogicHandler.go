/*
 * 业务处理的handler，
 * 用于处理获取到的数据
 */

package handler

import (
	"fmt"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/go-netty/go-netty"
)

type LogicHandler struct {
	SID string       //agent ID
	Do  func(string) //业务执行的方法,
	log *logp.Logger
}

/**
 * 建立连接
 */

func (l LogicHandler) HandleActive(ctx netty.ActiveContext) {

	if l.log == nil {
		l.log = logp.NewLogger("LogicHandler")
	}

	l.log.Info(l.SID, "->", "服务连接成功", ctx.Channel().RemoteAddr())

	fmt.Println(l.SID, "->", "服务连接成功", ctx.Channel().RemoteAddr())

	init := fmt.Sprintf("{\"f\": \"%v\", \"sid\": \"%v\"}", "init", l.SID)

	ctx.Write(init)

	ctx.HandleActive()
}

/**
 * 读取数据
 */

func (l LogicHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
	if l.log == nil {
		l.log = logp.NewLogger("LogicHandler")
	}
	l.log.Debug("客户端收到消息：", message)
	if _val, ok := message.(string); ok {
		l.Do(_val)
	}

}
