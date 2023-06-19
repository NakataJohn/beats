/*
 * 心跳handler，
 * 当读超时时，向服务端发送心跳消息
 */

package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/go-netty/go-netty"
	"github.com/go-ping/ping"
)

const HeartBeat = "Heartbeat"

type HeartBeatHandler struct {
	log *logp.Logger
}

func (h HeartBeatHandler) HandleEvent(ctx netty.EventContext, event netty.Event) {
	ip := strings.Split(ctx.Channel().RemoteAddr(), ":")[0]

	if h.log == nil {
		h.log = logp.NewLogger("HeartBeatHandler")
	}
	h.log.Debug("Heartbeat HandleEvent", event)
	// fmt.Println("Heartbeat HandleEvent", event)

	//只有当连接存活时出现写空闲则发送心跳
	//netty的IsActive方法无法准确判断网络异常,使用发送icmp包测试网络可达状态
	if _, _ok := event.(netty.WriteIdleEvent); _ok {
		pinger, _ := ping.NewPinger(ip)
		pinger.Count = 3
		pinger.Timeout = time.Second * 3
		pinger.SetPrivileged(true)
		err := pinger.Run()
		if err != nil {
			fmt.Println(ip, "网络异常:", err)
			h.log.Error("ping server failed")
		} else {
			fmt.Println(time.Now().Format("[2006-01-02 15:04:05] Send heartbeat_message to server"), HeartBeat)
			ctx.Channel().Write(HeartBeat)
		}
	}
}
