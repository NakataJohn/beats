/*
 * 监听连接是否中断，中断连接进行重连
 *
 */

package handler

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/go-netty/go-netty"
	//netty tls
)

type WatchDogHandler struct {
	mu         sync.Mutex
	bootstrap  netty.Bootstrap
	Rtime      time.Duration
	URI        string
	connecting bool
	log        *logp.Logger //= logp.NewLogger("AXClient")
}

func NewWatchDogHandler(rtime time.Duration, url string) *WatchDogHandler {
	return &WatchDogHandler{
		Rtime: rtime,
		URI:   url,
	}
}

func (w *WatchDogHandler) Watch(bootstrap netty.Bootstrap) {

	w.bootstrap = bootstrap
	if w.log == nil {
		w.log = logp.NewLogger("WatchDogHandler")
	}
	w.log.Info("watch dog")

	timer := time.NewTimer(w.Rtime)

	for con_time := range timer.C {
		w.log.Debug("重连时间：", con_time)
		w.connect()
		timer.Reset(w.Rtime)
	}
}

func (w *WatchDogHandler) connect() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.connecting {
		return
	}

	w.connecting = true

	//toption := tls.WithOptions(&tls.Options{CertFile: w.CertFile, KeyFile: w.KeyFile})

	ch, _err := w.bootstrap.Connect(w.URI)

	w.connecting = false

	if _err != nil {
		return
	}

	select {
	case <-ch.Context().Done():
		fmt.Println("ch.Context().Done()")
	case <-w.bootstrap.Context().Done():
		fmt.Println(w.bootstrap.Context().Done())
	}
}

/*
 * 断开连接
 */
func (w *WatchDogHandler) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
	fmt.Println("inactive:", ctx.Channel().RemoteAddr(), ex)
	w.log.Error("inactive:", ctx.Channel().RemoteAddr(), ex)
	if ex.Error() == "EOF" {
		fmt.Println("服务端已关闭")
		ctx.Channel().Close(ex)
	}

}

func (w *WatchDogHandler) HandleException(ctx netty.ExceptionContext, ex netty.Exception) {
	fmt.Println("HandleException", ex)
	w.log.Error("Exception:", ctx.Channel().RemoteAddr(), ex)
	if ex.Error() == "EOF" {
		fmt.Println("服务端已关闭")
		ctx.Channel().Close(ex)
	}
}
